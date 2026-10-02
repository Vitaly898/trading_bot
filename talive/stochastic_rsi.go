package talive

import "fmt"

// StochasticRSI is a Stochastic RSI indicator.
type StochasticRSI struct {
	RSIPeriod  int
	StochLen   int
	KSmooth    int
	DSmooth    int
	SourceFunc SourceFunc

	valueNumber int
	rsi         *RSI
	buffer      *ringBuffer
	kMA         Scalar
	dMA         Scalar
	out         []float64
}

// NewStochasticRSI creates a new Stochastic RSI indicator.
func NewStochasticRSI(rsiPeriod, stochLen, kSmooth, dSmooth int) (*StochasticRSI, error) {
	if rsiPeriod < 2 || stochLen < 2 || kSmooth < 1 || dSmooth < 1 {
		return nil, fmt.Errorf("invalid parameters")
	}
	rsi, _ := NewRSI(rsiPeriod)
	kMA, _ := NewSMA(kSmooth)
	dMA, _ := NewSMA(dSmooth)
	return &StochasticRSI{
		RSIPeriod:  rsiPeriod,
		StochLen:   stochLen,
		KSmooth:    kSmooth,
		DSmooth:    dSmooth,
		SourceFunc: SourceClose,
		rsi:        rsi,
		buffer:     newRingBuffer(stochLen),
		kMA:        kMA,
		dMA:        dMA,
		out:        make([]float64, 2),
	}, nil
}

// WithSource sets the price source used to extract values from candles.
func (sr *StochasticRSI) WithSource(source SourceFunc) *StochasticRSI {
	sr.SourceFunc = source
	return sr
}

func (sr *StochasticRSI) String() string {
	return fmt.Sprintf("StochasticRSI(%d,%d,%d,%d)", sr.RSIPeriod, sr.StochLen, sr.KSmooth, sr.DSmooth)
}

func (sr *StochasticRSI) Next(candle OHLCV) []float64 {
	sr.valueNumber++
	rsiValue := sr.rsi.NextVal(sr.SourceFunc(candle))
	if sr.rsi.IsIdle() {
		return sr.out
	}

	sr.buffer.Push(rsiValue)

	// we need to skip iteration with 1 value in buffer (min = max).
	if sr.buffer.Len() < 2 {
		return sr.out
	}

	minV, maxV := sr.buffer.MinMax()
	raw := sr.stochValue(rsiValue, minV, maxV)

	k := sr.kMA.NextVal(raw)
	if sr.kMA.IsIdle() {
		return sr.out
	}
	sr.out[0] = k

	d := sr.dMA.NextVal(k)
	if !sr.dMA.IsIdle() {
		sr.out[1] = d
	}

	return sr.out
}

func (sr *StochasticRSI) Current(candle OHLCV) []float64 {
	if sr.IsIdle() {
		return sr.out
	}

	rsiValue := sr.rsi.CurrentVal(sr.SourceFunc(candle))

	minV, maxV := sr.buffer.MinMaxExceptLast()
	minV = min(minV, rsiValue)
	maxV = max(maxV, rsiValue)

	raw := sr.stochValue(rsiValue, minV, maxV)
	k := sr.kMA.CurrentVal(raw)
	d := sr.dMA.CurrentVal(k)
	sr.out[0] = k
	sr.out[1] = d

	return sr.out
}

func (sr *StochasticRSI) stochValue(value, minV, maxV float64) float64 {
	if maxV == minV {
		return 0
	}
	return (value - minV) / (maxV - minV) * 100
}

func (sr *StochasticRSI) IsIdle() bool {
	return sr.dMA.IsIdle()
}

func (sr *StochasticRSI) IdlePeriod() int {
	return sr.rsi.IdlePeriod() + 1 + sr.kMA.IdlePeriod() + sr.dMA.IdlePeriod()
}

func (sr *StochasticRSI) IsWarmedUp() bool {
	return sr.valueNumber > sr.WarmUpPeriod()
}

func (sr *StochasticRSI) WarmUpPeriod() int {
	// StochLen*2 because Min/Max is sensitive to small RSI errorsr. Subject to further clarification.
	return sr.rsi.WarmUpPeriod() + sr.StochLen*2 + sr.kMA.WarmUpPeriod() + sr.dMA.WarmUpPeriod()
}
