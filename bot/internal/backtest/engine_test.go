package backtest

import (
	"math"
	"testing"
	"time"

	"bot/internal/candle"
	"bot/internal/market"
)

type testSignal struct {
	dir        int
	stop, take float64
}

type scriptedStrategy struct {
	signals []testSignal
	next    int
	changes []string
	funding []float64
}

func (s *scriptedStrategy) OnCandle(c candle.Candle) (int, float64, float64) {
	i := s.next
	s.next++
	if i >= len(s.signals) {
		return 0, 0, 0
	}
	v := s.signals[i]
	return v.dir, v.stop, v.take
}

func (s *scriptedStrategy) OnPositionChange(dir int, reason string) {
	s.changes = append(s.changes, reason)
}

func (s *scriptedStrategy) OnFunding(rate float64, at time.Time) {
	s.funding = append(s.funding, rate)
}

func bars(prices ...float64) []candle.Candle {
	out := make([]candle.Candle, len(prices))
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, p := range prices {
		out[i] = candle.Candle{Time: start.Add(time.Duration(i) * time.Hour), O: p, H: p, L: p, C: p, V: 1}
	}
	return out
}

func testConfig() Config {
	return Config{Symbol: "TEST", TF: "1h", StartEquity: 1000, RiskPct: 0.01}
}

func near(t *testing.T, got, want float64) {
	t.Helper()
	if math.IsNaN(got) || math.Abs(got-want) > 1e-8 {
		t.Fatalf("got %.12f, want %.12f", got, want)
	}
}

func reconciles(t *testing.T, r *Report) {
	t.Helper()
	equity := r.StartEquity
	var fees, funding float64
	for _, trade := range r.Trades {
		equity += trade.PnL
		fees += trade.Fees
		funding += trade.Funding
	}
	near(t, r.FinalEquity, equity)
	near(t, r.TotalFees, fees)
	near(t, r.TotalFunding, funding)
	if len(r.Equity) > 0 {
		near(t, r.Equity[len(r.Equity)-1], r.FinalEquity)
	}
}

func TestRunCostsAndFunding(t *testing.T) {
	for _, dir := range []int{1, -1} {
		for _, useFunding := range []bool{false, true} {
			t.Run(fmtCase(dir, useFunding), func(t *testing.T) {
				cfg := testConfig()
				cfg.TakerFeePct, cfg.SlippagePct, cfg.UseFunding = 0.001, 0.01, useFunding
				mark := 100 + float64(dir)*10
				cs := bars(100, mark, mark)
				st := &scriptedStrategy{signals: []testSignal{{dir, 100 - float64(dir)*10, 0}, {dir, 0, 0}}}
				fs := []market.Funding{{CalcTime: cs[1].Time, Rate: 0.001}}
				r := Run(cfg, st, cs, fs)
				entry := 100 * (1 + float64(dir)*0.01)
				exit := mark * (1 - float64(dir)*0.01)
				qty := 10 / 11.0
				fees := (entry + exit) * qty * 0.001
				fund := 0.0
				if useFunding {
					fund = -float64(dir) * qty * mark * 0.001
				}
				if len(r.Trades) != 1 {
					t.Fatalf("trades: %d", len(r.Trades))
				}
				near(t, r.Trades[0].Qty, qty)
				near(t, r.Trades[0].PnL, float64(dir)*(exit-entry)*qty-fees+fund)
				near(t, r.Equity[0], 1000+float64(dir)*(100-entry)*qty-entry*qty*0.001)
				near(t, r.Equity[1], 1000+float64(dir)*(mark-entry)*qty-entry*qty*0.001+fund)
				if len(st.funding) != 1 {
					t.Fatal("funding must reach strategy even when charges disabled")
				}
				reconciles(t, r)
			})
		}
	}
}

func fmtCase(dir int, funding bool) string {
	name := "long"
	if dir < 0 {
		name = "short"
	}
	if funding {
		name += "/funding"
	} else {
		name += "/no-funding"
	}
	return name
}

func TestRunStopTakePriority(t *testing.T) {
	for _, dir := range []int{1, -1} {
		for _, both := range []bool{false, true} {
			t.Run(fmtCase(dir, both), func(t *testing.T) {
				cs := bars(100, 100)
				stop, take := 100-float64(dir)*10, 100+float64(dir)*20
				cs[0].L, cs[0].H = 50, 150 // Entry candle cannot hit its own protection.
				cs[1].L, cs[1].H = 100, 100
				if dir > 0 {
					cs[1].H = take
					if both {
						cs[1].L = stop
					}
				} else {
					cs[1].L = take
					if both {
						cs[1].H = stop
					}
				}
				st := &scriptedStrategy{signals: []testSignal{{dir, stop, take}}}
				r := Run(testConfig(), st, cs, nil)
				reason, price := "take", take
				if both {
					reason, price = "stop", stop
				}
				if len(r.Trades) != 1 || r.Trades[0].ExitReason != reason {
					t.Fatalf("trades: %+v", r.Trades)
				}
				near(t, r.Trades[0].ExitPrice, price)
				if len(st.changes) != 2 || st.changes[1] != reason {
					t.Fatalf("changes: %v", st.changes)
				}
				reconciles(t, r)
			})
		}
	}
}

func TestRunWarmupAndEndCosts(t *testing.T) {
	cs := bars(100, 100, 100)
	cfg := testConfig()
	cfg.StartTrading = cs[1].Time
	cfg.TakerFeePct = 0.001
	st := &scriptedStrategy{signals: []testSignal{{1, 90, 0}, {1, 90, 0}, {1, 90, 0}}}
	r := Run(cfg, st, cs, nil)
	if st.next != 3 || len(r.Trades) != 1 || !r.Trades[0].EntryTime.Equal(cs[1].Time) {
		t.Fatalf("warmup: %+v", r.Trades)
	}
	if len(r.Equity) != 2 || !r.EquityTimes[0].Equal(cs[1].Time) {
		t.Fatal("warmup included in curve")
	}
	if r.Trades[0].ExitReason != "end" {
		t.Fatal("expected terminal close")
	}
	near(t, r.Equity[0], 999.9)
	near(t, r.FinalEquity, 999.8)
	near(t, r.MaxDrawdown, 0.0002)
	reconciles(t, r)
}

func TestRunNoTradingPreservesCapital(t *testing.T) {
	for _, cs := range [][]candle.Candle{nil, bars(100, 110)} {
		cfg := testConfig()
		cfg.StartTrading = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		r := Run(cfg, &scriptedStrategy{}, cs, nil)
		near(t, r.FinalEquity, cfg.StartEquity)
		near(t, r.TotalReturn, 0)
		reconciles(t, r)
	}
}

func TestRunDoesNotOpenWithExhaustedCapital(t *testing.T) {
	cfg := testConfig()
	cfg.StartEquity = 0
	rep := Run(cfg, &scriptedStrategy{signals: []testSignal{{dir: 1, stop: 90}}}, bars(100, 110), nil)
	if len(rep.Trades) != 0 || rep.FinalEquity != 0 {
		t.Fatalf("zero-capital position: %+v", rep)
	}
}
