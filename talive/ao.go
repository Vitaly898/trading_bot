package talive

// AO is an Awesome Oscillator indicator (Bill Williams).
type AO struct {
	fastMA Scalar
	slowMA Scalar
	out    []float64
}

// NewAO creates a new Awesome Oscillator indicator.
func NewAO() (*AO, error) {
	fastMA, _ := NewSMA(5)
	slowMA, _ := NewSMA(34)
	return &AO{
		fastMA: fastMA,
		slowMA: slowMA,
		out:    make([]float64, 1),
	}, nil
}

// WithMA replaces the internal smoothing method used for fast and slow components.
func (ao *AO) WithMA(ma MaType) *AO {
	ao.fastMA, _ = ma.New(5)
	ao.slowMA, _ = ma.New(34)
	return ao
}

func (ao *AO) String() string {
	return "AO()"
}

func (ao *AO) Next(candle OHLCV) []float64 {
	hl2 := (candle.High() + candle.Low()) / 2
	fast := ao.fastMA.NextVal(hl2)
	slow := ao.slowMA.NextVal(hl2)

	if ao.IsIdle() {
		return ao.out
	}

	ao.out[0] = fast - slow
	return ao.out
}

func (ao *AO) Current(candle OHLCV) []float64 {
	if ao.IsIdle() {
		return ao.out
	}

	hl2 := (candle.High() + candle.Low()) / 2
	fast := ao.fastMA.CurrentVal(hl2)
	slow := ao.slowMA.CurrentVal(hl2)

	ao.out[0] = fast - slow
	return ao.out
}

func (ao *AO) IsIdle() bool {
	return ao.slowMA.IsIdle()
}

func (ao *AO) IdlePeriod() int {
	return ao.slowMA.IdlePeriod()
}

func (ao *AO) IsWarmedUp() bool {
	return !ao.IsIdle()
}

func (ao *AO) WarmUpPeriod() int {
	return ao.IdlePeriod()
}
