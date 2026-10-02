package backtest

import (
	"testing"
	"time"

	"bot/internal/market"
)

func TestPortfolioMarksAllSymbolsAtTimestamp(t *testing.T) {
	ds := []SymbolData{{Symbol: "A", Candles: bars(100, 110, 110)}, {Symbol: "B", Candles: bars(100, 120, 120)}}
	for _, reverse := range []bool{false, true} {
		if reverse {
			ds[0], ds[1] = ds[1], ds[0]
		}
		mk := func(string) Strategy { return &scriptedStrategy{signals: []testSignal{{1, 90, 0}, {1, 0, 0}}} }
		r := RunPortfolio(PortfolioConfig{Config: testConfig()}, mk, ds)
		if len(r.Equity) != 3 {
			t.Fatalf("curve length: %d", len(r.Equity))
		}
		near(t, r.Equity[1], 1030)
		near(t, r.FinalEquity, 1030)
		reconciles(t, r)
	}
}

func TestPortfolioAccruedCostsAndFinalClose(t *testing.T) {
	cfg := testConfig()
	cfg.TakerFeePct, cfg.UseFunding = 0.001, true
	cs := bars(100, 100, 100)
	ds := []SymbolData{{Symbol: "A", Candles: cs, Funding: []market.Funding{{CalcTime: cs[1].Time, Rate: 0.001}}}}
	mk := func(string) Strategy {
		return &scriptedStrategy{signals: []testSignal{{1, 90, 0}, {1, 0, 0}, {1, 0, 0}}}
	}
	r := RunPortfolio(PortfolioConfig{Config: cfg}, mk, ds)
	near(t, r.Equity[0], 999.9)
	near(t, r.Equity[1], 999.8)
	near(t, r.FinalEquity, 999.7)
	near(t, r.MaxDrawdown, 0.0003)
	reconciles(t, r)
}

func TestPortfolioRiskLimits(t *testing.T) {
	ds := []SymbolData{{Symbol: "A", Candles: bars(100, 100)}, {Symbol: "B", Candles: bars(100, 100)}}
	for _, cfg := range []PortfolioConfig{
		{Config: testConfig(), MaxPositions: 1},
		{Config: testConfig(), MaxTotalRisk: 0.01},
	} {
		mk := func(string) Strategy { return &scriptedStrategy{signals: []testSignal{{1, 90, 0}, {1, 0, 0}}} }
		r := RunPortfolio(cfg, mk, ds)
		if len(r.Trades) != 1 {
			t.Fatalf("expected one accepted entry, got %d", len(r.Trades))
		}
		reconciles(t, r)
	}
}

func TestSplitIncludesTerminalCostsAndIdleAllocation(t *testing.T) {
	cfg := testConfig()
	cfg.TakerFeePct = 0.001
	ds := []SymbolData{{Symbol: "A", Candles: bars(100, 100)}, {Symbol: "B"}}
	mk := func(string) Strategy { return &scriptedStrategy{signals: []testSignal{{1, 90, 0}, {1, 0, 0}}} }
	r := RunSplit(cfg, mk, ds)
	near(t, r.Equity[0], 999.95)
	near(t, r.FinalEquity, 999.9)
	near(t, r.MaxDrawdown, 0.0001)
	reconciles(t, r)
}

func TestSplitAlignsDifferentHistories(t *testing.T) {
	ds := []SymbolData{{Symbol: "A", Candles: bars(100, 110)}, {Symbol: "B", Candles: bars(100, 100, 120)[1:]}}
	mk := func(string) Strategy { return &scriptedStrategy{signals: []testSignal{{1, 90, 0}, {1, 0, 0}}} }
	r := RunSplit(testConfig(), mk, ds)
	if len(r.Equity) != 3 {
		t.Fatalf("curve length: %d", len(r.Equity))
	}
	near(t, r.Equity[0], 1000)
	near(t, r.Equity[1], 1005)
	near(t, r.Equity[2], 1015)
	reconciles(t, r)
}

func TestPortfolioEmptyPreservesCapital(t *testing.T) {
	mk := func(string) Strategy { return &scriptedStrategy{} }
	for _, ds := range [][]SymbolData{nil, {{Symbol: "A"}}} {
		reconciles(t, RunPortfolio(PortfolioConfig{Config: testConfig()}, mk, ds))
		reconciles(t, RunSplit(testConfig(), mk, ds))
	}
}

func TestPortfolioSingleSymbolMatchesRun(t *testing.T) {
	for _, dir := range []int{1, -1} {
		cfg := testConfig()
		cfg.TakerFeePct, cfg.SlippagePct = 0.001, 0.002
		cs := bars(100, 100+float64(dir)*10, 100+float64(dir)*20)
		mk := func(string) Strategy {
			return &scriptedStrategy{signals: []testSignal{{dir, 100 - float64(dir)*30, 0}, {dir, 0, 0}, {dir, 0, 0}}}
		}
		single := Run(cfg, mk(cfg.Symbol), cs, nil)
		shared := RunPortfolio(PortfolioConfig{Config: cfg}, mk, []SymbolData{{Symbol: cfg.Symbol, Candles: cs}})
		split := RunSplit(cfg, mk, []SymbolData{{Symbol: cfg.Symbol, Candles: cs}})
		for _, r := range []*Report{shared, split} {
			near(t, r.FinalEquity, single.FinalEquity)
			near(t, r.MaxDrawdown, single.MaxDrawdown)
			near(t, r.Sharpe, single.Sharpe)
			for i, v := range single.Equity {
				near(t, r.Equity[i], v)
			}
			reconciles(t, r)
		}
	}
}

func TestPortfolioFinalFundingWithoutCandle(t *testing.T) {
	cfg := testConfig()
	cfg.UseFunding = true
	cs := bars(100)
	at := cs[0].Time.Add(time.Hour)
	ds := []SymbolData{{Symbol: "A", Candles: cs, Funding: []market.Funding{{CalcTime: at, Rate: 0.001}}}}
	mk := func(string) Strategy { return &scriptedStrategy{signals: []testSignal{{1, 90, 0}}} }
	r := RunPortfolio(PortfolioConfig{Config: cfg}, mk, ds)
	near(t, r.FinalEquity, 999.9)
	if len(r.EquityTimes) != 2 || !r.EquityTimes[1].Equal(at) {
		t.Fatalf("terminal funding timestamp: %v", r.EquityTimes)
	}
	reconciles(t, r)
}
