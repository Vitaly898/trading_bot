package talive

import (
	"fmt"
	"math"
)

// DMI is a Directional Movement Index indicator.
// It returns ADX, +DI and -DI values.
type DMI struct {
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

// NewDMI creates a new DMI indicator with the given period.
func NewDMI(period int) (*DMI, error) {
	plusDMMA, _ := NewSMMA(period)
	minusDMMA, _ := NewSMMA(period)
	trMA, _ := NewSMMA(period)
	adxMA, _ := NewSMMA(period)
	return &DMI{
		Period:    period,
		plusDMMA:  plusDMMA,
		minusDMMA: minusDMMA,
		trMA:      trMA,
		adxMA:     adxMA,
		out:       make([]float64, 3),
	}, nil
}

// WithMA replaces the internal smoothing method used for all DMI components.
func (dmi *DMI) WithMA(ma MaType) *DMI {
	dmi.plusDMMA, _ = ma.New(dmi.Period)
	dmi.minusDMMA, _ = ma.New(dmi.Period)
	dmi.trMA, _ = ma.New(dmi.Period)
	dmi.adxMA, _ = ma.New(dmi.Period)
	return dmi
}

func (dmi *DMI) String() string {
	return fmt.Sprintf("DMI(%d)", dmi.Period)
}

func (dmi *DMI) Next(candle OHLCV) []float64 {
	dmi.valueNumber++

	if dmi.valueNumber == 1 {
		dmi.prevHigh = candle.High()
		dmi.prevLow = candle.Low()
		dmi.prevClose = candle.Close()
		return dmi.out
	}

	plusDM, minusDM, tr := dmi.computeDMTR(candle)

	dmi.prevHigh = candle.High()
	dmi.prevLow = candle.Low()
	dmi.prevClose = candle.Close()

	sPlusDM := dmi.plusDMMA.NextVal(plusDM)
	sMinusDM := dmi.minusDMMA.NextVal(minusDM)
	sTR := dmi.trMA.NextVal(tr)

	if dmi.trMA.IsIdle() {
		return dmi.out
	}

	plusDI := 100 * sPlusDM / sTR
	minusDI := 100 * sMinusDM / sTR

	dmi.out[1] = plusDI
	dmi.out[2] = minusDI

	dx := 100 * math.Abs(plusDI-minusDI) / (plusDI + minusDI)
	adx := dmi.adxMA.NextVal(dx)

	if dmi.adxMA.IsIdle() {
		return dmi.out
	}

	dmi.out[0] = adx
	return dmi.out
}

func (dmi *DMI) Current(candle OHLCV) []float64 {
	if dmi.IsIdle() {
		return dmi.out
	}

	plusDM, minusDM, tr := dmi.computeDMTR(candle)

	sPlusDM := dmi.plusDMMA.CurrentVal(plusDM)
	sMinusDM := dmi.minusDMMA.CurrentVal(minusDM)
	sTR := dmi.trMA.CurrentVal(tr)

	plusDI := 100 * sPlusDM / sTR
	minusDI := 100 * sMinusDM / sTR

	dmi.out[1] = plusDI
	dmi.out[2] = minusDI

	dx := 100 * math.Abs(plusDI-minusDI) / (plusDI + minusDI)
	dmi.out[0] = dmi.adxMA.CurrentVal(dx)
	return dmi.out
}

func (dmi *DMI) computeDMTR(candle OHLCV) (plusDM, minusDM, tr float64) {
	upMove := candle.High() - dmi.prevHigh
	downMove := dmi.prevLow - candle.Low()

	if upMove > downMove && upMove > 0 {
		plusDM = upMove
	}
	if downMove > upMove && downMove > 0 {
		minusDM = downMove
	}

	highLow := candle.High() - candle.Low()
	highPrevClose := math.Abs(candle.High() - dmi.prevClose)
	lowPrevClose := math.Abs(candle.Low() - dmi.prevClose)
	tr = max(highLow, max(highPrevClose, lowPrevClose))
	return plusDM, minusDM, tr
}

func (dmi *DMI) IsIdle() bool {
	return dmi.valueNumber <= dmi.Period
}

func (dmi *DMI) IdlePeriod() int {
	return dmi.Period
}

func (dmi *DMI) IsWarmedUp() bool {
	return dmi.valueNumber > dmi.WarmUpPeriod()
}

func (dmi *DMI) WarmUpPeriod() int {
	return 2*dmi.Period - 1 + dmi.Period*9
}
