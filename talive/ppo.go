package talive

import "fmt"

// PPO is a Percentage Price Oscillator indicator.
type PPO struct {
	FastPeriod   int
	SlowPeriod   int
	SignalPeriod int
	SourceFunc   SourceFunc
	valueNumber  int
	fastMA       Scalar
	slowMA       Scalar
	signalMA     Scalar
	out          []float64
}

// NewPPO creates a new Percentage Price Oscillator indicator with the given periods.
func NewPPO(fastPeriod, slowPeriod, signalPeriod int) (*PPO, error) {
	if fastPeriod < 2 || slowPeriod < 2 || signalPeriod < 2 {
		return nil, fmt.Errorf("fastPeriod, slowPeriod, signalPeriod should be greater than 1")
	}
	fastMA, _ := NewEMA(fastPeriod)
	slowMA, _ := NewEMA(slowPeriod)
	signalMA, _ := NewEMA(signalPeriod)

	return &PPO{
		FastPeriod:   fastPeriod,
		SlowPeriod:   slowPeriod,
		SignalPeriod: signalPeriod,
		SourceFunc:   SourceClose,
		fastMA:       fastMA,
		slowMA:       slowMA,
		signalMA:     signalMA,
		out:          make([]float64, 3),
	}, nil
}

// WithMA replaces the internal moving average type used for all PPO components.
func (ppo *PPO) WithMA(ma MaType) *PPO {
	ppo.fastMA, _ = ma.New(ppo.FastPeriod)
	ppo.slowMA, _ = ma.New(ppo.SlowPeriod)
	ppo.signalMA, _ = ma.New(ppo.SignalPeriod)
	return ppo
}

// WithSource sets the price source used to extract values from candles.
func (ppo *PPO) WithSource(source SourceFunc) *PPO {
	ppo.SourceFunc = source
	return ppo
}

func (ppo *PPO) String() string {
	return fmt.Sprintf("PPO(%d,%d,%d)", ppo.FastPeriod, ppo.SlowPeriod, ppo.SignalPeriod)
}

func (ppo *PPO) Next(candle OHLCV) []float64 {
	ppo.valueNumber++
	value := ppo.SourceFunc(candle)
	fast := ppo.fastMA.NextVal(value)
	slow := ppo.slowMA.NextVal(value)

	if ppo.slowMA.IsIdle() {
		ppo.out[0] = 0.0
		ppo.out[1] = 0.0
		ppo.out[2] = 0.0
		return ppo.out
	}

	outPPO := (fast - slow) / slow * 100.0
	outSignal := ppo.signalMA.NextVal(outPPO)
	if ppo.signalMA.IsIdle() {
		ppo.out[0] = outPPO
		ppo.out[1] = 0.0
		ppo.out[2] = 0.0
		return ppo.out
	}

	ppo.out[0] = outPPO
	ppo.out[1] = outSignal
	ppo.out[2] = outPPO - outSignal
	return ppo.out
}

func (ppo *PPO) Current(candle OHLCV) []float64 {
	value := ppo.SourceFunc(candle)
	fast := ppo.fastMA.CurrentVal(value)
	slow := ppo.slowMA.CurrentVal(value)

	if ppo.slowMA.IsIdle() {
		ppo.out[0] = 0.0
		ppo.out[1] = 0.0
		ppo.out[2] = 0.0
		return ppo.out
	}

	outPPO := (fast - slow) / slow * 100.0
	outSignal := ppo.signalMA.CurrentVal(outPPO)
	if ppo.signalMA.IsIdle() {
		ppo.out[0] = outPPO
		ppo.out[1] = 0.0
		ppo.out[2] = 0.0
		return ppo.out
	}

	ppo.out[0] = outPPO
	ppo.out[1] = outSignal
	ppo.out[2] = outPPO - outSignal
	return ppo.out
}

func (ppo *PPO) IsIdle() bool {
	return ppo.signalMA.IsIdle()
}

func (ppo *PPO) IdlePeriod() int {
	return ppo.slowMA.IdlePeriod() + ppo.signalMA.IdlePeriod()
}

func (ppo *PPO) IsWarmedUp() bool {
	return ppo.valueNumber > ppo.WarmUpPeriod()
}

func (ppo *PPO) WarmUpPeriod() int {
	return ppo.IdlePeriod() + ppo.SlowPeriod*6
}
