package talive

import (
	"fmt"
	"math"
)

// ATR is an Average True Range indicator.
type ATR struct {
	Period      int
	valueNumber int

	prevClose float64
	ma        Scalar

	out []float64
}

// NewATR creates a new ATR indicator with the given period.
func NewATR(period int) (*ATR, error) {
	if period < 2 {
		return nil, fmt.Errorf("period should be greater than 1")
	}
	ma, _ := NewSMMA(period)
	return &ATR{
		Period: period,
		ma:     ma,
		out:    make([]float64, 1),
	}, nil
}

// WithMA replaces the internal smoothing method.
func (atr *ATR) WithMA(ma MaType) *ATR {
	atr.ma, _ = ma.New(atr.Period)
	return atr
}

func (atr *ATR) String() string {
	return fmt.Sprintf("ATR(%d)", atr.Period)
}

func (atr *ATR) Next(candle OHLCV) []float64 {
	atr.valueNumber++

	var trueRange float64
	if atr.valueNumber == 1 {
		trueRange = candle.High() - candle.Low()
	} else {
		highLow := candle.High() - candle.Low()
		highPrevClose := math.Abs(candle.High() - atr.prevClose)
		lowPrevClose := math.Abs(candle.Low() - atr.prevClose)
		trueRange = max(highLow, max(highPrevClose, lowPrevClose))
	}

	atr.prevClose = candle.Close()

	atrV := atr.ma.NextVal(trueRange)

	if atr.ma.IsIdle() {
		return atr.out
	}

	atr.out[0] = atrV
	return atr.out
}

func (atr *ATR) Current(candle OHLCV) []float64 {
	if atr.IsIdle() {
		return atr.out
	}

	highLow := candle.High() - candle.Low()
	highPrevClose := math.Abs(candle.High() - atr.prevClose)
	lowPrevClose := math.Abs(candle.Low() - atr.prevClose)
	trueRange := max(highLow, max(highPrevClose, lowPrevClose))

	atr.out[0] = atr.ma.CurrentVal(trueRange)
	return atr.out
}

func (atr *ATR) IsIdle() bool {
	return atr.ma.IsIdle()
}

func (atr *ATR) IdlePeriod() int {
	return atr.ma.IdlePeriod()
}

func (atr *ATR) IsWarmedUp() bool {
	return atr.ma.IsWarmedUp()
}

func (atr *ATR) WarmUpPeriod() int {
	return atr.ma.WarmUpPeriod()
}
