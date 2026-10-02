package talive

import (
	"fmt"
	"math"
)

// HMA is a Hull Moving Average indicator.
type HMA struct {
	Period      int
	SourceFunc  SourceFunc
	valueNumber int
	halfMA      Scalar
	fullMA      Scalar
	sqrtMA      Scalar
	out         []float64
}

// NewHMA creates a new Hull Moving Average indicator with the given period.
func NewHMA(period int) (*HMA, error) {
	if period < 2 {
		return nil, fmt.Errorf("period should be greater than 1")
	}
	halfPeriod := max(period/2, 1)
	sqrtPeriod := max(int(math.Floor(math.Sqrt(float64(period)))), 1)
	halfMA, _ := NewWMA(halfPeriod)
	fullMA, _ := NewWMA(period)
	sqrtMA, _ := NewWMA(sqrtPeriod)
	return &HMA{
		Period:     period,
		SourceFunc: SourceClose,
		halfMA:     halfMA,
		fullMA:     fullMA,
		sqrtMA:     sqrtMA,
		out:        make([]float64, 1),
	}, nil
}

// WithSource sets the price source used to extract values from candles.
func (hma *HMA) WithSource(source SourceFunc) *HMA {
	hma.SourceFunc = source
	return hma
}

func (hma *HMA) String() string {
	return fmt.Sprintf("HullMA(%d)", hma.Period)
}

func (hma *HMA) Next(candle OHLCV) []float64 {
	hma.valueNumber++
	value := hma.SourceFunc(candle)
	halfVal := hma.halfMA.NextVal(value)
	fullVal := hma.fullMA.NextVal(value)

	if hma.fullMA.IsIdle() {
		return hma.out
	}

	diff := 2*halfVal - fullVal
	hmaV := hma.sqrtMA.NextVal(diff)

	if hma.sqrtMA.IsIdle() {
		return hma.out
	}

	hma.out[0] = hmaV
	return hma.out
}

func (hma *HMA) Current(candle OHLCV) []float64 {
	if hma.IsIdle() {
		return hma.out
	}

	value := hma.SourceFunc(candle)
	halfVal := hma.halfMA.CurrentVal(value)
	fullVal := hma.fullMA.CurrentVal(value)
	diff := 2*halfVal - fullVal
	hma.out[0] = hma.sqrtMA.CurrentVal(diff)
	return hma.out
}

func (hma *HMA) IsIdle() bool {
	return hma.sqrtMA.IsIdle()
}

func (hma *HMA) IdlePeriod() int {
	return hma.fullMA.IdlePeriod() + hma.sqrtMA.IdlePeriod()
}

func (hma *HMA) IsWarmedUp() bool {
	return !hma.IsIdle()
}

func (hma *HMA) WarmUpPeriod() int {
	return hma.IdlePeriod()
}
