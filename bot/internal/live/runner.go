package live

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"bot/internal/accounting"
	"bot/internal/candle"
	"bot/internal/trading"
)

type RunnerConfig struct {
	Symbols      []string
	TF           string
	RiskPct      float64
	MaxPositions int
	MaxTotalRisk float64
	StatePath    string
	StrategyKey  string // Includes strategy name and parameters; guards incompatible restarts.
}

// Runner owns the strategy state. Handle, Reconcile and persistence run serially.
type Runner struct {
	log                *slog.Logger
	cfg                RunnerConfig
	exec               Executor
	strategies         map[string]trading.Strategy
	lastClose          map[string]float64
	lastClosed         map[string]time.Time
	entryBars          map[string]time.Time
	positionStates     map[string]trading.PositionState
	knownDirections    map[string]int
	syncBlocked        bool
	persistenceBlocked bool
}

func NewRunner(log *slog.Logger, cfg RunnerConfig, exec Executor) *Runner {
	r := &Runner{log: log, cfg: cfg, exec: exec, strategies: map[string]trading.Strategy{}, lastClose: map[string]float64{}, lastClosed: map[string]time.Time{}, entryBars: map[string]time.Time{}, positionStates: map[string]trading.PositionState{}, knownDirections: map[string]int{}}
	if e, ok := exec.(*TestnetExecutor); ok {
		e.beforeSubmit = r.saveState
	}
	return r
}
func (r *Runner) RegisterStrategy(sym string, s trading.Strategy) error {
	r.strategies[sym] = s
	return r.syncPosition(sym, "restore")
}

// WarmupStrategy rebuilds indicators and restores trading state afterwards.
// Downtime bars count towards time stops; no historical market orders are sent.
func (r *Runner) WarmupStrategy(sym string, s trading.Strategy, cs []candle.Candle) error {
	if len(cs) == 0 {
		return fmt.Errorf("%s: no closed warmup candles", sym)
	}
	if cs[len(cs)-1].Time.Before(r.lastClosed[sym]) {
		return fmt.Errorf("%s: warmup history is older than saved watermark", sym)
	}
	for i, c := range cs {
		if i > 0 && !c.Time.After(cs[i-1].Time) {
			return fmt.Errorf("%s: warmup candles are not strictly ordered", sym)
		}
	}
	for _, c := range cs {
		s.OnCandle(c)
	}
	state, ok := r.positionStates[sym]
	if ok && state.Dir != 0 {
		for _, c := range cs {
			if c.Time.After(r.lastClosed[sym]) && c.Time.After(r.entryBars[sym]) {
				state.BarsIn++
			}
		}
		r.positionStates[sym] = state
	}
	if err := r.RegisterStrategy(sym, s); err != nil {
		return err
	}
	r.lastClose[sym] = cs[len(cs)-1].C
	r.MarkClosed(sym, cs[len(cs)-1].Time)
	return nil
}
func (r *Runner) MarkClosed(sym string, t time.Time) {
	if t.After(r.lastClosed[sym]) {
		r.lastClosed[sym] = t
	}
}

func (r *Runner) syncPosition(sym, reason string) error {
	s := r.strategies[sym]
	if s == nil {
		return nil
	}
	p := r.exec.Positions()[sym]
	if p != nil && p.Unconfirmed {
		return nil
	}
	dir := 0
	if p != nil {
		dir = p.Dir
	}
	if dir == 0 && r.knownDirections[sym] == 0 {
		delete(r.positionStates, sym)
		delete(r.entryBars, sym)
		return nil
	}
	if dir == r.knownDirections[sym] {
		return nil
	}
	if dir == 0 {
		if pa, ok := s.(trading.PositionAware); ok {
			pa.OnPositionChange(0, reason)
		}
		delete(r.positionStates, sym)
		delete(r.entryBars, sym)
	} else if restorer, ok := s.(trading.PositionRestorer); ok {
		state, exists := r.positionStates[sym]
		if !exists || state.Dir != dir {
			return fmt.Errorf("%s: position has no matching saved strategy state", sym)
		}
		if err := restorer.RestorePosition(state); err != nil {
			return fmt.Errorf("restore %s: %w", sym, err)
		}
	} else {
		if pa, ok := s.(trading.PositionAware); ok {
			pa.OnPositionChange(dir, reason)
		}
		r.positionStates[sym] = trading.PositionState{Dir: dir}
	}
	r.knownDirections[sym] = dir
	return nil
}
func (r *Runner) Reconcile(ctx context.Context) error {
	if reconciler, ok := r.exec.(Reconciler); ok {
		if err := reconciler.Reconcile(ctx, r.cfg.Symbols); err != nil {
			r.syncBlocked = true
			r.checkpoint()
			return err
		}
	}
	for sym := range r.strategies {
		if err := r.syncPosition(sym, "exchange"); err != nil {
			r.syncBlocked = true
			r.checkpoint()
			return err
		}
	}
	r.syncBlocked = false
	if len(r.strategies) == 0 {
		return nil
	}
	return r.SaveState()
}
func (r *Runner) executionError(ctx context.Context, sym string, err error) {
	r.log.Error("ошибка исполнения", "sym", sym, "err", err)
	if _, ok := r.exec.(Reconciler); ok {
		r.syncBlocked = true
		if syncErr := r.Reconcile(ctx); syncErr != nil {
			r.log.Error("сверка после ошибки", "err", syncErr)
		}
	}
}
func (r *Runner) Handle(ctx context.Context, ev KlineEvent) {
	sym := ev.Symbol
	s := r.strategies[sym]
	if s == nil || !ev.Candle.Time.After(r.lastClosed[sym]) {
		return
	}
	if r.syncBlocked {
		return
	} // Preserve candles and pending entry intent until reconciliation succeeds.
	r.lastClose[sym] = ev.Candle.C
	if le, ok := r.exec.(*LocalExecutor); ok {
		if p := le.Positions()[sym]; p != nil && ev.Candle.Time.After(r.entryBars[sym]) {
			price, reason := 0.0, ""
			switch {
			case p.Stop > 0 && ((p.Dir > 0 && ev.Candle.L <= p.Stop) || (p.Dir < 0 && ev.Candle.H >= p.Stop)):
				price, reason = p.Stop, "stop"
			case p.Take > 0 && ((p.Dir > 0 && ev.Candle.H >= p.Take) || (p.Dir < 0 && ev.Candle.L <= p.Take)):
				price, reason = p.Take, "take"
			}
			if reason != "" {
				if _, err := le.CloseAt(sym, price, reason); err != nil {
					r.executionError(ctx, sym, err)
					return
				}
				if err := r.syncPosition(sym, reason); err != nil {
					r.log.Error("position state", "err", err)
					return
				}
				r.checkpoint()
				// A closed candle still advances indicators and evaluates the next signal.
			}
		}
	}
	if !ev.IsClosed {
		return
	}
	r.MarkClosed(sym, ev.Candle.Time)
	defer r.checkpoint()
	dir, stop, take := s.OnCandle(ev.Candle)
	if restorer, ok := s.(trading.PositionRestorer); ok && r.knownDirections[sym] != 0 {
		r.positionStates[sym] = restorer.PositionState()
	}
	r.log.Info("свеча закрыта", "sym", sym, "close", ev.Candle.C, "signal", dir, "stop", stop, "take", take)
	p := r.exec.Positions()[sym]
	cur := 0
	if p != nil {
		cur = p.Dir
	}
	if dir == cur {
		return
	}
	if p != nil {
		_, err := r.exec.Close(ctx, sym, ev.Candle.C)
		if syncErr := r.syncPosition(sym, "signal"); syncErr != nil {
			r.log.Error("position state", "err", syncErr)
			return
		}
		if err != nil {
			r.executionError(ctx, sym, err)
			return
		}
	}
	if dir == 0 {
		return
	}
	if r.persistenceBlocked {
		r.log.Warn("вход пропущен: состояние не сохранено", "sym", sym)
		return
	}
	positions := r.exec.Positions()
	if r.cfg.MaxPositions > 0 && len(positions) >= r.cfg.MaxPositions {
		r.log.Warn("вход пропущен: лимит позиций", "sym", sym)
		return
	}
	equity, err := r.exec.Equity(ctx)
	if err != nil {
		r.log.Error("equity", "err", err)
		return
	}
	dist := accounting.StopDistance(ev.Candle.C, stop)
	qty := accounting.Quantity(equity, r.cfg.RiskPct, dist)
	if err := validateEntry(dir, qty, ev.Candle.C); err != nil {
		r.log.Error("sizing", "err", err)
		return
	}
	if r.cfg.MaxTotalRisk > 0 {
		openRisk := 0.0
		for _, p := range positions {
			d := accounting.StopDistance(p.EntryPrice, p.Stop)
			openRisk += p.Qty * d
		}
		if openRisk+qty*dist > r.cfg.MaxTotalRisk*equity {
			r.log.Warn("вход пропущен: бюджет риска", "sym", sym)
			return
		}
	}
	state := trading.PositionState{Dir: dir}
	if restorer, ok := s.(trading.PositionRestorer); ok {
		state = restorer.EntryState(dir)
	}
	r.positionStates[sym] = state
	r.entryBars[sym] = ev.Candle.Time
	if _, err := r.exec.Open(ctx, sym, dir, qty, stop, take, ev.Candle.C); err != nil {
		r.executionError(ctx, sym, err)
		if r.exec.Positions()[sym] == nil {
			delete(r.positionStates, sym)
			delete(r.entryBars, sym)
		}
		return
	}
	if pa, ok := s.(trading.PositionAware); ok {
		pa.OnPositionChange(dir, "")
	}
	r.knownDirections[sym] = dir
	if restorer, ok := s.(trading.PositionRestorer); ok {
		r.positionStates[sym] = restorer.PositionState()
	}
	if dp, ok := s.(trading.DiagnosticsProvider); ok {
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
		r.log.Info("индикаторы при входе", attrs...)
	}
}
func (r *Runner) checkpoint() {
	if err := r.SaveState(); err != nil {
		r.log.Error("состояние не сохранено", "err", err)
	}
}
func absf(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
