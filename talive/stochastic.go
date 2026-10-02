package talive

import "fmt"

// Stochastic is a Stochastic Oscillator indicator.
type Stochastic struct {
	KLen        int
	KSmooth     int
	DSmooth     int
	valueNumber int
	lowest      *ringBuffer
	highest     *ringBuffer
	kMA         Scalar
	dMA         Scalar
	out         []float64
}

// NewStochastic creates a new Stochastic Oscillator indicator with the given params.
func NewStochastic(kLen, kSmooth, dSmooth int) (*Stochastic, error) {
	kMA, _ := NewSMA(kSmooth)
	dMA, _ := NewSMA(dSmooth)
	return &Stochastic{
		KLen:        kLen,
		KSmooth:     kSmooth,
		DSmooth:     dSmooth,
		valueNumber: 0,
		lowest:      newRingBuffer(kLen),
		highest:     newRingBuffer(kLen),
		kMA:         kMA,
		dMA:         dMA,
		out:         make([]float64, 2),
	}, nil
}

func (stoch *Stochastic) String() string {
	return fmt.Sprintf("Stochastic(%d,%d,%d)", stoch.KLen, stoch.KSmooth, stoch.DSmooth)
}

func (stoch *Stochastic) Next(candle OHLCV) []float64 {
	stoch.valueNumber++

	stoch.lowest.Push(candle.Low())
	stoch.highest.Push(candle.High())

	if stoch.valueNumber < stoch.KLen {
		return stoch.out
	}

	lowestLow := stoch.lowest.Min()
	value := (candle.Close() - lowestLow) / (stoch.highest.Max() - lowestLow) * 100.0
	kSmooth := stoch.kMA.NextVal(value)
	dSmooth := stoch.dMA.NextVal(kSmooth)
	stoch.out[0] = kSmooth
	stoch.out[1] = dSmooth

	return stoch.out
}

func (stoch *Stochastic) Current(candle OHLCV) []float64 {
	if stoch.valueNumber < stoch.KLen {
		return stoch.out
	}

	lowestV := min(stoch.lowest.MinExceptLast(), candle.Low())
	highestV := max(stoch.highest.MaxExceptLast(), candle.High())

	value := (candle.Close() - lowestV) / (highestV - lowestV) * 100.0
	kSmooth := stoch.kMA.CurrentVal(value)
	dSmooth := stoch.dMA.CurrentVal(kSmooth)
	stoch.out[0] = kSmooth
	stoch.out[1] = dSmooth

	return stoch.out
}

func (stoch *Stochastic) IsIdle() bool {
	return stoch.dMA.IsIdle()
}

func (stoch *Stochastic) IsWarmedUp() bool {
	return stoch.valueNumber > stoch.WarmUpPeriod()
}

func (stoch *Stochastic) IdlePeriod() int {
	return stoch.KLen - 1 + stoch.dMA.IdlePeriod()
}

func (stoch *Stochastic) WarmUpPeriod() int {
	return stoch.IdlePeriod()
}
