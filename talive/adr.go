package talive

import "fmt"

// ADR is an Average Daily Range indicator.
type ADR struct {
	Period int

	ma  Scalar
	out []float64
}

// NewADR creates a new ADR indicator with the given period.
func NewADR(period int) (*ADR, error) {
	if period < 1 {
		return nil, fmt.Errorf("period should be positive")
	}
	ma, _ := NewSMA(period)
	return &ADR{
		Period: period,
		ma:     ma,
		out:    make([]float64, 1),
	}, nil
}

// WithMA replaces the internal smoothing method.
func (adr *ADR) WithMA(ma MaType) *ADR {
	adr.ma, _ = ma.New(adr.Period)
	return adr
}

func (adr *ADR) String() string {
	return fmt.Sprintf("ADR(%d)", adr.Period)
}

func (adr *ADR) Next(candle OHLCV) []float64 {
	adr.out[0] = adr.ma.NextVal(candle.High() - candle.Low())
	return adr.out
}

func (adr *ADR) Current(candle OHLCV) []float64 {
	if adr.IsIdle() {
		return adr.out
	}
	adr.out[0] = adr.ma.CurrentVal(candle.High() - candle.Low())
	return adr.out
}

func (adr *ADR) IsIdle() bool {
	return adr.ma.IsIdle()
}

func (adr *ADR) IdlePeriod() int {
	return adr.ma.IdlePeriod()
}

func (adr *ADR) IsWarmedUp() bool {
	return adr.ma.IsWarmedUp()
}

func (adr *ADR) WarmUpPeriod() int {
	return adr.ma.WarmUpPeriod()
}
