package talive

import (
	"fmt"
	"math"
)

// StdDev is a Standard Deviation indicator.
type StdDev struct {
	Period     int
	Deviation  float64
	SourceFunc SourceFunc
	variance   *Variance
	out        []float64
}

// NewStdDev creates a new Standard Deviation indicator.
func NewStdDev(period int, deviation float64) (*StdDev, error) {
	// TODO: add validations
	variance, err := NewVariance(period)
	if err != nil {
		return nil, err
	}
	return &StdDev{
		Period:     period,
		Deviation:  deviation,
		SourceFunc: SourceClose,
		variance:   variance,
		out:        make([]float64, 1),
	}, nil
}

// WithSource sets the price source used to extract values from candles.
func (stdDev *StdDev) WithSource(source SourceFunc) *StdDev {
	stdDev.SourceFunc = source
	stdDev.variance.SourceFunc = source
	return stdDev
}

func (stdDev *StdDev) String() string {
	return fmt.Sprintf("StdDev(%d,%.2f)", stdDev.Period, stdDev.Deviation)
}

func (stdDev *StdDev) NextVal(value float64) float64 {
	variance := stdDev.variance.NextVal(value)
	return math.Sqrt(variance) * stdDev.Deviation
}

func (stdDev *StdDev) CurrentVal(value float64) float64 {
	variance := stdDev.variance.CurrentVal(value)
	return math.Sqrt(variance) * stdDev.Deviation
}

func (stdDev *StdDev) Next(candle OHLCV) []float64 {
	stdDev.out[0] = stdDev.NextVal(stdDev.SourceFunc(candle))
	return stdDev.out
}

func (stdDev *StdDev) Current(candle OHLCV) []float64 {
	stdDev.out[0] = stdDev.CurrentVal(stdDev.SourceFunc(candle))
	return stdDev.out
}

func (stdDev *StdDev) IsIdle() bool {
	return stdDev.variance.IsIdle()
}

func (stdDev *StdDev) IdlePeriod() int {
	return stdDev.variance.IdlePeriod()
}

func (stdDev *StdDev) IsWarmedUp() bool {
	return !stdDev.IsIdle()
}

func (stdDev *StdDev) WarmUpPeriod() int {
	return stdDev.IdlePeriod()
}

// Variance is a Variance indicator.
type Variance struct {
	Period          int
	SourceFunc      SourceFunc
	valueNumber     int
	buffer          *ringBuffer
	quadraticBuffer *ringBuffer
	out             []float64
}

// NewVariance creates a new Variance indicator with the given period.
func NewVariance(period int) (*Variance, error) {
	// TODO: add validations
	return &Variance{
		Period:          period,
		SourceFunc:      SourceClose,
		valueNumber:     0,
		buffer:          newRingBuffer(period),
		quadraticBuffer: newRingBuffer(period),
		out:             make([]float64, 1),
	}, nil
}

// WithSource sets the price source used to extract values from candles.
func (variance *Variance) WithSource(source SourceFunc) *Variance {
	variance.SourceFunc = source
	return variance
}

func (variance *Variance) String() string {
	return fmt.Sprintf("Variance(%d)", variance.Period)
}

func (variance *Variance) NextVal(value float64) float64 {
	variance.valueNumber++
	variance.buffer.Push(value)
	variance.quadraticBuffer.Push(value * value)
	if variance.IsIdle() {
		return 0.0
	}
	meanValue := variance.buffer.Sum / float64(variance.Period)
	meanQuadroValue := variance.quadraticBuffer.Sum / float64(variance.Period)
	return meanQuadroValue - meanValue*meanValue
}

func (variance *Variance) CurrentVal(value float64) float64 {
	variance.valueNumber++
	if variance.IsIdle() {
		variance.valueNumber--
		return 0.0
	}
	meanValue := (variance.buffer.SumExceptLast() + value) / float64(variance.Period)
	meanQuadroValue := (variance.quadraticBuffer.SumExceptLast() + value*value) / float64(variance.Period)
	result := meanQuadroValue - meanValue*meanValue
	variance.valueNumber--
	return result
}

func (variance *Variance) Next(candle OHLCV) []float64 {
	variance.out[0] = variance.NextVal(variance.SourceFunc(candle))
	return variance.out
}

func (variance *Variance) Current(candle OHLCV) []float64 {
	variance.out[0] = variance.CurrentVal(variance.SourceFunc(candle))
	return variance.out
}

func (variance *Variance) IsIdle() bool {
	return variance.valueNumber < variance.Period
}

func (variance *Variance) IdlePeriod() int {
	return variance.Period - 1
}

func (variance *Variance) IsWarmedUp() bool {
	return !variance.IsIdle()
}

func (variance *Variance) WarmUpPeriod() int {
	return variance.IdlePeriod()
}
