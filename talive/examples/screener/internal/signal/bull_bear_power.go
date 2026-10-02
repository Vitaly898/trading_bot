package signal

import (
	"log/slog"
	"screener/internal/binance"

	"github.com/MoroZvlg/talive"
)

type BullBearPowerSignal struct {
	log      *slog.Logger
	ema      talive.Scalar
	trendMA  talive.Scalar
	lastBull float64
	lastBear float64
	hasLast  bool
}

func NewBullBearPowerSignal(log *slog.Logger, ema talive.Scalar, trendMA talive.Scalar) *BullBearPowerSignal {
	return &BullBearPowerSignal{
		log:     log,
		ema:     ema,
		trendMA: trendMA,
	}
}

func (s *BullBearPowerSignal) Next(kline *binance.Kline) int {
	emaVal := s.ema.Next(kline)[0]
	trendVal := s.trendMA.Next(kline)[0]

	bullPower := kline.High() - emaVal
	bearPower := kline.Low() - emaVal
	signal := s.signal(bullPower, bearPower, trendVal, kline)

	s.lastBull = bullPower
	s.lastBear = bearPower
	s.hasLast = true
	return signal
}

func (s *BullBearPowerSignal) Current(kline *binance.Kline) int {
	emaVal := s.ema.Current(kline)[0]
	trendVal := s.trendMA.Current(kline)[0]

	return s.signal(kline.High()-emaVal, kline.Low()-emaVal, trendVal, kline)
}

func (s *BullBearPowerSignal) signal(bullPower, bearPower, trendVal float64, kline *binance.Kline) int {
	signal := 0
	if s.ema.IsWarmedUp() && s.trendMA.IsWarmedUp() && s.hasLast {
		if kline.Close() > trendVal && bearPower < 0 && bearPower > s.lastBear {
			signal = 1
		}
		if kline.Close() < trendVal && bullPower > 0 && bullPower < s.lastBull {
			signal = -1
		}
	}
	s.log.Debug("signal", "indicator", "BullBearPower(13)", "signal", signal, "bull", bullPower, "bear", bearPower)
	return signal
}

func (s *BullBearPowerSignal) MaxWarmUp() int {
	return max(s.ema.WarmUpPeriod(), s.trendMA.WarmUpPeriod())
}
