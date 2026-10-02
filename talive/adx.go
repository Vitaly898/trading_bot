package talive

import (
	"fmt"
	"math"
)

// ADX is an Average Directional Movement Index indicator.
type ADX struct {
	Period      int
	valueNumber int

	prevHigh  float64
	prevLow   float64
	prevClose float64

	plusDMMA  Scalar
	minusDMMA Scalar
	trMA      Scalar
	adxMA     Scalar

	out []float64
}

// NewADX creates a new ADX indicator with the given period.
func NewADX(period int) (*ADX, error) {
	plusDMMA, _ := NewSMMA(period)
	minusDMMA, _ := NewSMMA(period)
	trMA, _ := NewSMMA(period)
	adxMA, _ := NewSMMA(period)
	return &ADX{
		Period:    period,
		plusDMMA:  plusDMMA,
		minusDMMA: minusDMMA,
		trMA:      trMA,
		adxMA:     adxMA,
		out:       make([]float64, 1),
	}, nil
}

// WithMA replaces the internal smoothing method used for all ADX components.
func (adx *ADX) WithMA(ma MaType) *ADX {
	adx.plusDMMA, _ = ma.New(adx.Period)
	adx.minusDMMA, _ = ma.New(adx.Period)
	adx.trMA, _ = ma.New(adx.Period)
	adx.adxMA, _ = ma.New(adx.Period)
	return adx
}

func (adx *ADX) String() string {
	return fmt.Sprintf("ADX(%d)", adx.Period)
}

func (adx *ADX) Next(candle OHLCV) []float64 {
	adx.valueNumber++

	if adx.valueNumber == 1 {
		adx.prevHigh = candle.High()
		adx.prevLow = candle.Low()
		adx.prevClose = candle.Close()
		return adx.out
	}

	plusDM, minusDM, tr := adx.computeDMTR(candle)

	adx.prevHigh = candle.High()
	adx.prevLow = candle.Low()
	adx.prevClose = candle.Close()

	sPlusDM := adx.plusDMMA.NextVal(plusDM)
	sMinusDM := adx.minusDMMA.NextVal(minusDM)
	sTR := adx.trMA.NextVal(tr)

	if adx.trMA.IsIdle() {
		return adx.out
	}

	plusDI := 100 * sPlusDM / sTR
	minusDI := 100 * sMinusDM / sTR

	dx := 100 * math.Abs(plusDI-minusDI) / (plusDI + minusDI)
	adxV := adx.adxMA.NextVal(dx)

	if adx.adxMA.IsIdle() {
		return adx.out
	}

	adx.out[0] = adxV
	return adx.out
}

func (adx *ADX) Current(candle OHLCV) []float64 {
	if adx.IsIdle() {
		return adx.out
	}

	plusDM, minusDM, tr := adx.computeDMTR(candle)

	sPlusDM := adx.plusDMMA.CurrentVal(plusDM)
	sMinusDM := adx.minusDMMA.CurrentVal(minusDM)
	sTR := adx.trMA.CurrentVal(tr)

	plusDI := 100 * sPlusDM / sTR
	minusDI := 100 * sMinusDM / sTR

	dx := 100 * math.Abs(plusDI-minusDI) / (plusDI + minusDI)
	adx.out[0] = adx.adxMA.CurrentVal(dx)
	return adx.out
}

func (adx *ADX) computeDMTR(candle OHLCV) (plusDM, minusDM, tr float64) {
	upMove := candle.High() - adx.prevHigh
	downMove := adx.prevLow - candle.Low()

	if upMove > downMove && upMove > 0 {
		plusDM = upMove
	}
	if downMove > upMove && downMove > 0 {
		minusDM = downMove
	}

	highLow := candle.High() - candle.Low()
	highPrevClose := math.Abs(candle.High() - adx.prevClose)
	lowPrevClose := math.Abs(candle.Low() - adx.prevClose)
	tr = max(highLow, max(highPrevClose, lowPrevClose))
	return plusDM, minusDM, tr
}

func (adx *ADX) IsIdle() bool {
	return adx.valueNumber < 2*adx.Period
}

func (adx *ADX) IdlePeriod() int {
	return 2*adx.Period - 1
}

func (adx *ADX) IsWarmedUp() bool {
	return adx.valueNumber > adx.WarmUpPeriod()
}

func (adx *ADX) WarmUpPeriod() int {
	return adx.IdlePeriod() + adx.Period*9
}
