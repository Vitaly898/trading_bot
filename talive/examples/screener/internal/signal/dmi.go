package signal

import (
	"log/slog"
	"screener/internal/binance"

	"github.com/MoroZvlg/talive"
)

type DMISignal struct {
	log     *slog.Logger
	dmi     *talive.DMI
	lastADX float64
	hasLast bool
}

func NewDMISignal(log *slog.Logger, dmi *talive.DMI) *DMISignal {
	return &DMISignal{log: log, dmi: dmi}
}

func (s *DMISignal) Next(kline *binance.Kline) int {
	result := s.dmi.Next(kline)
	signal := s.signal(result)

	s.lastADX = result[0]
	s.hasLast = true
	return signal
}

func (s *DMISignal) Current(kline *binance.Kline) int {
	return s.signal(s.dmi.Current(kline))
}

func (s *DMISignal) signal(result []float64) int {
	adx, diPlus, diMinus := result[0], result[1], result[2]
	signal := 0

	if s.dmi.IsWarmedUp() && s.hasLast {
		if diPlus > diMinus && adx > 20 && adx > s.lastADX {
			signal = 1
		}
		if diPlus < diMinus && adx > 20 && adx > s.lastADX {
			signal = -1
		}
	}
	s.log.Debug("signal", "indicator", s.dmi, "signal", signal, "adx", adx, "di+", diPlus, "di-", diMinus)
	return signal
}

func (s *DMISignal) MaxWarmUp() int {
	return s.dmi.WarmUpPeriod()
}
