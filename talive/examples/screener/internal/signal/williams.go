package signal

import (
	"log/slog"
	"screener/internal/binance"

	"github.com/MoroZvlg/talive"
)

type WilliamsSignal struct {
	log        *slog.Logger
	indicator  *talive.Williams
	lastResult float64
	hasLast    bool
}

func NewWilliamsSignal(log *slog.Logger, will *talive.Williams) *WilliamsSignal {
	return &WilliamsSignal{
		log:       log,
		indicator: will,
	}
}

func (s *WilliamsSignal) Next(kline *binance.Kline) int {
	value := s.indicator.Next(kline)[0]
	signal := s.signal(value)

	s.lastResult = value
	s.hasLast = true
	return signal
}

func (s *WilliamsSignal) Current(kline *binance.Kline) int {
	return s.signal(s.indicator.Current(kline)[0])
}

func (s *WilliamsSignal) signal(value float64) int {
	signal := 0
	if s.indicator.IsWarmedUp() && s.hasLast {
		if value < -80 && value > s.lastResult {
			signal = 1
		}
		if value > -20 && value < s.lastResult {
			signal = -1
		}
	}
	s.log.Debug("signal", "indicator", s.indicator, "signal", signal, "value", value)
	return signal
}

func (s *WilliamsSignal) MaxWarmUp() int {
	return s.indicator.WarmUpPeriod()
}
