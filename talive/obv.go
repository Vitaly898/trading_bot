package talive

import (
	"fmt"
	"time"
)

// OBV is an On Balance Volume indicator.
type OBV struct {
	AnchorMode Anchor

	valueNumber int
	prevClose   float64
	value       float64
	prevTime    time.Time
	out         []float64
}

// NewOBV creates a new OBV indicator.
func NewOBV() (*OBV, error) {
	return &OBV{
		out: make([]float64, 1),
	}, nil
}

// WithAnchor configures automatic Reset at the start of each Anchor period.
func (o *OBV) WithAnchor(mode Anchor) *OBV {
	o.AnchorMode = mode
	return o
}

func (o *OBV) String() string {
	if o.AnchorMode != AnchorNone {
		return fmt.Sprintf("OBV(anchor=%s)", o.AnchorMode)
	}
	return "OBV"
}

func (o *OBV) Next(candle OHLCV) []float64 {
	ts := candle.Timestamp()
	if o.valueNumber > 0 && anchorChanged(o.prevTime, ts, o.AnchorMode) {
		o.Reset()
	}
	o.prevTime = ts
	o.valueNumber++
	closeV := candle.Close()

	if o.valueNumber == 1 {
		o.prevClose = closeV
		o.out[0] = 0.0
		return o.out
	}

	switch {
	case closeV > o.prevClose:
		o.value += candle.Volume()
	case closeV < o.prevClose:
		o.value -= candle.Volume()
	}

	o.prevClose = closeV
	o.out[0] = o.value
	return o.out
}

func (o *OBV) Current(candle OHLCV) []float64 {
	resetPeek := o.valueNumber > 0 && anchorChanged(o.prevTime, candle.Timestamp(), o.AnchorMode)
	if resetPeek {
		o.out[0] = 0.0
		return o.out
	}

	o.valueNumber++
	if o.IsIdle() {
		o.valueNumber--
		o.out[0] = 0.0
		return o.out
	}

	closeV := candle.Close()
	value := o.value
	switch {
	case closeV > o.prevClose:
		value += candle.Volume()
	case closeV < o.prevClose:
		value -= candle.Volume()
	}
	o.valueNumber--
	o.out[0] = value
	return o.out
}

// Reset clears accumulated state, returning OBV to its initial idle state.
func (o *OBV) Reset() {
	o.valueNumber = 0
	o.prevClose = 0
	o.value = 0
	o.out[0] = 0
}

func (o *OBV) IsIdle() bool {
	return o.valueNumber <= 1
}

func (o *OBV) IdlePeriod() int {
	return 1
}

func (o *OBV) IsWarmedUp() bool {
	return !o.IsIdle()
}

func (o *OBV) WarmUpPeriod() int {
	return o.IdlePeriod()
}
