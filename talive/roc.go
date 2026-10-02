package talive

import "fmt"

// ROC is a Rate of Change indicator.
type ROC struct {
	Period      int
	SourceFunc  SourceFunc
	valueNumber int
	buffer      *ringBuffer
	out         []float64
}

// NewROC creates a new Rate of Change indicator with the given period.
func NewROC(period int) (*ROC, error) {
	if period < 1 {
		return nil, fmt.Errorf("period should be greater than 0")
	}
	return &ROC{
		Period:     period,
		SourceFunc: SourceClose,
		buffer:     newRingBuffer(period),
		out:        make([]float64, 1),
	}, nil
}

// WithSource sets the price source used to extract values from candles.
func (roc *ROC) WithSource(source SourceFunc) *ROC {
	roc.SourceFunc = source
	return roc
}

func (roc *ROC) String() string {
	return fmt.Sprintf("ROC(%d)", roc.Period)
}

func (roc *ROC) Next(candle OHLCV) []float64 {
	roc.valueNumber++

	value := roc.SourceFunc(candle)
	oldest := roc.buffer.Last()
	roc.buffer.Push(value)

	if roc.IsIdle() {
		return roc.out
	}

	roc.out[0] = (value - oldest) / oldest * 100.0
	return roc.out
}

func (roc *ROC) Current(candle OHLCV) []float64 {
	if roc.IsIdle() {
		return roc.out
	}

	value := roc.SourceFunc(candle)
	oldest := roc.buffer.Last()
	roc.out[0] = (value - oldest) / oldest * 100.0
	return roc.out
}

func (roc *ROC) IsIdle() bool {
	return roc.valueNumber <= roc.Period
}

func (roc *ROC) IdlePeriod() int {
	return roc.Period
}

func (roc *ROC) IsWarmedUp() bool {
	return !roc.IsIdle()
}

func (roc *ROC) WarmUpPeriod() int {
	return roc.IdlePeriod()
}
