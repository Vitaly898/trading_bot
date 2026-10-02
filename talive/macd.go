package talive

import "fmt"

// MACD is a Moving Average Convergence Divergence indicator.
type MACD struct {
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

// NewMACD creates a new MACD indicator with the given periods.
func NewMACD(fastPeriod, slowPeriod, signalPeriod int) (*MACD, error) {
	if fastPeriod < 2 || slowPeriod < 2 || signalPeriod < 2 {
		return nil, fmt.Errorf("fastPeriod, slowPeriod, signalPeriod should be greater than 1")
	}
	fastMA, _ := NewEMA(fastPeriod)
	slowMA, _ := NewEMA(slowPeriod)
	signalMA, _ := NewEMA(signalPeriod)

	return &MACD{
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

// WithMA replaces the internal moving average type used for all MACD components.
func (macd *MACD) WithMA(ma MaType) *MACD {
	macd.fastMA, _ = ma.New(macd.FastPeriod)
	macd.slowMA, _ = ma.New(macd.SlowPeriod)
	macd.signalMA, _ = ma.New(macd.SignalPeriod)
	return macd
}

// WithSource sets the price source used to extract values from candles.
func (macd *MACD) WithSource(source SourceFunc) *MACD {
	macd.SourceFunc = source
	return macd
}

func (macd *MACD) String() string {
	return fmt.Sprintf("MACD(%d,%d,%d)", macd.FastPeriod, macd.SlowPeriod, macd.SignalPeriod)
}

func (macd *MACD) Next(candle OHLCV) []float64 {
	macd.valueNumber++
	value := macd.SourceFunc(candle)
	outMACD := macd.fastMA.NextVal(value) - macd.slowMA.NextVal(value)

	if macd.slowMA.IsIdle() {
		macd.out[0] = 0.0
		macd.out[1] = 0.0
		macd.out[2] = 0.0
		return macd.out
	}

	outMACDSignal := macd.signalMA.NextVal(outMACD)
	if macd.signalMA.IsIdle() {
		macd.out[0] = outMACD
		macd.out[1] = 0.0
		macd.out[2] = 0.0
		return macd.out
	}

	macd.out[0] = outMACD
	macd.out[1] = outMACDSignal
	macd.out[2] = outMACD - outMACDSignal
	return macd.out
}

func (macd *MACD) Current(candle OHLCV) []float64 {
	value := macd.SourceFunc(candle)
	outMACD := macd.fastMA.CurrentVal(value) - macd.slowMA.CurrentVal(value)

	if macd.slowMA.IsIdle() {
		macd.out[0] = 0.0
		macd.out[1] = 0.0
		macd.out[2] = 0.0
		return macd.out
	}

	outMACDSignal := macd.signalMA.CurrentVal(outMACD)
	if macd.signalMA.IsIdle() {
		macd.out[0] = outMACD
		macd.out[1] = 0.0
		macd.out[2] = 0.0
		return macd.out
	}

	macd.out[0] = outMACD
	macd.out[1] = outMACDSignal
	macd.out[2] = outMACD - outMACDSignal
	return macd.out
}

func (macd *MACD) IsIdle() bool {
	return macd.signalMA.IsIdle()
}

func (macd *MACD) IdlePeriod() int {
	return macd.slowMA.IdlePeriod() + macd.signalMA.IdlePeriod()
}

func (macd *MACD) IsWarmedUp() bool {
	return macd.valueNumber > macd.WarmUpPeriod()
}

func (macd *MACD) WarmUpPeriod() int {
	return macd.IdlePeriod() + macd.SlowPeriod*6
}
