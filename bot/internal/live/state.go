package live

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"time"

	"bot/internal/trading"
)

const stateVersion = 1

type stateSnapshot struct {
	Version        int                              `json:"version"`
	Mode           string                           `json:"mode"`
	TF             string                           `json:"tf"`
	Symbols        []string                         `json:"symbols"`
	StrategyKey    string                           `json:"strategy_key"`
	UpdatedAt      time.Time                        `json:"updated_at"`
	Equity         *float64                         `json:"equity,omitempty"`
	Positions      map[string]*Position             `json:"positions"`
	LastClosed     map[string]time.Time             `json:"last_closed"`
	EntryBars      map[string]time.Time             `json:"entry_bars"`
	StrategyStates map[string]trading.PositionState `json:"strategy_states"`
}

func (r *Runner) LoadState() error {
	if r.cfg.StatePath == "" {
		return nil
	}
	buf, err := os.ReadFile(r.cfg.StatePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var snap stateSnapshot
	if err := json.Unmarshal(buf, &snap); err != nil {
		return fmt.Errorf("read state: %w", err)
	}
	if snap.Version != stateVersion {
		return fmt.Errorf("unsupported state version %d: legacy snapshots cannot recover equity/strategy ownership; use a separate -state path", snap.Version)
	}
	wantSymbols := slices.Clone(r.cfg.Symbols)
	gotSymbols := slices.Clone(snap.Symbols)
	slices.Sort(wantSymbols)
	slices.Sort(gotSymbols)
	if snap.Mode != r.exec.Name() || snap.TF != r.cfg.TF || snap.StrategyKey != r.cfg.StrategyKey || !slices.Equal(wantSymbols, gotSymbols) {
		return fmt.Errorf("state mode, symbols, timeframe or strategy differs from configuration; use a separate -state path")
	}
	for sym, p := range snap.Positions {
		if p == nil || p.Symbol != sym || !slices.Contains(wantSymbols, sym) || validateEntry(p.Dir, p.Qty, p.EntryPrice) != nil || p.EntryTime.IsZero() {
			return fmt.Errorf("invalid saved position: %s", sym)
		}
		if p.Stop < 0 || p.Take < 0 || math.IsNaN(p.Stop) || math.IsNaN(p.Take) || math.IsInf(p.Stop, 0) || math.IsInf(p.Take, 0) {
			return fmt.Errorf("invalid saved protection: %s", sym)
		}
		state, ok := snap.StrategyStates[sym]
		if !ok || state.Dir != p.Dir || state.BarsIn < 0 {
			return fmt.Errorf("missing saved strategy state: %s", sym)
		}
	}
	switch e := r.exec.(type) {
	case *LocalExecutor:
		if snap.Equity == nil || math.IsNaN(*snap.Equity) || math.IsInf(*snap.Equity, 0) {
			return fmt.Errorf("missing or invalid local equity")
		}
		e.mu.Lock()
		e.equity = *snap.Equity
		e.positions = copyPositions(snap.Positions)
		e.mu.Unlock()
	case *TestnetExecutor:
		e.mu.Lock()
		e.positions = copyPositions(snap.Positions)
		e.mu.Unlock()
	default:
		return fmt.Errorf("executor does not support state restoration")
	}
	if snap.LastClosed != nil {
		r.lastClosed = snap.LastClosed
	}
	if snap.EntryBars != nil {
		r.entryBars = snap.EntryBars
	}
	if snap.StrategyStates != nil {
		r.positionStates = snap.StrategyStates
	}
	return nil
}
func (r *Runner) SaveState() error { return r.saveState(r.exec.Positions()) }

// positions may be an executor's pre-submission snapshot; do not lock it again.
func (r *Runner) saveState(positions map[string]*Position) (err error) {
	defer func() { r.persistenceBlocked = err != nil }()
	if r.cfg.StatePath == "" {
		return nil
	}
	snap := stateSnapshot{Version: stateVersion, Mode: r.exec.Name(), TF: r.cfg.TF, Symbols: r.cfg.Symbols, StrategyKey: r.cfg.StrategyKey, UpdatedAt: time.Now().UTC(), Positions: positions, LastClosed: r.lastClosed, EntryBars: r.entryBars, StrategyStates: r.positionStates}
	if e, ok := r.exec.(*LocalExecutor); ok {
		e.mu.RLock()
		v := e.equity
		snap.Equity = &v
		snap.Positions = copyPositions(e.positions)
		e.mu.RUnlock()
	}
	buf, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return writeStateAtomic(r.cfg.StatePath, buf)
}
func writeStateAtomic(path string, buf []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".paper-state-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(buf); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
