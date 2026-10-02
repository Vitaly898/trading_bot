package talive

import (
	"fmt"
	"time"
)

// PivotType defines the formula variant used by Pivot Points.
type PivotType int

// Supported pivot formulas.
const (
	PivotStandard PivotType = iota
	PivotFibonacci
	PivotCamarilla
)

func (pt PivotType) String() string {
	switch pt {
	case PivotStandard:
		return "Standard"
	case PivotFibonacci:
		return "Fibonacci"
	case PivotCamarilla:
		return "Camarilla"
	}
	return "UnknownPivot"
}

// outSize returns the number of output values for this pivot method.
// Standard/Fibonacci produce 7 levels (P + 3R + 3S); Camarilla produces 9 (P + 4R + 4S).
func (pt PivotType) outSize() int {
	if pt == PivotCamarilla {
		return 9
	}
	return 7
}

// PivotPoints is a previous-period anchored Pivot Points indicator.
//
// Output layout depends on PivotType:
//   - Standard/Fibonacci: [P, R1, R2, R3, S1, S2, S3]
//   - Camarilla:          [P, R1, R2, R3, R4, S1, S2, S3, S4]
//
// Values remain constant throughout the active anchor period and update only
// when a new anchor period starts.
type PivotPoints struct {
	AnchorMode Anchor
	PivotType  PivotType

	prevTime    time.Time
	valueNumber int // candles aggregated into the current window; reset on anchor flip

	windowHigh    float64
	windowLow     float64
	windowClose   float64
	hasPrevPeriod bool // a prior anchor period has completed, so levels are published

	out []float64
}

// NewPivotPoints creates a new Pivot Points indicator with Daily anchor and
// Standard formula.
func NewPivotPoints() (*PivotPoints, error) {
	return &PivotPoints{
		AnchorMode: AnchorDaily,
		PivotType:  PivotStandard,
		out:        make([]float64, PivotStandard.outSize()),
	}, nil
}

// WithAnchor configures which completed period drives the next period's levels.
func (pp *PivotPoints) WithAnchor(mode Anchor) *PivotPoints {
	pp.AnchorMode = mode
	return pp
}

// WithType configures the pivot formula variant. Reallocates the output slice
// when the new method has a different output size (Standard/Fib=7, Camarilla=9).
func (pp *PivotPoints) WithType(pivotType PivotType) *PivotPoints {
	pp.PivotType = pivotType
	if sz := pivotType.outSize(); len(pp.out) != sz {
		pp.out = make([]float64, sz)
	}
	return pp
}

func (pp *PivotPoints) String() string {
	return fmt.Sprintf("PivotPoints(type=%s,anchor=%s)", pp.PivotType, pp.AnchorMode)
}

func (pp *PivotPoints) Next(candle OHLCV) []float64 {
	ts := candle.Timestamp()
	if pp.valueNumber > 0 && anchorChanged(pp.prevTime, ts, pp.AnchorMode) {
		pp.Reset()
	}
	pp.prevTime = ts
	if pp.valueNumber == 0 {
		pp.windowHigh = candle.High()
		pp.windowLow = candle.Low()
		pp.windowClose = candle.Close()
	} else {
		pp.windowHigh = max(pp.windowHigh, candle.High())
		pp.windowLow = min(pp.windowLow, candle.Low())
		pp.windowClose = candle.Close()
	}
	pp.valueNumber++
	return pp.out
}

func (pp *PivotPoints) Current(candle OHLCV) []float64 {
	if pp.valueNumber == 0 {
		return pp.out
	}
	if anchorChanged(pp.prevTime, candle.Timestamp(), pp.AnchorMode) {
		pp.writeLevels()
	}
	return pp.out
}

func (pp *PivotPoints) Reset() {
	if pp.valueNumber > 0 {
		pp.writeLevels()
		pp.hasPrevPeriod = true
	}
	pp.valueNumber = 0
}

func (pp *PivotPoints) IsIdle() bool {
	return !pp.hasPrevPeriod
}

func (pp *PivotPoints) IdlePeriod() int {
	return 0
}

func (pp *PivotPoints) IsWarmedUp() bool {
	return !pp.IsIdle()
}

func (pp *PivotPoints) WarmUpPeriod() int {
	return 0
}

func (pp *PivotPoints) writeLevels() {
	high, low, closeV := pp.windowHigh, pp.windowLow, pp.windowClose
	p := (high + low + closeV) / 3.0
	rng := high - low
	switch pp.PivotType {
	case PivotStandard:
		pp.out[0] = p
		pp.out[1] = 2*p - low
		pp.out[2] = p + rng
		pp.out[3] = high + 2*(p-low)
		pp.out[4] = 2*p - high
		pp.out[5] = p - rng
		pp.out[6] = low - 2*(high-p)
	case PivotFibonacci:
		pp.out[0] = p
		pp.out[1] = p + 0.382*rng
		pp.out[2] = p + 0.618*rng
		pp.out[3] = p + 1.000*rng
		pp.out[4] = p - 0.382*rng
		pp.out[5] = p - 0.618*rng
		pp.out[6] = p - 1.000*rng
	case PivotCamarilla:
		pp.out[0] = p
		pp.out[1] = closeV + rng*1.1/12
		pp.out[2] = closeV + rng*1.1/6
		pp.out[3] = closeV + rng*1.1/4
		pp.out[4] = closeV + rng*1.1/2
		pp.out[5] = closeV - rng*1.1/12
		pp.out[6] = closeV - rng*1.1/6
		pp.out[7] = closeV - rng*1.1/4
		pp.out[8] = closeV - rng*1.1/2
	}
}
