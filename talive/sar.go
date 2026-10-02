package talive

import (
	"fmt"
	"math"
)

// SAR is a Parabolic SAR indicator.
type SAR struct {
	AfStart     float64
	AfIncrement float64
	AfMax       float64

	valueNumber  int
	inUpTrend    bool
	prevAf       float64 // acceleration factor
	prevEp       float64 // extreme price
	prevSar      float64
	prevHigh     float64
	prevLow      float64
	prevPrevHigh float64
	prevPrevLow  float64
	prevClose    float64
	out          []float64
}

// NewSAR creates a new Parabolic SAR indicator.
func NewSAR(start, increment, maxAF float64) (*SAR, error) {
	return &SAR{
		AfStart:     start,
		AfIncrement: increment,
		AfMax:       maxAF,
		out:         make([]float64, 1),
	}, nil
}

func (sar *SAR) String() string {
	return fmt.Sprintf("SAR(%.2f,%.2f,%.2f)", sar.AfStart, sar.AfIncrement, sar.AfMax)
}

func (sar *SAR) Next(candle OHLCV) []float64 {
	sar.valueNumber++

	if sar.valueNumber == 1 {
		sar.prevHigh = candle.High()
		sar.prevLow = candle.Low()
		sar.prevClose = candle.Close()

		return sar.out
	}

	var currSar float64
	ep := sar.prevEp
	af := sar.prevAf

	if sar.valueNumber == 2 {
		if candle.Close() > sar.prevClose {
			sar.inUpTrend = true
			ep = math.Max(sar.prevHigh, candle.High())
			currSar = math.Min(sar.prevLow, candle.Low())
		} else {
			sar.inUpTrend = false
			ep = math.Min(sar.prevLow, candle.Low())
			currSar = math.Max(sar.prevHigh, candle.High())
		}
		af = sar.AfStart
		sar.out[0] = currSar

		sar.prevPrevHigh = sar.prevHigh
		sar.prevPrevLow = sar.prevLow
		sar.prevEp = ep
		sar.prevSar = currSar
		sar.prevAf = af
		sar.prevHigh = candle.High()
		sar.prevLow = candle.Low()
		sar.prevClose = candle.Close()

		return sar.out
	}

	currSar = sar.prevSar + sar.prevAf*(sar.prevEp-sar.prevSar)

	if sar.inUpTrend {
		if candle.High() > sar.prevEp {
			ep = candle.High()
			af = math.Min(sar.prevAf+sar.AfIncrement, sar.AfMax)
		}
		if currSar > candle.Low() {
			sar.inUpTrend = false
			currSar = ep
			ep = candle.Low()
			af = sar.AfStart
		} else {
			currSar = math.Min(currSar, math.Min(sar.prevLow, sar.prevPrevLow))
		}
	} else {
		if candle.Low() < sar.prevEp {
			ep = candle.Low()
			af = math.Min(sar.prevAf+sar.AfIncrement, sar.AfMax)
		}
		if currSar < candle.High() {
			sar.inUpTrend = true
			currSar = ep
			ep = candle.High()
			af = sar.AfStart
		} else {
			currSar = math.Max(currSar, math.Max(sar.prevHigh, sar.prevPrevHigh))
		}
	}

	sar.out[0] = currSar

	sar.prevSar = currSar
	sar.prevAf = af
	sar.prevEp = ep
	sar.prevPrevHigh = sar.prevHigh
	sar.prevPrevLow = sar.prevLow
	sar.prevHigh = candle.High()
	sar.prevLow = candle.Low()
	sar.prevClose = candle.Close()

	return sar.out
}

func (sar *SAR) Current(candle OHLCV) []float64 {
	if sar.IsIdle() {
		return sar.out
	}

	currSar := sar.prevSar + sar.prevAf*(sar.prevEp-sar.prevSar)
	ep := sar.prevEp

	if sar.inUpTrend {
		if candle.High() > ep {
			ep = candle.High()
		}
		if currSar > candle.Low() {
			currSar = ep
		} else {
			currSar = math.Min(currSar, math.Min(sar.prevLow, sar.prevPrevLow))
		}
	} else {
		if candle.Low() < ep {
			ep = candle.Low()
		}
		if currSar < candle.High() {
			currSar = ep
		} else {
			currSar = math.Max(currSar, math.Max(sar.prevHigh, sar.prevPrevHigh))
		}
	}

	sar.out[0] = currSar
	return sar.out
}

func (sar *SAR) IsIdle() bool {
	return sar.valueNumber <= sar.IdlePeriod()
}

func (sar *SAR) IdlePeriod() int {
	return 1
}

func (sar *SAR) IsWarmedUp() bool {
	return sar.valueNumber > sar.WarmUpPeriod()
}

// WarmUpPeriod returns a conservative warmup estimate.
// SAR warmup depends on data (trend length), not on formula parametersar.
// Bigger AfStart and AfIncrement reduce warmup (faster trend flips), but the dependency is not linear.
func (sar *SAR) WarmUpPeriod() int {
	return 100
}
