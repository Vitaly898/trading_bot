package talive

import "fmt"

// Momentum is a Momentum indicator.
type Momentum struct {
	Period      int
	SourceFunc  SourceFunc
	valueNumber int
	buffer      *ringBuffer
	out         []float64
}

// NewMomentum creates a new Momentum indicator with the given period.
func NewMomentum(period int) (*Momentum, error) {
	if period < 1 {
		return nil, fmt.Errorf("period should be greater than 0")
	}
	return &Momentum{
		Period:     period,
		SourceFunc: SourceClose,
		buffer:     newRingBuffer(period),
		out:        make([]float64, 1),
	}, nil
}

// WithSource sets the price source used to extract values from candles.
func (m *Momentum) WithSource(source SourceFunc) *Momentum {
	m.SourceFunc = source
	return m
}

func (m *Momentum) String() string {
	return fmt.Sprintf("Momentum(%d)", m.Period)
}

func (m *Momentum) Next(candle OHLCV) []float64 {
	m.valueNumber++

	value := m.SourceFunc(candle)
	oldest := m.buffer.Last()
	m.buffer.Push(value)

	if m.IsIdle() {
		return m.out
	}

	m.out[0] = value - oldest
	return m.out
}

func (m *Momentum) Current(candle OHLCV) []float64 {
	if m.IsIdle() {
		return m.out
	}

	m.out[0] = m.SourceFunc(candle) - m.buffer.Last()
	return m.out
}

func (m *Momentum) IsIdle() bool {
	return m.valueNumber <= m.Period
}

func (m *Momentum) IdlePeriod() int {
	return m.Period
}

func (m *Momentum) IsWarmedUp() bool {
	return !m.IsIdle()
}

func (m *Momentum) WarmUpPeriod() int {
	return m.IdlePeriod()
}
