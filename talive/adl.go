package talive

import (
	"fmt"
	"time"
)

// ADL is an Accumulation/Distribution Line indicator.
type ADL struct {
	AnchorMode Anchor

	valueNumber int
	value       float64
	prevTime    time.Time
	out         []float64
}

// NewADL creates a new Accumulation/Distribution Line indicator.
func NewADL() (*ADL, error) {
	return &ADL{
		out: make([]float64, 1),
	}, nil
}

// WithAnchor configures automatic Reset at the start of each Anchor period.
func (adl *ADL) WithAnchor(mode Anchor) *ADL {
	adl.AnchorMode = mode
	return adl
}

func (adl *ADL) String() string {
	if adl.AnchorMode != AnchorNone {
		return fmt.Sprintf("ADL(anchor=%s)", adl.AnchorMode)
	}
	return "ADL"
}

func (adl *ADL) Next(candle OHLCV) []float64 {
	ts := candle.Timestamp()
	if adl.valueNumber > 0 && anchorChanged(adl.prevTime, ts, adl.AnchorMode) {
		adl.Reset()
	}
	adl.prevTime = ts
	adl.valueNumber++

	adl.value += adl.moneyFlowVolume(candle)
	adl.out[0] = adl.value
	return adl.out
}

func (adl *ADL) Current(candle OHLCV) []float64 {
	value := adl.value
	if adl.valueNumber > 0 && anchorChanged(adl.prevTime, candle.Timestamp(), adl.AnchorMode) {
		value = 0
	}

	adl.out[0] = value + adl.moneyFlowVolume(candle)
	return adl.out
}

func (adl *ADL) Reset() {
	adl.valueNumber = 0
	adl.value = 0
	adl.out[0] = 0
}

func (adl *ADL) IsIdle() bool {
	return adl.valueNumber == 0
}

func (adl *ADL) IdlePeriod() int {
	return 0
}

func (adl *ADL) IsWarmedUp() bool {
	return !adl.IsIdle()
}

func (adl *ADL) WarmUpPeriod() int {
	return adl.IdlePeriod()
}

func (adl *ADL) moneyFlowVolume(candle OHLCV) float64 {
	high, low := candle.High(), candle.Low()
	if high == low {
		return 0
	}
	closeV := candle.Close()
	multiplier := (2*closeV - low - high) / (high - low)
	return multiplier * candle.Volume()
}
