package live

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"bot/internal/backtest"
	"bot/internal/candle"
)

func quietLog() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
func event(bar int, low float64, closed bool) KlineEvent {
	return KlineEvent{Symbol: "TEST", IsClosed: closed, Candle: candle.Candle{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(bar) * time.Hour), O: 100, H: 100, L: low, C: 100, V: 1}}
}

type change struct {
	dir    int
	reason string
}
type recordingStrategy struct {
	calls   int
	signals []int
	changes []change
	state   backtest.PositionState
}

func (s *recordingStrategy) OnCandle(c candle.Candle) (int, float64, float64) {
	i := s.calls
	s.calls++
	if s.state.Dir != 0 {
		s.state.BarsIn++
	}
	dir := 0
	if i < len(s.signals) {
		dir = s.signals[i]
	}
	return dir, 90, 120
}
func (s *recordingStrategy) OnPositionChange(dir int, reason string) {
	s.changes = append(s.changes, change{dir, reason})
	s.state = s.EntryState(dir)
}
func (s *recordingStrategy) PositionState() backtest.PositionState { return s.state }
func (s *recordingStrategy) EntryState(dir int) backtest.PositionState {
	owner := ""
	if dir != 0 {
		owner = "mr"
	}
	return backtest.PositionState{Dir: dir, Owner: owner}
}
func (s *recordingStrategy) RestorePosition(p backtest.PositionState) error { s.state = p; return nil }
func runnerConfig(path string) RunnerConfig {
	return RunnerConfig{Symbols: []string{"TEST"}, TF: "1h", RiskPct: 0.01, StatePath: path, StrategyKey: "test"}
}
func assertNear(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %.12f want %.12f", got, want)
	}
}
func TestRunnerIgnoresStaleProtectionAndProcessesStopCandle(t *testing.T) {
	ctx := context.Background()
	e := NewLocalExecutor(quietLog(), 1000, 0, 0)
	r := NewRunner(quietLog(), runnerConfig(""), e)
	s := &recordingStrategy{signals: []int{1, 0}}
	if err := r.RegisterStrategy("TEST", s); err != nil {
		t.Fatal(err)
	}
	r.Handle(ctx, event(0, 100, true))
	r.Handle(ctx, event(0, 50, true))
	r.Handle(ctx, event(0, 50, false))
	r.Handle(ctx, event(-1, 50, true))
	if s.calls != 1 || len(e.Positions()) != 1 {
		t.Fatal("stale candle affected position or indicators")
	}
	r.Handle(ctx, event(1, 85, true))
	if s.calls != 2 || len(e.Positions()) != 0 {
		t.Fatal("stop candle did not advance strategy")
	}
	if len(s.changes) != 2 || s.changes[1] != (change{0, "stop"}) {
		t.Fatalf("changes: %+v", s.changes)
	}
	assertNear(t, mustEquity(t, e), 990)
}
func TestRunnerAllowsReentryAfterStopAndIgnoresDuplicate(t *testing.T) {
	e := NewLocalExecutor(quietLog(), 1000, 0, 0)
	r := NewRunner(quietLog(), runnerConfig(""), e)
	s := &recordingStrategy{signals: []int{1, 1}}
	if err := r.RegisterStrategy("TEST", s); err != nil {
		t.Fatal(err)
	}
	r.Handle(context.Background(), event(0, 100, true))
	r.Handle(context.Background(), event(1, 85, true))
	r.Handle(context.Background(), event(1, 85, true))
	if s.calls != 2 || len(s.changes) != 3 || len(e.Positions()) != 1 {
		t.Fatalf("reentry: calls=%d changes=%v", s.calls, s.changes)
	}
}
func mustEquity(t *testing.T, e Executor) float64 {
	t.Helper()
	v, err := e.Equity(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func TestLocalProtectionUsesSlippage(t *testing.T) {
	e := NewLocalExecutor(quietLog(), 1000, 0.001, 0.01)
	if _, err := e.Open(context.Background(), "TEST", 1, 1, 90, 120, 100); err != nil {
		t.Fatal(err)
	}
	fill, err := e.CloseAt("TEST", 90, "stop")
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, fill, 89.1)
	assertNear(t, mustEquity(t, e), 987.9099)
}
func TestStateRestoresEquityOwnershipWatermarkAndDowntimeBars(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	cfg := runnerConfig(path)
	e := NewLocalExecutor(quietLog(), 1000, 0.001, 0.01)
	r := NewRunner(quietLog(), cfg, e)
	s := &recordingStrategy{signals: []int{1}}
	if err := r.RegisterStrategy("TEST", s); err != nil {
		t.Fatal(err)
	}
	r.Handle(context.Background(), event(0, 100, true))
	nextExec := NewLocalExecutor(quietLog(), 5000, 0.001, 0.01)
	next := NewRunner(quietLog(), cfg, nextExec)
	if err := next.LoadState(); err != nil {
		t.Fatal(err)
	}
	assertNear(t, mustEquity(t, nextExec), 999.899)
	restored := &recordingStrategy{}
	cs := []candle.Candle{event(0, 100, true).Candle, event(1, 100, true).Candle, event(2, 100, true).Candle}
	if err := next.WarmupStrategy("TEST", restored, cs); err != nil {
		t.Fatal(err)
	}
	if restored.state != (backtest.PositionState{Dir: 1, Owner: "mr", BarsIn: 2}) {
		t.Fatalf("restored: %+v", restored.state)
	}
	if !next.lastClosed["TEST"].Equal(cs[2].Time) {
		t.Fatal("warmup watermark not advanced")
	}
	next.Handle(context.Background(), event(0, 50, true))
	if len(nextExec.Positions()) != 1 {
		t.Fatal("replayed warmup closed restored position")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("snapshot permissions: %v", info.Mode())
	}
}
func TestStateRejectsCorruptLegacyAndIncompatibleFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	cfg := runnerConfig(path)
	for _, content := range []string{"{", `{"mode":"local","positions":{}}`, `{"version":99}`, `{"version":1,"mode":"testnet"}`} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		e := NewLocalExecutor(quietLog(), 1000, 0, 0)
		r := NewRunner(quietLog(), cfg, e)
		if err := r.LoadState(); err == nil {
			t.Fatalf("accepted invalid state: %s", content)
		}
		assertNear(t, mustEquity(t, e), 1000)
	}
}
func TestSaveFailurePreservesSnapshotAndBlocksNewEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	e := NewLocalExecutor(quietLog(), 1000, 0, 0)
	r := NewRunner(quietLog(), runnerConfig(path), e)
	if err := r.SaveState(); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	e.equity = math.NaN()
	if err := r.SaveState(); err == nil || !r.persistenceBlocked {
		t.Fatal("failed save was ignored")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed save overwrote snapshot")
	}
	e.equity = 1000
	s := &recordingStrategy{signals: []int{1}}
	if err := r.RegisterStrategy("TEST", s); err != nil {
		t.Fatal(err)
	}
	r.Handle(context.Background(), event(0, 100, true))
	if len(e.Positions()) != 0 {
		t.Fatal("opened while persistence blocked")
	}
	if r.persistenceBlocked {
		t.Fatal("successful retry did not unblock persistence")
	}
}
func TestPositionsReturnsIndependentCopies(t *testing.T) {
	e := NewLocalExecutor(quietLog(), 1000, 0, 0)
	if _, err := e.Open(context.Background(), "TEST", 1, 1, 90, 0, 100); err != nil {
		t.Fatal(err)
	}
	snapshot := e.Positions()
	snapshot["TEST"].Qty = 99
	delete(snapshot, "TEST")
	assertNear(t, e.Positions()["TEST"].Qty, 1)
}

type failingReconciler struct {
	*LocalExecutor
	fail bool
}

func (e *failingReconciler) Reconcile(context.Context, []string) error {
	if e.fail {
		return fmt.Errorf("exchange unavailable")
	}
	return nil
}
func TestFailedReconcilePausesAndThenReplaysCandle(t *testing.T) {
	e := &failingReconciler{LocalExecutor: NewLocalExecutor(quietLog(), 1000, 0, 0), fail: true}
	r := NewRunner(quietLog(), runnerConfig(""), e)
	s := &recordingStrategy{}
	if err := r.RegisterStrategy("TEST", s); err != nil {
		t.Fatal(err)
	}
	if err := r.Reconcile(context.Background()); err == nil {
		t.Fatal("failure ignored")
	}
	r.Handle(context.Background(), event(0, 100, true))
	if s.calls != 0 {
		t.Fatal("strategy advanced while reconciliation failed")
	}
	e.fail = false
	if err := r.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	r.Handle(context.Background(), event(0, 100, true))
	if s.calls != 1 {
		t.Fatal("recovered candle was marked consumed")
	}
}
func TestWarmupRejectsHistoryBehindSnapshot(t *testing.T) {
	e := NewLocalExecutor(quietLog(), 1000, 0, 0)
	r := NewRunner(quietLog(), runnerConfig(""), e)
	r.MarkClosed("TEST", event(2, 100, true).Candle.Time)
	s := &recordingStrategy{}
	if err := r.WarmupStrategy("TEST", s, []candle.Candle{event(0, 100, true).Candle}); err == nil {
		t.Fatal("accepted stale history")
	}
	if s.calls != 0 {
		t.Fatal("mutated indicators before validation")
	}
}
