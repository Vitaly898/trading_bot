package live

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Reconciler runs in the same event loop as candle handling.
type Reconciler interface {
	Reconcile(context.Context, []string) error
}

type algoOrder struct {
	AlgoID        int64  `json:"algoId"`
	ClientAlgoID  string `json:"clientAlgoId"`
	Symbol        string `json:"symbol"`
	OrderType     string `json:"orderType"`
	Side          string `json:"side"`
	PositionSide  string `json:"positionSide"`
	ClosePosition bool   `json:"closePosition"`
	TriggerPrice  string `json:"triggerPrice"`
}

func (e *TestnetExecutor) openProtection(ctx context.Context, sym string) ([]algoOrder, error) {
	var orders []algoOrder
	if err := e.signedGet(ctx, "/fapi/v1/openAlgoOrders", map[string]string{"symbol": sym}, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}
func ownedProtection(o algoOrder, p *Position) bool {
	return strings.HasPrefix(o.ClientAlgoID, "tb-") || (p != nil && (o.AlgoID == p.StopOrderID || o.AlgoID == p.TakeOrderID))
}
func (e *TestnetExecutor) cleanupProtection(ctx context.Context, sym string, p *Position) error {
	orders, err := e.openProtection(ctx, sym)
	if err != nil {
		return err
	}
	for _, o := range orders {
		if o.Symbol != sym || !ownedProtection(o, p) {
			continue
		}
		if err := e.signedDelete(ctx, "/fapi/v1/algoOrder", map[string]string{"algoId": strconv.FormatInt(o.AlgoID, 10)}); err != nil {
			return err
		}
	}
	return nil
}
func (e *TestnetExecutor) restoreProtection(ctx context.Context, p *Position, orders []algoOrder) error {
	side := "SELL"
	if p.Dir < 0 {
		side = "BUY"
	}
	stopFound, takeFound := false, false
	for _, o := range orders {
		if o.Symbol != p.Symbol || o.Side != side || !o.ClosePosition || (o.PositionSide != "" && o.PositionSide != "BOTH") {
			continue
		}
		if o.OrderType != "STOP_MARKET" && o.OrderType != "TAKE_PROFIT_MARKET" {
			continue
		}
		if o.AlgoID <= 0 {
			return fmt.Errorf("invalid protection order ID: %s", p.Symbol)
		}
		trigger, err := strconv.ParseFloat(o.TriggerPrice, 64)
		if err != nil || trigger <= 0 || math.IsNaN(trigger) || math.IsInf(trigger, 0) {
			return fmt.Errorf("invalid protection trigger: %s", p.Symbol)
		}
		if o.OrderType == "STOP_MARKET" && !stopFound {
			p.Stop, p.StopOrderID = trigger, o.AlgoID
			stopFound = true
		}
		if o.OrderType == "TAKE_PROFIT_MARKET" && !takeFound {
			p.Take, p.TakeOrderID = trigger, o.AlgoID
			takeFound = true
		}
	}
	if !stopFound {
		if p.Stop <= 0 {
			return fmt.Errorf("%s: open position has no known stop", p.Symbol)
		}
		id, err := e.placeAlgo(ctx, p.Symbol, side, "STOP_MARKET", p.Stop)
		if err != nil {
			return err
		}
		p.StopOrderID = id
	}
	if !takeFound && p.Take > 0 {
		id, err := e.placeAlgo(ctx, p.Symbol, side, "TAKE_PROFIT_MARKET", p.Take)
		if err != nil {
			return err
		}
		p.TakeOrderID = id
	}
	return nil
}

// Reconcile reads authoritative positions and protections. A failed response does
// not partially replace the cache or erase known metadata.
func (e *TestnetExecutor) Reconcile(ctx context.Context, symbols []string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	next := copyPositions(e.positions)
	for _, sym := range symbols {
		old := e.positions[sym]
		var pending orderFill
		if old != nil && (old.Unconfirmed || old.PendingClose != "") {
			id := old.ClientOrderID
			if old.PendingClose != "" {
				id = old.PendingClose
			}
			if err := e.signedGet(ctx, "/fapi/v1/order", map[string]string{"symbol": sym, "origClientOrderId": id}, &pending); err != nil {
				return fmt.Errorf("reconcile pending %s: %w", sym, err)
			}
		}
		var rows []struct {
			Symbol       string `json:"symbol"`
			PositionSide string `json:"positionSide"`
			PositionAmt  string `json:"positionAmt"`
			EntryPrice   string `json:"entryPrice"`
		}
		if err := e.signedGet(ctx, "/fapi/v2/positionRisk", map[string]string{"symbol": sym}, &rows); err != nil {
			return fmt.Errorf("reconcile %s: %w", sym, err)
		}
		if len(rows) != 1 || rows[0].Symbol != sym || (rows[0].PositionSide != "" && rows[0].PositionSide != "BOTH") {
			return fmt.Errorf("%s: expected one-way position response", sym)
		}
		amt, err := strconv.ParseFloat(rows[0].PositionAmt, 64)
		if err != nil || math.IsNaN(amt) || math.IsInf(amt, 0) {
			return fmt.Errorf("%s: invalid position amount", sym)
		}
		orders, err := e.openProtection(ctx, sym)
		if err != nil {
			return fmt.Errorf("reconcile protection %s: %w", sym, err)
		}
		if amt == 0 {
			if old != nil && (old.Unconfirmed || old.PendingClose != "") && !terminalOrder(pending.Status) {
				return fmt.Errorf("%s: order %s still pending", sym, pending.Status)
			}
			for _, o := range orders {
				if o.Symbol == sym && ownedProtection(o, old) {
					if err := e.signedDelete(ctx, "/fapi/v1/algoOrder", map[string]string{"algoId": strconv.FormatInt(o.AlgoID, 10)}); err != nil {
						return err
					}
				}
			}
			delete(next, sym)
			continue
		}
		ep, err := strconv.ParseFloat(rows[0].EntryPrice, 64)
		if err != nil || ep <= 0 || math.IsNaN(ep) || math.IsInf(ep, 0) {
			return fmt.Errorf("%s: invalid entry price", sym)
		}
		dir := 1
		if amt < 0 {
			dir = -1
		}
		p := &Position{Symbol: sym, Dir: dir, Qty: math.Abs(amt), EntryPrice: ep, EntryTime: time.Now().UTC()}
		if old != nil && old.Dir == dir {
			*p = *old
			p.Qty, p.EntryPrice = math.Abs(amt), ep
		}
		if err := e.restoreProtection(ctx, p, orders); err != nil {
			return fmt.Errorf("protect %s: %w", sym, err)
		}
		if old != nil && (old.Unconfirmed || old.PendingClose != "") {
			if !terminalOrder(pending.Status) {
				return fmt.Errorf("%s: execution remains pending (%s)", sym, pending.Status)
			}
			if old.Unconfirmed {
				p.EntryOrderID = pending.OrderID
			}
		}
		p.Unconfirmed, p.PendingClose = false, ""
		next[sym] = p
	}
	e.positions = next
	return nil
}
func terminalOrder(status string) bool {
	switch status {
	case "FILLED", "CANCELED", "EXPIRED", "EXPIRED_IN_MATCH", "REJECTED":
		return true
	}
	return false
}
