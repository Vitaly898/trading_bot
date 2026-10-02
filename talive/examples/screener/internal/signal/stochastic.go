package signal

import (
	"log/slog"
	"screener/internal/binance"

	"github.com/MoroZvlg/talive"
)

type StochasticSignal struct {
	log   *slog.Logger
	stoch talive.Indicator
}

func NewStochasticSignal(log *slog.Logger, stoch talive.Indicator) *StochasticSignal {
	return &StochasticSignal{
		log:   log,
		stoch: stoch,
	}
}

func (s *StochasticSignal) Next(kline *binance.Kline) int {
	return s.signal(s.stoch.Next(kline))
}

func (s *StochasticSignal) Current(kline *binance.Kline) int {
	return s.signal(s.stoch.Current(kline))
}

func (s *StochasticSignal) signal(result []float64) int {
	k, d := result[0], result[1]
	signal := 0
	if s.stoch.IsWarmedUp() {
		if k < 20 && d < 20 && k > d {
			signal = 1
		}
		if k > 80 && d > 80 && k < d {
			signal = -1
		}
	}
	s.log.Debug("signal", "indicator", s.stoch, "signal", signal, "k", k, "d", d)
	return signal
}

func (s *StochasticSignal) MaxWarmUp() int {
	return s.stoch.WarmUpPeriod()
}
