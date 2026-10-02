package live

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"bot/internal/accounting"
)

// ---------- Local: виртуальное исполнение по реальным ценам ----------

type LocalExecutor struct {
	mu        sync.RWMutex
	log       *slog.Logger
	equity    float64
	feePct    float64
	slipPct   float64
	positions map[string]*Position
}

func NewLocalExecutor(log *slog.Logger, startEquity, feePct, slipPct float64) *LocalExecutor {
	return &LocalExecutor{
		log: log, equity: startEquity, feePct: feePct, slipPct: slipPct,
		positions: map[string]*Position{},
	}
}

func (e *LocalExecutor) Name() string { return "local" }

func (e *LocalExecutor) Equity(context.Context) (float64, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.equity, nil
}

func (e *LocalExecutor) Positions() map[string]*Position {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return copyPositions(e.positions)
}

func (e *LocalExecutor) Open(_ context.Context, sym string, dir int, qty, stop, take, refPrice float64) (float64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.positions[sym] != nil {
		return 0, fmt.Errorf("position already exists: %s", sym)
	}
	if err := validateEntry(dir, qty, refPrice); err != nil {
		return 0, err
	}
	fill := accounting.Slippage(refPrice, dir, e.slipPct)
	fee := accounting.Fee(fill, qty, e.feePct)
	e.equity -= fee
	e.positions[sym] = &Position{
		Symbol: sym, Dir: dir, Qty: qty, EntryPrice: fill,
		Stop: stop, Take: take, EntryTime: time.Now().UTC(),
	}
	e.log.Info("🟢 ОТКРЫТИЕ", "sym", sym, "dir", dir, "qty", round4(qty),
		"fill", round2(fill), "stop", round2(stop), "take", round2(take), "fee", round2(fee))
	return fill, nil
}

func (e *LocalExecutor) Close(_ context.Context, sym string, refPrice float64) (float64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	pos := e.positions[sym]
	if pos == nil {
		return 0, fmt.Errorf("нет позиции %s", sym)
	}
	fill := accounting.Slippage(refPrice, -pos.Dir, e.slipPct)
	fee := accounting.Fee(fill, pos.Qty, e.feePct)
	pnl := accounting.Gross(pos.Dir, pos.EntryPrice, fill, pos.Qty) - fee
	e.equity += pnl
	delete(e.positions, sym)
	e.log.Info("🔴 ЗАКРЫТИЕ", "sym", sym, "fill", round2(fill),
		"pnl", round2(pnl), "equity", round2(e.equity))
	return fill, nil
}

// CloseAt — закрытие по конкретной цене (стоп/тейк в local-режиме).
func (e *LocalExecutor) CloseAt(sym string, price float64, reason string) (float64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	pos := e.positions[sym]
	if pos == nil {
		return 0, fmt.Errorf("нет позиции %s", sym)
	}
	price = accounting.Slippage(price, -pos.Dir, e.slipPct)
	fee := accounting.Fee(price, pos.Qty, e.feePct)
	pnl := accounting.Gross(pos.Dir, pos.EntryPrice, price, pos.Qty) - fee
	e.equity += pnl
	delete(e.positions, sym)
	e.log.Info(fmt.Sprintf("🔴 ЗАКРЫТИЕ (%s)", reason), "sym", sym,
		"fill", round2(price), "pnl", round2(pnl), "equity", round2(e.equity))
	return price, nil
}
