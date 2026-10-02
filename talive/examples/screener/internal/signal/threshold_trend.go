package signal

import (
	"log/slog"
	"screener/internal/binance"

	"github.com/MoroZvlg/talive"
)

type ThresholdTrendSignal struct {
	log        *slog.Logger
	indicator  talive.Indicator
	buyTh      float64
	sellTh     float64
	lastResult float64
	hasLast    bool
}

func NewThresholdTrendSignal(log *slog.Logger, indicator talive.Indicator, buyTh, sellTh float64) *ThresholdTrendSignal {
	return &ThresholdTrendSignal{
		log:       log,
		indicator: indicator,
		buyTh:     buyTh,
		sellTh:    sellTh,
	}
}

func (s *ThresholdTrendSignal) Next(kline *binance.Kline) int {
	value := s.indicator.Next(kline)[0]
	signal := s.signal(value)

	s.lastResult = value
	s.hasLast = true
	return signal
}

func (s *ThresholdTrendSignal) Current(kline *binance.Kline) int {
	return s.signal(s.indicator.Current(kline)[0])
}

func (s *ThresholdTrendSignal) signal(value float64) int {
	signal := 0
	if s.indicator.IsWarmedUp() && s.hasLast {
		if value < s.buyTh && value > s.lastResult {
			signal = 1
		}
		if value > s.sellTh && value < s.lastResult {
			signal = -1
		}
	}
	s.log.Debug("signal", "indicator", s.indicator, "signal", signal, "value", value, "prev", s.lastResult)
	return signal
}

func (s *ThresholdTrendSignal) MaxWarmUp() int {
	return s.indicator.WarmUpPeriod()
}
