package signal

import (
	"log/slog"
	"screener/internal/binance"

	"github.com/MoroZvlg/talive"
)

type MomentumSignal struct {
	log        *slog.Logger
	indicator  talive.Indicator
	lastResult float64
	hasLast    bool
}

func NewMomentumSignal(log *slog.Logger, indicator talive.Indicator) *MomentumSignal {
	return &MomentumSignal{
		log:       log,
		indicator: indicator,
	}
}

func (s *MomentumSignal) Next(kline *binance.Kline) int {
	value := s.indicator.Next(kline)[0]
	signal := s.signal(value)

	s.lastResult = value
	s.hasLast = true
	return signal
}

func (s *MomentumSignal) Current(kline *binance.Kline) int {
	return s.signal(s.indicator.Current(kline)[0])
}

func (s *MomentumSignal) signal(value float64) int {
	signal := 0
	if s.indicator.IsWarmedUp() && s.hasLast {
		if value > s.lastResult {
			signal = 1
		} else if value < s.lastResult {
			signal = -1
		}
	}
	s.log.Debug("signal", "indicator", s.indicator, "signal", signal, "value", value, "prev", s.lastResult)
	return signal
}

func (s *MomentumSignal) MaxWarmUp() int {
	return s.indicator.WarmUpPeriod()
}
