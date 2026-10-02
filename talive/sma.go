package talive

import "fmt"

// SMA is a Simple Moving Average indicator.
type SMA struct {
	Period      int
	SourceFunc  SourceFunc
	valueNumber int
	buffer      *ringBuffer
	out         []float64
}

// NewSMA creates a new SMA indicator with the given period.
func NewSMA(period int) (*SMA, error) {
	return &SMA{
		Period:      period,
		SourceFunc:  SourceClose,
		valueNumber: 0,
		buffer:      newRingBuffer(period),
		out:         make([]float64, 1),
	}, nil
}

// WithSource sets the price source used to extract values from candles.
func (sma *SMA) WithSource(source SourceFunc) *SMA {
	sma.SourceFunc = source
	return sma
}

func (sma *SMA) String() string {
	return fmt.Sprintf("SMA(%d)", sma.Period)
}

func (sma *SMA) NextVal(value float64) float64 {
	sma.buffer.Push(value)
	sma.valueNumber++
	if sma.IsIdle() {
		return 0.0
	}
	return sma.buffer.Sum / float64(sma.Period)
}

func (sma *SMA) CurrentVal(value float64) float64 {
	if sma.IsIdle() {
		return 0.0
	}
	return (sma.buffer.SumExceptLast() + value) / float64(sma.Period)
}

func (sma *SMA) Next(candle OHLCV) []float64 {
	sma.out[0] = sma.NextVal(sma.SourceFunc(candle))
	return sma.out
}

func (sma *SMA) Current(candle OHLCV) []float64 {
	sma.out[0] = sma.CurrentVal(sma.SourceFunc(candle))
	return sma.out
}

func (sma *SMA) IsIdle() bool {
	return sma.valueNumber < sma.Period
}

func (sma *SMA) IdlePeriod() int {
	return sma.Period - 1
}

func (sma *SMA) IsWarmedUp() bool {
	return !sma.IsIdle()
}

func (sma *SMA) WarmUpPeriod() int {
	return sma.IdlePeriod()
}
