package internal

import (
	"log/slog"
	"screener/internal/binance"
	"screener/internal/signal"

	"github.com/MoroZvlg/talive"
)

type Screener struct {
	log        *slog.Logger
	MASignals  []signal.Signaler
	OscSignals []signal.Signaler
}

func NewScreener(log *slog.Logger) *Screener {
	var maSignals []signal.Signaler
	var oscSignals []signal.Signaler

	// ================
	// MA Signals (15)
	// ================

	for _, period := range []int{10, 20, 30, 50, 100, 200} {
		ma, _ := talive.NewSMA(period)
		maSignals = append(maSignals, signal.NewMASignal(log, ma))
	}

	for _, period := range []int{10, 20, 30, 50, 100, 200} {
		ma, _ := talive.NewEMA(period)
		maSignals = append(maSignals, signal.NewMASignal(log, ma))
	}

	hma, _ := talive.NewHMA(9)
	maSignals = append(maSignals, signal.NewMASignal(log, hma))

	vwma, _ := talive.NewVWMA(20)
	maSignals = append(maSignals, signal.NewMASignal(log, vwma))

	ich, _ := talive.NewIchimoku(9, 26, 52, 27)
	maSignals = append(maSignals, signal.NewIchimokuSignal(log, ich))

	// =====================
	// Oscillator Signals (11)
	// =====================

	rsi, _ := talive.NewRSI(14)
	oscSignals = append(oscSignals, signal.NewThresholdTrendSignal(log, rsi, 30, 70))

	stoch, _ := talive.NewStochastic(14, 3, 3)
	oscSignals = append(oscSignals, signal.NewStochasticSignal(log, stoch))

	cci, _ := talive.NewCCI(20)
	cci.WithSource(talive.SourceClose)
	oscSignals = append(oscSignals, signal.NewThresholdTrendSignal(log, cci, -100, 100))

	dmi, _ := talive.NewDMI(14)
	oscSignals = append(oscSignals, signal.NewDMISignal(log, dmi))

	ao, _ := talive.NewAO()
	oscSignals = append(oscSignals, signal.NewAOSignal(log, ao))

	momentum, _ := talive.NewMomentum(10)
	oscSignals = append(oscSignals, signal.NewMomentumSignal(log, momentum))

	macd, _ := talive.NewMACD(12, 26, 9)
	oscSignals = append(oscSignals, signal.NewMACDSignal(log, macd))

	stochRSI, _ := talive.NewStochasticRSI(14, 14, 3, 3)
	stochRSITrend, _ := talive.NewEMA(50)
	oscSignals = append(oscSignals, signal.NewStochRSISignal(log, stochRSI, stochRSITrend))

	williams, _ := talive.NewWilliams(14)
	oscSignals = append(oscSignals, signal.NewWilliamsSignal(log, williams))

	// Screener using bull/bear power separately. Can't use talive indicator here
	bbpEma, _ := talive.NewEMA(13)
	bbpTrend, _ := talive.NewEMA(50)
	oscSignals = append(oscSignals, signal.NewBullBearPowerSignal(log, bbpEma, bbpTrend))

	uo, _ := talive.NewUO(7, 14, 28)
	oscSignals = append(oscSignals, signal.NewThresholdSignal(log, uo, 30, 70, true))

	return &Screener{
		log:        log,
		MASignals:  maSignals,
		OscSignals: oscSignals,
	}
}

func (s *Screener) Next(kline *binance.Kline) float64 {
	maSum := 0.0
	for _, sig := range s.MASignals {
		maSum += float64(sig.Next(kline))
	}
	maRating := maSum / float64(len(s.MASignals))

	oscSum := 0.0
	for _, sig := range s.OscSignals {
		oscSum += float64(sig.Next(kline))
	}
	oscRating := oscSum / float64(len(s.OscSignals))

	total := (maRating + oscRating) / 2
	s.log.Debug("screener", "ma", maRating, "osc", oscRating, "total", total)
	return total
}

func (s *Screener) Current(kline *binance.Kline) float64 {
	maSum := 0.0
	for _, sig := range s.MASignals {
		maSum += float64(sig.Current(kline))
	}
	maRating := maSum / float64(len(s.MASignals))

	oscSum := 0.0
	for _, sig := range s.OscSignals {
		oscSum += float64(sig.Current(kline))
	}
	oscRating := oscSum / float64(len(s.OscSignals))

	total := (maRating + oscRating) / 2
	s.log.Debug("screener", "ma", maRating, "osc", oscRating, "total", total)
	return total
}

func (s *Screener) MaxWarmUp() int {
	result := 0
	for _, sig := range s.MASignals {
		result = max(result, sig.MaxWarmUp())
	}
	for _, sig := range s.OscSignals {
		result = max(result, sig.MaxWarmUp())
	}
	return result
}
