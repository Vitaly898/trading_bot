package signal

import (
	"log/slog"
	"screener/internal/binance"

	"github.com/MoroZvlg/talive"
)

type AOSignal struct {
	log       *slog.Logger
	indicator *talive.AO
	prev      float64
	prevPrev  float64
	count     int
}

func NewAOSignal(log *slog.Logger, indicator *talive.AO) *AOSignal {
	return &AOSignal{
		log:       log,
		indicator: indicator,
	}
}

func (s *AOSignal) Next(kline *binance.Kline) int {
	ao := s.indicator.Next(kline)[0]
	signal := s.signal(ao)

	s.prevPrev = s.prev
	s.prev = ao
	s.count++
	return signal
}

func (s *AOSignal) Current(kline *binance.Kline) int {
	return s.signal(s.indicator.Current(kline)[0])
}

func (s *AOSignal) signal(ao float64) int {
	signal := 0
	if s.indicator.IsWarmedUp() && s.count >= 2 {
		crossover := ao > 0 && s.prev <= 0
		crossunder := ao < 0 && s.prev >= 0
		saucerBull := ao > 0 && s.prev > 0 && ao > s.prev && s.prevPrev > s.prev
		saucerBear := ao < 0 && s.prev < 0 && ao < s.prev && s.prevPrev < s.prev

		if crossover || saucerBull {
			signal = 1
		}
		if crossunder || saucerBear {
			signal = -1
		}
	}
	s.log.Debug("signal", "indicator", s.indicator, "signal", signal, "value", ao)
	return signal
}

func (s *AOSignal) MaxWarmUp() int {
	return s.indicator.WarmUpPeriod()
}
