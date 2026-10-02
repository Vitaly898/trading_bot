package backtest

import (
	"time"

	"bot/internal/accounting"
	"bot/internal/candle"
	"bot/internal/market"
)

type position struct {
	dir        int
	qty        float64
	entryPrice float64
	entryTime  time.Time
	stop       float64
	take       float64
	fees       float64
	funding    float64 // накопленный funding по позиции (отрицательный = платили)
}

// markPnL includes accrued costs without settling them twice at close.
func (p *position) markPnL(price float64) float64 {
	return float64(p.dir)*(price-p.entryPrice)*p.qty - p.fees + p.funding
}

// Run — прогон стратегии по свечам. Исполнение: по закрытию сигнальной свечи
// с проскальзыванием. Стоп: срабатывает при касании внутри следующих свечей.
func Run(cfg Config, strat Strategy, candles []candle.Candle, funding []market.Funding) *Report {
	rep := &Report{Config: cfg, FinalEquity: cfg.StartEquity}
	if len(candles) == 0 {
		return rep
	}
	rep.StartTime = candles[0].Time
	if !cfg.StartTrading.IsZero() {
		rep.StartTime = cfg.StartTrading
	}
	rep.EndTime = candles[len(candles)-1].Time
	active := func(t time.Time) bool {
		return cfg.StartTrading.IsZero() || !t.Before(cfg.StartTrading)
	}

	// уведомление стратегии о фактическом состоянии позиции
	var notify func(dir int, reason string)
	if pa, ok := strat.(PositionAware); ok {
		notify = pa.OnPositionChange
	} else {
		notify = func(int, string) {}
	}
	// уведомление стратегии о funding-событиях
	var notifyFunding func(rate float64, t time.Time)
	if fa, ok := strat.(FundingAware); ok {
		notifyFunding = fa.OnFunding
	}

	equity := cfg.StartEquity
	var pos *position
	fundIdx := 0

	timeInMarket := 0

	equityCurve := make([]float64, 0, len(candles))

	closePos := func(price float64, t time.Time, reason string) {
		if pos == nil {
			return
		}
		fill := applySlippage(price, -pos.dir, cfg.SlippagePct)
		exitFee := accounting.Fee(fill, pos.qty, cfg.TakerFeePct)
		pos.fees += exitFee
		gross := accounting.Gross(pos.dir, pos.entryPrice, fill, pos.qty)
		pnl := gross - pos.fees + pos.funding
		equity += pnl
		rep.Trades = append(rep.Trades, Trade{
			Symbol: cfg.Symbol,
			Dir:    pos.dir, EntryTime: pos.entryTime, ExitTime: t,
			EntryPrice: pos.entryPrice, ExitPrice: fill, Qty: pos.qty,
			PnL: pnl, Fees: pos.fees, Funding: pos.funding, ExitReason: reason,
		})
		rep.TotalFees += pos.fees
		rep.TotalFunding += pos.funding
		pos = nil
		notify(0, reason)
	}

	for _, c := range candles {
		// 1. Funding между предыдущей и текущей свечой
		for fundIdx < len(funding) && !funding[fundIdx].CalcTime.After(c.Time) {
			f := funding[fundIdx]
			fundIdx++
			if notifyFunding != nil {
				notifyFunding(f.Rate, f.CalcTime)
			}
			if cfg.UseFunding && pos != nil && f.CalcTime.After(pos.entryTime) {
				// long платит при rate>0, short получает
				pos.funding += -float64(pos.dir) * pos.qty * c.C * f.Rate
			}
		}

		// 2. Стоп внутри свечи (для уже открытой позиции).
		// Приоритет стопа над тейком при конфликте в одной свече (консервативно).
		if pos != nil && pos.stop > 0 {
			if pos.dir > 0 && c.L <= pos.stop {
				closePos(pos.stop, c.Time, "stop")
			} else if pos.dir < 0 && c.H >= pos.stop {
				closePos(pos.stop, c.Time, "stop")
			}
		}
		// 2.5. Тейк внутри свечи
		if pos != nil && pos.take > 0 {
			if pos.dir > 0 && c.H >= pos.take {
				closePos(pos.take, c.Time, "take")
			} else if pos.dir < 0 && c.L <= pos.take {
				closePos(pos.take, c.Time, "take")
			}
		}

		// 3. Сигнал стратегии на закрытии свечи
		dir, stopPrice, takePrice := strat.OnCandle(c)
		curDir := 0
		if pos != nil {
			curDir = pos.dir
		}
		if dir != curDir {
			if pos != nil {
				closePos(c.C, c.Time, "signal")
			}
			if dir != 0 && active(c.Time) {
				// сайзинг от риска: qty = equity*risk / stopDist
				entry := applySlippage(c.C, dir, cfg.SlippagePct)
				stopDist := accounting.StopDistance(entry, stopPrice)
				qty := accounting.Quantity(equity, cfg.RiskPct, stopDist)
				if qty > 0 {
					entryFee := accounting.Fee(entry, qty, cfg.TakerFeePct)
					pos = &position{
						dir: dir, qty: qty, entryPrice: entry,
						entryTime: c.Time, stop: stopPrice, take: takePrice, fees: entryFee,
					}
					notify(dir, "")
				}
			}
		}

		// 4. Учёт equity (mark-to-market) и просадки — только активный период
		if !active(c.Time) {
			continue
		}
		if pos != nil {
			timeInMarket++
		}
		mark := equity
		if pos != nil {
			mark += pos.markPnL(c.C)
		}
		equityCurve = append(equityCurve, mark)
	}

	if pos != nil {
		closePos(candles[len(candles)-1].C, candles[len(candles)-1].Time, "end")
		equityCurve[len(equityCurve)-1] = equity
	}

	rep.FinalEquity = equity
	rep.TotalReturn = equity/cfg.StartEquity - 1
	rep.MaxDrawdown = maxDrawdown(cfg.StartEquity, equityCurve)
	if len(equityCurve) > 0 {
		rep.ExposurePct = 100 * float64(timeInMarket) / float64(len(equityCurve))
	}
	calcTradeStats(rep)
	rep.Sharpe = calcSharpe(equityCurve, candles[len(candles)-len(equityCurve):])
	rep.Equity = equityCurve
	curveTimes := make([]time.Time, 0, len(equityCurve))
	for _, c := range candles[len(candles)-len(equityCurve):] {
		curveTimes = append(curveTimes, c.Time)
	}
	rep.EquityTimes = curveTimes
	return rep
}

func applySlippage(price float64, dir int, slipPct float64) float64 {
	// покупка дороже, продажа дешевле
	return accounting.Slippage(price, dir, slipPct)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
