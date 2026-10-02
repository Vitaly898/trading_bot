package experiment

import (
	"math"
	"testing"
	"time"

	"bot/internal/backtest"
	"bot/internal/candle"
	"bot/internal/config"
	"bot/internal/strategy"
)

func experimentConfig() *config.Config {
	c := &config.Config{Symbols: []string{"A", "B"}, TF: "1h", DB: "unused", Equity: 1000, Risk: .01, Fee: .001, Slip: .002, MaxPositions: 1, MaxTotalRisk: .01}
	c.Strategy.Name = "ema_cross"
	c.Strategy.Params = map[string]any{"fast": 2, "slow": 4, "atr_period": 2, "atr_mult": 2.0}
	return c
}
func history() []backtest.SymbolData {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	var ds []backtest.SymbolData
	for _, sym := range []string{"A", "B"} {
		var cs []candle.Candle
		for i := 0; i < 40; i++ {
			price := 100 + float64(i)
			cs = append(cs, candle.Candle{Time: start.Add(time.Duration(i) * time.Hour), O: price, H: price + .5, L: price - .5, C: price, V: 1})
		}
		ds = append(ds, backtest.SymbolData{Symbol: sym, Candles: cs})
	}
	return ds
}
func TestRunHonorsPortfolioModeAndFreshStrategies(t *testing.T) {
	ds := history()
	c := experimentConfig()
	start := ds[0].Candles[10].Time
	mk := func(string) backtest.Strategy {
		s, err := strategy.New(c.Strategy.Name, strategy.Params(c.Strategy.Params))
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	tradeCounts := map[string]int{}
	for _, mode := range []string{"shared", "split"} {
		c.Mode = mode
		got, err := Run(c, ds, start)
		if err != nil {
			t.Fatal(err)
		}
		cfg := EngineConfig(c, start)
		var want *backtest.Report
		if mode == "split" {
			want = backtest.RunSplit(cfg, mk, ds)
		} else {
			want = backtest.RunPortfolio(backtest.PortfolioConfig{Config: cfg, MaxPositions: 1, MaxTotalRisk: .01}, mk, ds)
		}
		if len(got.Trades) == 0 || len(got.Trades) != len(want.Trades) || math.Abs(got.FinalEquity-want.FinalEquity) > 1e-9 {
			t.Fatalf("%s: got %v want %v", mode, got, want)
		}
		repeated, err := Run(c, ds, start)
		if err != nil || repeated.FinalEquity != got.FinalEquity {
			t.Fatalf("strategy state leaked: %v", err)
		}
		for _, tr := range got.Trades {
			if tr.EntryTime.Before(start) {
				t.Fatal("traded during warmup")
			}
		}
		tradeCounts[mode] = len(got.Trades)
	}
	if tradeCounts["shared"] == tradeCounts["split"] {
		t.Fatal("fixture does not distinguish dispatch modes")
	}
}
func TestRunSingleAndInvalidOverride(t *testing.T) {
	c := experimentConfig()
	c.Symbol = "A"
	c.Symbols = nil
	ds := history()[:1]
	got, err := Run(c, ds, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	s, err := strategy.New(c.Strategy.Name, strategy.Params(c.Strategy.Params))
	if err != nil {
		t.Fatal(err)
	}
	want := backtest.Run(EngineConfig(c, time.Time{}), s, ds[0].Candles, nil)
	if got.FinalEquity != want.FinalEquity {
		t.Fatal("single dispatch changed accounting")
	}
	if _, err := c.WithParams(map[string]any{"fast": "2"}); err == nil {
		t.Fatal("invalid override accepted")
	}
}

func TestRunRejectsMissingAndDuplicateSymbols(t *testing.T) {
	c := experimentConfig()
	ds := history()
	if _, err := Run(c, ds[:1], time.Time{}); err == nil {
		t.Fatal("accepted missing symbol")
	}
	ds[1].Symbol = ds[0].Symbol
	if _, err := Run(c, ds, time.Time{}); err == nil {
		t.Fatal("accepted duplicate dataset")
	}
}
