package live

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sort"
	"time"

	"bot/internal/backtest"
)

// RunnerConfig — параметры раннера.
type RunnerConfig struct {
	Symbols      []string
	TF           string
	RiskPct      float64 // риск на сделку (доля equity)
	MaxPositions int
	MaxTotalRisk float64
	StatePath    string // JSON-снапшот состояния (рестарт-безопасность)
}

// Runner — live-цикл: события свечей → стратегии → исполнение с риск-лимитами.
type Runner struct {
	log  *slog.Logger
	cfg  RunnerConfig
	exec Executor

	strategies map[string]backtest.Strategy
	lastClose  map[string]float64
	lastClosed map[string]time.Time // дедупликация закрытых свечей (WS + reconcile)
}

func NewRunner(log *slog.Logger, cfg RunnerConfig, exec Executor) *Runner {
	return &Runner{
		log: log, cfg: cfg, exec: exec,
		strategies: map[string]backtest.Strategy{},
		lastClose:  map[string]float64{},
		lastClosed: map[string]time.Time{},
	}
}

// RegisterStrategy — стратегия на символ (после прогрева!).
func (r *Runner) RegisterStrategy(sym string, s backtest.Strategy) {
	r.strategies[sym] = s
}

// MarkClosed — пометить свечу обработанной (для дедупликации после прогрева).
func (r *Runner) MarkClosed(sym string, t time.Time) {
	r.lastClosed[sym] = t
}

// Handle — одно событие свечи из потока.
func (r *Runner) Handle(ctx context.Context, ev KlineEvent) {
	sym := ev.Symbol
	r.lastClose[sym] = ev.Candle.C
	strat := r.strategies[sym]
	if strat == nil {
		return
	}

	// тики: обновления живой свечи (на закрытии — отдельная логика ниже)
	r.log.Debug("тик", "sym", sym, "close", ev.Candle.C, "closed", ev.IsClosed)

	// 1. Интрабар-проверка стопов/тейков (только local-режим: в testnet их держит биржа)
	if le, ok := r.exec.(*LocalExecutor); ok {
		if pos := le.Positions()[sym]; pos != nil {
			if pos.Stop > 0 {
				if pos.Dir > 0 && ev.Candle.L <= pos.Stop {
					r.closeLocal(ctx, le, sym, pos.Stop, "STOP")
					return
				}
				if pos.Dir < 0 && ev.Candle.H >= pos.Stop {
					r.closeLocal(ctx, le, sym, pos.Stop, "STOP")
					return
				}
			}
			if pos := le.Positions()[sym]; pos != nil && pos.Take > 0 {
				if pos.Dir > 0 && ev.Candle.H >= pos.Take {
					r.closeLocal(ctx, le, sym, pos.Take, "TAKE")
					return
				}
				if pos.Dir < 0 && ev.Candle.L <= pos.Take {
					r.closeLocal(ctx, le, sym, pos.Take, "TAKE")
					return
				}
			}
		}
	}

	// 2. Сигналы стратегии — только на закрытии свечи (паритет с бэктестом)
	if !ev.IsClosed {
		return
	}
	// дедупликация: WS и reconcile могут прислать одну свечу дважды
	if !ev.Candle.Time.After(r.lastClosed[sym]) {
		return
	}
	r.lastClosed[sym] = ev.Candle.Time
	dir, stopPrice, takePrice := strat.OnCandle(ev.Candle)
	r.log.Info("свеча закрыта", "sym", sym, "close", ev.Candle.C,
		"signal", dir, "stop", stopPrice, "take", takePrice)

	pos := r.exec.Positions()[sym]
	curDir := 0
	if pos != nil {
		curDir = pos.Dir
	}
	if dir == curDir {
		r.saveState()
		return
	}

	// 3. Закрытие текущей
	if pos != nil {
		if _, err := r.exec.Close(ctx, sym, ev.Candle.C); err != nil {
			r.log.Error("ошибка закрытия", "sym", sym, "err", err)
			return
		}
		if pa, ok := strat.(backtest.PositionAware); ok {
			pa.OnPositionChange(0, "signal")
		}
	}

	// 4. Открытие новой с риск-лимитами
	if dir != 0 {
		if r.cfg.MaxPositions > 0 && len(r.exec.Positions()) >= r.cfg.MaxPositions {
			r.log.Warn("вход пропущен: лимит позиций", "sym", sym)
			r.saveState()
			return
		}
		equity, err := r.exec.Equity(ctx)
		if err != nil {
			r.log.Error("equity", "err", err)
			return
		}
		stopDist := ev.Candle.C * 0.02
		if stopPrice > 0 {
			if d := absf(ev.Candle.C - stopPrice); d > 0 {
				stopDist = d
			}
		}
		qty := equity * r.cfg.RiskPct / stopDist
		if r.cfg.MaxTotalRisk > 0 {
			openRisk := 0.0
			for _, p := range r.exec.Positions() {
				openRisk += p.Qty * absf(p.EntryPrice-p.Stop)
			}
			if openRisk+qty*stopDist > r.cfg.MaxTotalRisk*equity {
				r.log.Warn("вход пропущен: бюджет риска", "sym", sym)
				r.saveState()
				return
			}
		}
		if _, err := r.exec.Open(ctx, sym, dir, qty, stopPrice, takePrice, ev.Candle.C); err != nil {
			r.log.Error("ошибка открытия", "sym", sym, "err", err)
			return
		}
		// контекст входа: все значения индикаторов на момент открытия
		if dp, ok := strat.(backtest.DiagnosticsProvider); ok {
			d := dp.Diagnostics()
			keys := make([]string, 0, len(d))
			for k := range d {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			attrs := make([]any, 0, len(d)*2)
			for _, k := range keys {
				attrs = append(attrs, k, d[k])
			}
			r.log.Info("📊 индикаторы при входе", attrs...)
		}
		if pa, ok := strat.(backtest.PositionAware); ok {
			pa.OnPositionChange(dir, "")
		}
	}
	r.saveState()
}

func (r *Runner) closeLocal(ctx context.Context, le *LocalExecutor, sym string, price float64, reason string) {
	if _, err := le.CloseAt(sym, price, reason); err != nil {
		r.log.Error("closeLocal", "sym", sym, "err", err)
		return
	}
	if pa, ok := r.strategies[sym].(backtest.PositionAware); ok {
		pa.OnPositionChange(0, reason)
	}
	r.saveState()
}

type stateSnapshot struct {
	Mode      string               `json:"mode"`
	UpdatedAt time.Time            `json:"updated_at"`
	Positions map[string]*Position `json:"positions"`
}

func (r *Runner) saveState() {
	if r.cfg.StatePath == "" {
		return
	}
	snap := stateSnapshot{
		Mode: r.exec.Name(), UpdatedAt: time.Now().UTC(),
		Positions: r.exec.Positions(),
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(r.cfg.StatePath, data, 0o644)
}

func absf(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
