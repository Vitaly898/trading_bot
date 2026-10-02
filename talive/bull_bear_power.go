package talive

import "fmt"

// BullBearPower is an Elder's Bull Bear Power indicator.
type BullBearPower struct {
	Period     int
	SourceFunc SourceFunc

	ma  Scalar
	out []float64
}

// NewBullBearPower creates a new Bull Bear Power indicator with the given period.
func NewBullBearPower(period int) (*BullBearPower, error) {
	if period < 2 {
		return nil, fmt.Errorf("period should be greater than 1")
	}
	ma, _ := NewEMA(period)
	return &BullBearPower{
		Period:     period,
		SourceFunc: SourceClose,
		ma:         ma,
		out:        make([]float64, 1),
	}, nil
}

// WithSource sets the price source used to extract values from candles.
func (bbp *BullBearPower) WithSource(source SourceFunc) *BullBearPower {
	bbp.SourceFunc = source
	return bbp
}

func (bbp *BullBearPower) String() string {
	return fmt.Sprintf("BullBearPower(%d)", bbp.Period)
}

func (bbp *BullBearPower) Next(candle OHLCV) []float64 {
	emaVal := bbp.ma.NextVal(bbp.SourceFunc(candle))

	if bbp.ma.IsIdle() {
		return bbp.out
	}

	bbp.out[0] = candle.High() + candle.Low() - 2*emaVal
	return bbp.out
}

func (bbp *BullBearPower) Current(candle OHLCV) []float64 {
	if bbp.IsIdle() {
		return bbp.out
	}

	emaVal := bbp.ma.CurrentVal(bbp.SourceFunc(candle))
	bbp.out[0] = candle.High() + candle.Low() - 2*emaVal
	return bbp.out
}

func (bbp *BullBearPower) IsIdle() bool {
	return bbp.ma.IsIdle()
}

func (bbp *BullBearPower) IdlePeriod() int {
	return bbp.ma.IdlePeriod()
}

func (bbp *BullBearPower) IsWarmedUp() bool {
	return bbp.ma.IsWarmedUp()
}

func (bbp *BullBearPower) WarmUpPeriod() int {
	return bbp.ma.WarmUpPeriod()
}
