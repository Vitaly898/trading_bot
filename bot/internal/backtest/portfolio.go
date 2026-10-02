package backtest

import (
	"sort"
	"time"

	"bot/internal/candle"
	"bot/internal/data"
)

// PortfolioConfig — параметры портфельного прогона.
type PortfolioConfig struct {
	Config
	MaxPositions int     // макс. одновременных позиций (0 = без лимита)
	MaxTotalRisk float64 // макс. суммарный риск открытых позиций, доля equity (0 = без лимита)
}

// SymbolData — данные одного символа.
type SymbolData struct {
	Symbol  string
	Candles []candle.Candle
	Funding []data.Funding
}

// portfolioPosition — позиция в портфеле: базовая позиция + дистанция стопа
// для учёта в бюджете риска.
type portfolioPosition struct {
	*position
	stopDist float64
}

// event — единица общего таймлайна.
type event struct {
	t      time.Time
	sym    string
	candle *candle.Candle
	fund   *float64 // ставка funding, если это funding-событие
}

// RunPortfolio — прогон одной стратегии на корзине символов с общим капиталом.
// На каждый символ — свой инстанс стратегии (stateful!). Сайзинг от текущей
// портфельной equity. Лимит позиций: лишние входы пропускаются.
func RunPortfolio(cfg PortfolioConfig, mkStrategy func(sym string) Strategy, ds []SymbolData) *Report {
	rep := &Report{Config: cfg.Config}

	// общий таймлайн: funding-события и свечи
	var events []event
	strategies := map[string]Strategy{}
	lastClose := map[string]float64{}
	for _, d := range ds {
		if len(d.Candles) == 0 {
			continue
		}
		strategies[d.Symbol] = mkStrategy(d.Symbol)
		for _, f := range d.Funding {
			rate := f.Rate
			events = append(events, event{t: f.CalcTime, sym: d.Symbol, fund: &rate})
		}
		for i := range d.Candles {
			events = append(events, event{t: d.Candles[i].Time, sym: d.Symbol, candle: &d.Candles[i]})
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		if !events[i].t.Equal(events[j].t) {
			return events[i].t.Before(events[j].t)
		}
		return events[i].fund != nil // funding раньше свечи при равном времени
	})
	if len(events) == 0 {
		return rep
	}
	rep.StartTime = events[0].t
	if !cfg.StartTrading.IsZero() {
		rep.StartTime = cfg.StartTrading
	}
	rep.EndTime = events[len(events)-1].t
	active := func(t time.Time) bool {
		return cfg.StartTrading.IsZero() || !t.Before(cfg.StartTrading)
	}

	equity := cfg.StartEquity
	positions := map[string]*portfolioPosition{}
	openRisk := func() float64 {
		sum := 0.0
		for _, p := range positions {
			sum += p.qty * p.stopDist
		}
		return sum
	}
	peak := equity
	maxDD := 0.0
	timeInMarket := 0
	activeBars := 0

	closePos := func(sym string, price float64, t time.Time, reason string) {
		pp := positions[sym]
		if pp == nil {
			return
		}
		pos := pp.position
		fill := applySlippage(price, -pos.dir, cfg.SlippagePct)
		exitFee := fill * pos.qty * cfg.TakerFeePct
		pos.fees += exitFee
		gross := float64(pos.dir) * (fill - pos.entryPrice) * pos.qty
		pnl := gross - pos.fees + pos.funding
		equity += pnl
		rep.Trades = append(rep.Trades, Trade{
			Symbol: sym,
			Dir: pos.dir, EntryTime: pos.entryTime, ExitTime: t,
			EntryPrice: pos.entryPrice, ExitPrice: fill, Qty: pos.qty,
			PnL: pnl, Fees: pos.fees, Funding: pos.funding, ExitReason: reason,
		})
		rep.TotalFees += pos.fees
		rep.TotalFunding += pos.funding
		delete(positions, sym)
		if pa, ok := strategies[sym].(PositionAware); ok {
			pa.OnPositionChange(0, reason)
		}
	}

	equityCurve := []float64{}
	var curveTimes []time.Time

	for _, ev := range events {
		// funding-событие
		if ev.fund != nil {
			if fa, ok := strategies[ev.sym].(FundingAware); ok {
				fa.OnFunding(*ev.fund, ev.t)
			}
			if cfg.UseFunding {
				if pp := positions[ev.sym]; pp != nil && ev.t.After(pp.entryTime) {
					px := lastClose[ev.sym]
					if px > 0 {
						pp.funding += -float64(pp.dir) * pp.qty * px * (*ev.fund)
					}
				}
			}
			continue
		}

		// свеча
		c := ev.candle
		sym := ev.sym
		strat := strategies[sym]
		lastClose[sym] = c.C

		// стоп / тейк внутри свечи
		if pp := positions[sym]; pp != nil {
			pos := pp.position
			if pos.stop > 0 {
				if pos.dir > 0 && c.L <= pos.stop {
					closePos(sym, pos.stop, c.Time, "stop")
				} else if pos.dir < 0 && c.H >= pos.stop {
					closePos(sym, pos.stop, c.Time, "stop")
				}
			}
			if pp := positions[sym]; pp != nil && pp.take > 0 {
				if pp.dir > 0 && c.H >= pp.take {
					closePos(sym, pp.take, c.Time, "take")
				} else if pp.dir < 0 && c.L <= pp.take {
					closePos(sym, pp.take, c.Time, "take")
				}
			}
		}

		// сигнал стратегии
		dir, stopPrice, takePrice := strat.OnCandle(*c)
		curDir := 0
		if pp := positions[sym]; pp != nil {
			curDir = pp.dir
		}
		if dir != curDir {
			if curDir != 0 {
				closePos(sym, c.C, c.Time, "signal")
			}
			if dir != 0 && active(c.Time) && (cfg.MaxPositions == 0 || len(positions) < cfg.MaxPositions) {
				entry := applySlippage(c.C, dir, cfg.SlippagePct)
				stopDist := entry * 0.02
				if stopPrice > 0 {
					if d := abs(entry - stopPrice); d > 0 {
						stopDist = d
					}
				}
				qty := equity * cfg.RiskPct / stopDist
				// бюджет риска: суммарный риск открытых + новый ≤ лимита
				if cfg.MaxTotalRisk > 0 && openRisk()+qty*stopDist > cfg.MaxTotalRisk*equity {
					qty = 0 // вход запрещён бюджетом
				}
				if qty > 0 {
					entryFee := entry * qty * cfg.TakerFeePct
					positions[sym] = &portfolioPosition{
						position: &position{
							dir: dir, qty: qty, entryPrice: entry,
							entryTime: c.Time, stop: stopPrice, take: takePrice, fees: entryFee,
						},
						stopDist: stopDist,
					}
					if pa, ok := strat.(PositionAware); ok {
						pa.OnPositionChange(dir, "")
					}
				}
			}
		}

		// mark-to-market (раз на timestamp, только активный период)
		if active(c.Time) && (len(curveTimes) == 0 || !curveTimes[len(curveTimes)-1].Equal(c.Time)) {
			mark := equity
			for s, pp := range positions {
				if px := lastClose[s]; px > 0 {
					mark += float64(pp.dir) * (px - pp.entryPrice) * pp.qty
				}
			}
			equityCurve = append(equityCurve, mark)
			curveTimes = append(curveTimes, c.Time)
			if len(positions) > 0 {
				timeInMarket++
			}
			activeBars++
			if mark > peak {
				peak = mark
			}
			if dd := (peak - mark) / peak; dd > maxDD {
				maxDD = dd
			}
		}
	}

	// закрыть хвосты
	lastT := events[len(events)-1].t
	for sym := range positions {
		closePos(sym, lastClose[sym], lastT, "end")
	}

	rep.FinalEquity = equity
	rep.TotalReturn = equity/cfg.StartEquity - 1
	rep.MaxDrawdown = maxDD
	if activeBars > 0 {
		rep.ExposurePct = 100 * float64(timeInMarket) / float64(activeBars)
	}
	calcTradeStats(rep)
	rep.Equity = equityCurve
	rep.EquityTimes = curveTimes
	// Sharpe по кривой equity
	if len(equityCurve) > 2 {
		var rets []float64
		for i := 1; i < len(equityCurve); i++ {
			if equityCurve[i-1] != 0 {
				rets = append(rets, equityCurve[i]/equityCurve[i-1]-1)
			}
		}
		rep.Sharpe = sharpeFromReturns(rets, curveTimes)
	}
	return rep
}

func sharpeFromReturns(rets []float64, times []time.Time) float64 {
	if len(rets) < 2 || len(times) < 2 {
		return 0
	}
	var sum float64
	for _, r := range rets {
		sum += r
	}
	mean := sum / float64(len(rets))
	var sq float64
	for _, r := range rets {
		sq += (r - mean) * (r - mean)
	}
	std := sqrt(sq / float64(len(rets)-1))
	if std == 0 {
		return 0
	}
	span := times[len(times)-1].Sub(times[0]).Hours()
	if span <= 0 {
		return 0
	}
	perYear := float64(len(times)) / (span / (24 * 365))
	return mean / std * sqrt(perYear)
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 50; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

// RunSplit — режим «раздельных суб-портфелей»: капитал делится поровну между
// символами, каждый торгуется независимо (макс. 1 позиция на символ).
// Нет конкуренции за слоты — сильный тренд в одном символе никогда не
// блокируется пилой в другом. Портфельная equity = сумма суб-портфелей.
func RunSplit(cfg Config, mkStrategy func(sym string) Strategy, ds []SymbolData) *Report {
	rep := &Report{Config: cfg}
	if len(ds) == 0 {
		return rep
	}
	perEquity := cfg.StartEquity / float64(len(ds))

	type curve struct {
		times  []time.Time
		values []float64
	}
	curves := map[string]curve{}

	for _, d := range ds {
		if len(d.Candles) == 0 {
			continue
		}
		sub := cfg
		sub.Symbol = d.Symbol
		sub.StartEquity = perEquity
		r := Run(sub, mkStrategy(d.Symbol), d.Candles, d.Funding)
		rep.Trades = append(rep.Trades, r.Trades...)
		rep.TotalFees += r.TotalFees
		rep.TotalFunding += r.TotalFunding
		curves[d.Symbol] = curve{times: r.EquityTimes, values: r.Equity}
	}

	// объединённый таймлайн
	timeSet := map[int64]time.Time{}
	for _, c := range curves {
		for _, t := range c.times {
			timeSet[t.Unix()] = t
		}
	}
	times := make([]time.Time, 0, len(timeSet))
	for _, t := range timeSet {
		times = append(times, t)
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	if len(times) == 0 {
		calcTradeStats(rep)
		return rep
	}

	// суммируем кривые с last-value-fill
	rep.StartTime = times[0]
	rep.EndTime = times[len(times)-1]
	equity := make([]float64, 0, len(times))
	idx := map[string]int{}
	last := map[string]float64{}
	peak := 0.0
	maxDD := 0.0
	for _, t := range times {
		sum := 0.0
		for sym, c := range curves {
			for idx[sym] < len(c.times) && !c.times[idx[sym]].After(t) {
				last[sym] = c.values[idx[sym]]
				idx[sym]++
			}
			if v, ok := last[sym]; ok {
				sum += v
			} else {
				sum += perEquity // символ ещё не торговал (нет данных)
			}
		}
		equity = append(equity, sum)
		if sum > peak {
			peak = sum
		}
		if dd := (peak - sum) / peak; dd > maxDD {
			maxDD = dd
		}
	}

	rep.Equity = equity
	rep.EquityTimes = times
	rep.FinalEquity = equity[len(equity)-1]
	rep.TotalReturn = rep.FinalEquity/cfg.StartEquity - 1
	rep.MaxDrawdown = maxDD
	calcTradeStats(rep)
	rep.Sharpe = sharpeFromReturns(returns(equity), times)
	return rep
}

func returns(equity []float64) []float64 {
	var rets []float64
	for i := 1; i < len(equity); i++ {
		if equity[i-1] != 0 {
			rets = append(rets, equity[i]/equity[i-1]-1)
		}
	}
	return rets
}
