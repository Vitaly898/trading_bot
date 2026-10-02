package backtest

import (
	"math"

	"bot/internal/candle"
)

func calcTradeStats(rep *Report) {
	var grossWin, grossLoss float64
	rep.ExitReasons = map[string]int{}
	for _, t := range rep.Trades {
		rep.ExitReasons[t.ExitReason]++
		if t.PnL > 0 {
			rep.Wins++
			grossWin += t.PnL
		} else if t.PnL < 0 {
			rep.Losses++
			grossLoss += t.PnL
		}
	}
	n := rep.Wins + rep.Losses
	if n > 0 {
		rep.WinRate = 100 * float64(rep.Wins) / float64(n)
		rep.Expectancy = (grossWin + grossLoss) / float64(n)
	}
	if rep.Wins > 0 {
		rep.AvgWin = grossWin / float64(rep.Wins)
	}
	if rep.Losses > 0 {
		rep.AvgLoss = grossLoss / float64(rep.Losses)
	}
	if grossLoss < 0 {
		rep.ProfitFactor = grossWin / (-grossLoss)
	} else if grossWin > 0 {
		rep.ProfitFactor = math.Inf(1)
	}
}

// calcSharpe — Sharpe по доходностям свечей, годовая.
func calcSharpe(equity []float64, candles []candle.Candle) float64 {
	if len(equity) < 3 {
		return 0
	}
	var rets []float64
	for i := 1; i < len(equity); i++ {
		if equity[i-1] != 0 {
			rets = append(rets, equity[i]/equity[i-1]-1)
		}
	}
	if len(rets) < 2 {
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
	std := math.Sqrt(sq / float64(len(rets)-1))
	if std == 0 {
		return 0
	}
	// свечей в год
	span := candles[len(candles)-1].Time.Sub(candles[0].Time).Hours()
	if span <= 0 {
		return 0
	}
	perYear := float64(len(candles)) / (span / (24 * 365))
	return mean / std * math.Sqrt(perYear)
}

// maxDrawdown starts at the initial capital, so first-bar costs count too.
func maxDrawdown(startEquity float64, equity []float64) float64 {
	peak, maxDD := startEquity, 0.0
	for _, mark := range equity {
		if mark > peak {
			peak = mark
		}
		if peak > 0 {
			if dd := (peak - mark) / peak; dd > maxDD {
				maxDD = dd
			}
		}
	}
	return maxDD
}
