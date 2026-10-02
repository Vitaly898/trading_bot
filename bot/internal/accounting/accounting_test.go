package accounting_test

import (
	"context"
	"io"
	"log/slog"
	"math"
	"testing"
	"time"

	"bot/internal/accounting"
	"bot/internal/backtest"
	"bot/internal/candle"
	"bot/internal/live"
)

type hold struct{ dir int }

func (s hold) OnCandle(c candle.Candle) (int, float64, float64) {
	return s.dir, 100 - float64(s.dir)*20, 0
}
func TestLocalAndBacktestCostsAgree(t *testing.T) {
	for _, dir := range []int{-1, 1} {
		cfg := backtest.Config{Symbol: "A", TF: "1h", StartEquity: 1000, RiskPct: .01, TakerFeePct: .001, SlippagePct: .002}
		now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		cs := []candle.Candle{{Time: now, O: 100, H: 100, L: 100, C: 100}, {Time: now.Add(time.Hour), O: 110, H: 110, L: 110, C: 110}}
		rep := backtest.Run(cfg, hold{dir}, cs, nil)
		entry := accounting.Slippage(100, dir, cfg.SlippagePct)
		stop := 100 - float64(dir)*20
		qty := accounting.Quantity(1000, .01, accounting.StopDistance(entry, stop))
		e := live.NewLocalExecutor(slog.New(slog.NewTextHandler(io.Discard, nil)), 1000, .001, .002)
		if _, err := e.Open(context.Background(), "A", dir, qty, stop, 0, 100); err != nil {
			t.Fatal(err)
		}
		if _, err := e.Close(context.Background(), "A", 110); err != nil {
			t.Fatal(err)
		}
		equity, err := e.Equity(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(equity-rep.FinalEquity) > 1e-9 {
			t.Fatalf("dir %d: local %.10f backtest %.10f", dir, equity, rep.FinalEquity)
		}
	}
}
func TestInvalidSizingCannotCreateQuantity(t *testing.T) {
	for _, inputs := range [][3]float64{{0, .01, 2}, {-1, .01, 2}, {1000, .01, 0}, {math.NaN(), .01, 2}, {1000, .01, math.Inf(1)}, {math.MaxFloat64, .9, .001}} {
		if accounting.Quantity(inputs[0], inputs[1], inputs[2]) != 0 {
			t.Fatal(inputs)
		}
	}
}
