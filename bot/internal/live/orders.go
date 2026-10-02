package live

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

type exchangeError struct {
	HTTPStatus int    `json:"-"`
	Code       int    `json:"code"`
	Message    string `json:"msg"`
}

func (e *exchangeError) Error() string {
	return fmt.Sprintf("exchange HTTP %d code %d: %s", e.HTTPStatus, e.Code, e.Message)
}

type confirmationError struct{ error }

func (e *confirmationError) Unwrap() error { return e.error }

func definitelyRejected(err error) bool {
	var confirmation *confirmationError
	if errors.As(err, &confirmation) {
		return false
	}
	var e *exchangeError
	return errors.As(err, &e) && e.HTTPStatus >= 400 && e.HTTPStatus < 500 && e.Code != 0 && e.Code != -1006 && e.Code != -1007
}
func clientOrderID() (string, error) {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return "tb-" + hex.EncodeToString(buf[:]), nil
}
func validateEntry(dir int, qty, price float64) error {
	if (dir != 1 && dir != -1) || qty <= 0 || price <= 0 || math.IsNaN(qty) || math.IsInf(qty, 0) || math.IsNaN(price) || math.IsInf(price, 0) {
		return fmt.Errorf("invalid entry direction, quantity or price")
	}
	return nil
}
func copyPositions(src map[string]*Position) map[string]*Position {
	out := make(map[string]*Position, len(src))
	for sym, p := range src {
		if p != nil {
			v := *p
			out[sym] = &v
		}
	}
	return out
}

type orderFill struct {
	OrderID     int64  `json:"orderId"`
	Status      string `json:"status"`
	AvgPrice    string `json:"avgPrice"`
	ExecutedQty string `json:"executedQty"`
}

func (f orderFill) values() (price, qty float64, err error) {
	if f.OrderID <= 0 {
		return 0, 0, fmt.Errorf("missing confirmed order ID")
	}
	price, err = strconv.ParseFloat(f.AvgPrice, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid fill price: %w", err)
	}
	qty, err = strconv.ParseFloat(f.ExecutedQty, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid executed quantity: %w", err)
	}
	if err := validateEntry(1, qty, price); err != nil {
		return 0, 0, err
	}
	return price, qty, nil
}
func (e *TestnetExecutor) market(ctx context.Context, sym, side, qty, id string, reduce bool) (result orderFill, resultErr error) {
	submitted := false
	defer func() {
		if submitted && resultErr != nil {
			resultErr = &confirmationError{resultErr}
		}
	}()
	params := map[string]string{"symbol": sym, "side": side, "type": "MARKET", "quantity": qty, "newOrderRespType": "RESULT", "newClientOrderId": id}
	if reduce {
		params["reduceOnly"] = "true"
	}
	resp, err := e.signedPost(ctx, "/fapi/v1/order", params, nil)
	if err != nil {
		return orderFill{}, err
	}
	submitted = true
	buf, err := json.Marshal(resp)
	if err != nil {
		return orderFill{}, err
	}
	var fill orderFill
	if err := json.Unmarshal(buf, &fill); err != nil {
		return orderFill{}, err
	}
	for attempt := 0; attempt < 5; attempt++ {
		if fill.Status == "FILLED" {
			if _, _, err := fill.values(); err == nil {
				return fill, nil
			}
		}
		if attempt > 0 {
			if err := waitContext(ctx, e.pollInterval); err != nil {
				return fill, err
			}
		}
		if err := e.signedGet(ctx, "/fapi/v1/order", map[string]string{"symbol": sym, "origClientOrderId": id}, &fill); err != nil {
			return fill, err
		}
	}
	return fill, fmt.Errorf("market execution not confirmed: %s %s", sym, id)
}
func waitContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (e *TestnetExecutor) Open(ctx context.Context, sym string, dir int, qty, stop, take, refPrice float64) (float64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.positions[sym] != nil {
		return 0, fmt.Errorf("position already exists: %s", sym)
	}
	if err := validateEntry(dir, qty, refPrice); err != nil {
		return 0, err
	}
	qtyStr := e.fmtQty(sym, qty)
	if qtyStr == "" {
		return 0, fmt.Errorf("%s: quantity below step size", sym)
	}
	for _, level := range []float64{stop, take} {
		if level < 0 || math.IsNaN(level) || math.IsInf(level, 0) {
			return 0, fmt.Errorf("invalid protection level")
		}
	}
	if stop > 0 {
		stop, _ = strconv.ParseFloat(e.fmtPrice(sym, stop), 64)
		if stop <= 0 {
			return 0, fmt.Errorf("stop below tick size")
		}
	}
	if take > 0 {
		take, _ = strconv.ParseFloat(e.fmtPrice(sym, take), 64)
		if take <= 0 {
			return 0, fmt.Errorf("take below tick size")
		}
	}
	id, err := clientOrderID()
	if err != nil {
		return 0, err
	}
	rounded, _ := strconv.ParseFloat(qtyStr, 64)
	p := &Position{Symbol: sym, Dir: dir, Qty: rounded, EntryPrice: refPrice, Stop: stop, Take: take, EntryTime: time.Now().UTC(), ClientOrderID: id, Unconfirmed: true}
	e.positions[sym] = p // Persist intent and client ID BEFORE the external side effect.
	if e.beforeSubmit != nil {
		if err := e.beforeSubmit(copyPositions(e.positions)); err != nil {
			delete(e.positions, sym)
			return 0, fmt.Errorf("save entry intent: %w", err)
		}
	}
	side := "BUY"
	if dir < 0 {
		side = "SELL"
	}
	fill, err := e.market(ctx, sym, side, qtyStr, id, false)
	if err != nil {
		if definitelyRejected(err) {
			delete(e.positions, sym)
		}
		return 0, fmt.Errorf("entry execution: %w", err)
	}
	p.EntryPrice, p.Qty, err = fill.values()
	if err != nil {
		return 0, err
	}
	p.EntryOrderID, p.Unconfirmed = fill.OrderID, false
	closeSide := "SELL"
	if dir < 0 {
		closeSide = "BUY"
	}
	if stop > 0 {
		p.StopOrderID, err = e.placeAlgo(ctx, sym, closeSide, "STOP_MARKET", stop)
	}
	if err == nil && take > 0 {
		p.TakeOrderID, err = e.placeAlgo(ctx, sym, closeSide, "TAKE_PROFIT_MARKET", take)
	}
	if err != nil {
		protectionErr := err
		if _, closeErr := e.closePosition(ctx, sym); closeErr != nil {
			return p.EntryPrice, fmt.Errorf("protection failed: %v; emergency close failed: %w", protectionErr, closeErr)
		}
		return p.EntryPrice, fmt.Errorf("protection failed, position closed: %w", protectionErr)
	}
	e.log.Info("открытие (testnet)", "sym", sym, "qty", p.Qty, "fill", p.EntryPrice, "order_id", p.EntryOrderID)
	return p.EntryPrice, nil
}
func (e *TestnetExecutor) placeAlgo(ctx context.Context, symbol, side, kind string, trigger float64) (int64, error) {
	id, err := clientOrderID()
	if err != nil {
		return 0, err
	}
	resp, err := e.signedPost(ctx, "/fapi/v1/algoOrder", map[string]string{"symbol": symbol, "side": side, "type": kind, "algoType": "CONDITIONAL", "triggerPrice": e.fmtPrice(symbol, trigger), "closePosition": "true", "workingType": "CONTRACT_PRICE", "clientAlgoId": id}, nil)
	if err != nil {
		return 0, err
	}
	buf, err := json.Marshal(resp)
	if err != nil {
		return 0, err
	}
	var result struct {
		AlgoID int64 `json:"algoId"`
	}
	if err := json.Unmarshal(buf, &result); err != nil {
		return 0, err
	}
	if result.AlgoID <= 0 {
		return 0, fmt.Errorf("missing algo order ID")
	}
	return result.AlgoID, nil
}
func (e *TestnetExecutor) Close(ctx context.Context, sym string, refPrice float64) (float64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.closePosition(ctx, sym)
}
func (e *TestnetExecutor) closePosition(ctx context.Context, sym string) (float64, error) {
	p := e.positions[sym]
	if p == nil {
		return 0, fmt.Errorf("no position: %s", sym)
	}
	if p.PendingClose != "" {
		return 0, fmt.Errorf("previous close requires reconciliation: %s", sym)
	}
	id, err := clientOrderID()
	if err != nil {
		return 0, err
	}
	p.PendingClose = id
	if e.beforeSubmit != nil {
		if err := e.beforeSubmit(copyPositions(e.positions)); err != nil {
			p.PendingClose = ""
			return 0, fmt.Errorf("save close intent: %w", err)
		}
	}
	side := "SELL"
	if p.Dir < 0 {
		side = "BUY"
	}
	fill, err := e.market(ctx, sym, side, e.fmtQty(sym, p.Qty), id, true)
	if err != nil {
		if definitelyRejected(err) {
			p.PendingClose = ""
		}
		return 0, fmt.Errorf("close execution: %w", err)
	}
	price, qty, err := fill.values()
	if err != nil {
		return 0, err
	}
	p.PendingClose = ""
	if qty+1e-9 < p.Qty {
		p.Qty -= qty
		return price, fmt.Errorf("partial close: %s remaining %g", sym, p.Qty)
	}
	// Protection remains active until the market close is confirmed.
	delete(e.positions, sym)
	if err := e.cleanupProtection(ctx, sym, p); err != nil {
		return price, err
	}
	e.log.Info("закрытие (testnet)", "sym", sym, "fill", price, "order_id", fill.OrderID)
	return price, nil
}
