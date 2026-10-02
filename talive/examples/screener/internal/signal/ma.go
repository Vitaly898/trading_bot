package signal

import (
	"log/slog"
	"screener/internal/binance"

	"github.com/MoroZvlg/talive"
)

type MASignal struct {
	log *slog.Logger
	ma  talive.Indicator
}

func NewMASignal(log *slog.Logger, ma talive.Indicator) *MASignal {
	return &MASignal{log: log, ma: ma}
}

func (s *MASignal) Next(kline *binance.Kline) int {
	return s.signal(s.ma.Next(kline)[0], kline)

}

func (s *MASignal) Current(kline *binance.Kline) int {
	return s.signal(s.ma.Current(kline)[0], kline)
}

func (s *MASignal) signal(maV float64, kline *binance.Kline) int {
	signal := 0
	if s.ma.IsWarmedUp() {
		if maV > kline.Close() {
			signal = -1
		} else if maV < kline.Close() {
			signal = 1
		}
	}
	s.log.Debug("signal", "indicator", s.ma, "signal", signal, "value", maV)
	return signal
}

func (s *MASignal) MaxWarmUp() int {
	return s.ma.WarmUpPeriod()
}
