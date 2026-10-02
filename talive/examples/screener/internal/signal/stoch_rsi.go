package signal

import (
	"log/slog"
	"screener/internal/binance"

	"github.com/MoroZvlg/talive"
)

type StochRSISignal struct {
	log      *slog.Logger
	stochRSI talive.Indicator
	trendMA  talive.Scalar
}

func NewStochRSISignal(log *slog.Logger, stochRSI talive.Indicator, trendMA talive.Scalar) *StochRSISignal {
	return &StochRSISignal{
		log:      log,
		stochRSI: stochRSI,
		trendMA:  trendMA,
	}
}

func (s *StochRSISignal) Next(kline *binance.Kline) int {
	return s.signal(s.stochRSI.Next(kline), s.trendMA.Next(kline)[0], kline)
}

func (s *StochRSISignal) Current(kline *binance.Kline) int {
	return s.signal(s.stochRSI.Current(kline), s.trendMA.Current(kline)[0], kline)
}

func (s *StochRSISignal) signal(result []float64, priceAvg float64, kline *binance.Kline) int {
	k, d := result[0], result[1]
	signal := 0

	if s.stochRSI.IsWarmedUp() && s.trendMA.IsWarmedUp() {
		if kline.Close() < priceAvg && k < 20 && d < 20 && k > d {
			signal = 1
		}
		if kline.Close() > priceAvg && k > 80 && d > 80 && k < d {
			signal = -1
		}
	}

	s.log.Debug("signal", "indicator", s.stochRSI, "signal", signal, "k", k, "d", d)
	return signal
}

func (s *StochRSISignal) MaxWarmUp() int {
	return max(s.stochRSI.WarmUpPeriod(), s.trendMA.WarmUpPeriod())
}
