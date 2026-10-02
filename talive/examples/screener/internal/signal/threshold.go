package signal

import (
	"log/slog"
	"screener/internal/binance"

	"github.com/MoroZvlg/talive"
)

type ThresholdSignal struct {
	log           *slog.Logger
	indicator     talive.Indicator
	buyThreshold  float64
	sellThreshold float64
	reverse       int
}

func NewThresholdSignal(log *slog.Logger, indicator talive.Indicator, buyTh, sellTh float64, reverse bool) *ThresholdSignal {
	var r int
	if reverse {
		r = -1
	} else {
		r = 1
	}
	return &ThresholdSignal{
		log:           log,
		indicator:     indicator,
		buyThreshold:  buyTh,
		sellThreshold: sellTh,
		reverse:       r,
	}
}

func (s *ThresholdSignal) Next(kline *binance.Kline) int {
	return s.signal(s.indicator.Next(kline)[0])
}

func (s *ThresholdSignal) Current(kline *binance.Kline) int {
	return s.signal(s.indicator.Current(kline)[0])
}

func (s *ThresholdSignal) signal(value float64) int {
	signal := 0
	if s.indicator.IsWarmedUp() {
		if value > s.sellThreshold {
			signal = -1 * s.reverse
		}
		if value < s.buyThreshold {
			signal = 1 * s.reverse
		}
	}
	s.log.Debug("signal", "indicator", s.indicator, "signal", signal, "value", value)
	return signal
}

func (s *ThresholdSignal) MaxWarmUp() int {
	return s.indicator.WarmUpPeriod()
}
