package talive

import "fmt"

// DEMA is a Double Exponential Moving Average indicator.
// DEMA = 2*EMA(price) - EMA(EMA(price)). Reduces lag versus a plain EMA.
type DEMA struct {
	Period      int
	SourceFunc  SourceFunc
	valueNumber int
	ema1        *EMA
	ema2        *EMA
	out         []float64
}

// NewDEMA creates a new DEMA indicator with the given period.
func NewDEMA(period int) (*DEMA, error) {
	if period < 2 {
		return nil, fmt.Errorf("period should be greater than 1")
	}
	ema1, _ := NewEMA(period)
	ema2, _ := NewEMA(period)
	return &DEMA{
		Period:     period,
		SourceFunc: SourceClose,
		ema1:       ema1,
		ema2:       ema2,
		out:        make([]float64, 1),
	}, nil
}

// WithSource sets the price source used to extract values from candles.
func (dema *DEMA) WithSource(source SourceFunc) *DEMA {
	dema.SourceFunc = source
	return dema
}

func (dema *DEMA) String() string {
	return fmt.Sprintf("DEMA(%d)", dema.Period)
}

func (dema *DEMA) NextVal(value float64) float64 {
	dema.valueNumber++
	e1 := dema.ema1.NextVal(value)
	if dema.ema1.IsIdle() {
		return 0.0
	}
	e2 := dema.ema2.NextVal(e1)
	if dema.ema2.IsIdle() {
		return 0.0
	}
	return 2*e1 - e2
}

func (dema *DEMA) CurrentVal(value float64) float64 {
	if dema.IsIdle() {
		return 0.0
	}
	e1 := dema.ema1.CurrentVal(value)
	e2 := dema.ema2.CurrentVal(e1)
	return 2*e1 - e2
}

func (dema *DEMA) Next(candle OHLCV) []float64 {
	dema.out[0] = dema.NextVal(dema.SourceFunc(candle))
	return dema.out
}

func (dema *DEMA) Current(candle OHLCV) []float64 {
	dema.out[0] = dema.CurrentVal(dema.SourceFunc(candle))
	return dema.out
}

func (dema *DEMA) IsIdle() bool {
	return dema.ema2.IsIdle()
}

func (dema *DEMA) IdlePeriod() int {
	return dema.ema1.IdlePeriod() + dema.ema2.IdlePeriod()
}

func (dema *DEMA) IsWarmedUp() bool {
	return dema.valueNumber > dema.WarmUpPeriod()
}

func (dema *DEMA) WarmUpPeriod() int {
	return dema.IdlePeriod() + 2*dema.Period
}
