package talive

import "fmt"

// Williams is a Williams %R indicator.
type Williams struct {
	Period      int
	SourceFunc  SourceFunc
	valueNumber int
	lowest      *ringBuffer
	highest     *ringBuffer
	out         []float64
}

// NewWilliams creates a new Williams %R indicator with the given period.
func NewWilliams(period int) (*Williams, error) {
	return &Williams{
		Period:      period,
		SourceFunc:  SourceClose,
		valueNumber: 0,
		lowest:      newRingBuffer(period),
		highest:     newRingBuffer(period),
		out:         make([]float64, 1),
	}, nil
}

// WithSource sets the price source used to extract values from candles.
func (will *Williams) WithSource(source SourceFunc) *Williams {
	will.SourceFunc = source
	return will
}

func (will *Williams) String() string {
	return fmt.Sprintf("Williams(%d)", will.Period)
}

func (will *Williams) Next(candle OHLCV) []float64 {
	will.valueNumber++

	will.lowest.Push(candle.Low())
	will.highest.Push(candle.High())

	if will.valueNumber < will.Period {
		return will.out
	}

	highestV := will.highest.Max()
	will.out[0] = (highestV - will.SourceFunc(candle)) / (highestV - will.lowest.Min()) * -100.0

	return will.out
}

func (will *Williams) Current(candle OHLCV) []float64 {
	if will.valueNumber < will.Period {
		return will.out
	}

	lowestV := min(will.lowest.MinExceptLast(), candle.Low())
	highestV := max(will.highest.MaxExceptLast(), candle.High())

	will.out[0] = (highestV - will.SourceFunc(candle)) / (highestV - lowestV) * -100.0

	return will.out
}

func (will *Williams) IsIdle() bool {
	return will.valueNumber < will.Period
}

func (will *Williams) IsWarmedUp() bool {
	return will.valueNumber > will.WarmUpPeriod()
}

func (will *Williams) IdlePeriod() int {
	return will.Period - 1
}

func (will *Williams) WarmUpPeriod() int {
	return will.IdlePeriod()
}
