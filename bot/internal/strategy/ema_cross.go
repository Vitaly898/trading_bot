// Package strategy — стратегии для бэктеста и лайва.
package strategy

import (
	"bot/internal/candle"

	"github.com/MoroZvlg/talive"
)

// EMACrossATR — тестовая стратегия: пересечение EMA fast/slow,
// стоп = 2×ATR от цены закрытия сигнальной свечи.
// Long+short: fast>slow → long, fast<slow → short.
type EMACrossATR struct {
	fast    talive.Indicator
	slow    talive.Indicator
	atr     talive.Indicator
	atrMult float64
}

func NewEMACrossATR(fastPeriod, slowPeriod, atrPeriod int, atrMult float64) *EMACrossATR {
	fast, _ := talive.NewEMA(fastPeriod)
	slow, _ := talive.NewEMA(slowPeriod)
	atr, _ := talive.NewATR(atrPeriod)
	return &EMACrossATR{fast: fast, slow: slow, atr: atr, atrMult: atrMult}
}

func (s *EMACrossATR) OnCandle(c candle.Candle) (int, float64, float64) {
	f := s.fast.Next(c)[0]
	sl := s.slow.Next(c)[0]
	a := s.atr.Next(c)[0]

	if s.fast.IsIdle() || s.slow.IsIdle() || s.atr.IsIdle() {
		return 0, 0, 0
	}
	if f > sl {
		return 1, c.C - s.atrMult*a, 0
	}
	if f < sl {
		return -1, c.C + s.atrMult*a, 0
	}
	return 0, 0, 0
}
