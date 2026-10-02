package signal

import (
	"log/slog"
	"screener/internal/binance"

	"github.com/MoroZvlg/talive"
)

type IchimokuSignal struct {
	log *slog.Logger
	ich *talive.Ichimoku
}

func NewIchimokuSignal(log *slog.Logger, ich *talive.Ichimoku) *IchimokuSignal {
	return &IchimokuSignal{
		log: log,
		ich: ich,
	}
}

func (s *IchimokuSignal) Next(kline *binance.Kline) int {
	return s.signal(s.ich.Next(kline), kline)
}

func (s *IchimokuSignal) Current(kline *binance.Kline) int {
	return s.signal(s.ich.Current(kline), kline)
}

func (s *IchimokuSignal) signal(result []float64, kline *binance.Kline) int {
	conv, base, leadA, leadB := result[0], result[1], result[2], result[3]
	signal := 0
	if s.ich.IsWarmedUp() {
		if leadA > leadB && base > leadA && conv > base && kline.Close() > conv {
			signal = 1
		}
		if leadA < leadB && base < leadA && conv < base && kline.Close() < conv {
			signal = -1
		}
	}
	s.log.Debug("signal", "indicator", s.ich, "signal", signal, "conv", conv, "base", base, "leadA", leadA, "leadB", leadB)
	return signal
}

func (s *IchimokuSignal) MaxWarmUp() int {
	return s.ich.WarmUpPeriod()
}
