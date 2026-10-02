package signal

import (
	"log/slog"
	"screener/internal/binance"

	"github.com/MoroZvlg/talive"
)

type MACDSignal struct {
	log  *slog.Logger
	macd *talive.MACD
}

func NewMACDSignal(log *slog.Logger, macd *talive.MACD) *MACDSignal {
	return &MACDSignal{
		log:  log,
		macd: macd,
	}
}

func (s *MACDSignal) Next(kline *binance.Kline) int {
	return s.signal(s.macd.Next(kline))
}

func (s *MACDSignal) Current(kline *binance.Kline) int {
	return s.signal(s.macd.Current(kline))
}

func (s *MACDSignal) signal(result []float64) int {
	macdLine, signalLine := result[0], result[1]
	signal := 0
	if s.macd.IsWarmedUp() {
		if macdLine > signalLine {
			signal = 1
		} else if macdLine < signalLine {
			signal = -1
		}
	}
	s.log.Debug("signal", "indicator", s.macd, "signal", signal, "macd", macdLine, "signal_line", signalLine)
	return signal
}

func (s *MACDSignal) MaxWarmUp() int {
	return s.macd.WarmUpPeriod()
}
