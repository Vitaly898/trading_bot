package talive

import "fmt"

// TEMA is a Triple Exponential Moving Average indicator.
// TEMA = 3*EMA1 - 3*EMA2 + EMA3, where EMA2 = EMA(EMA1) and EMA3 = EMA(EMA2).
// Reduces lag further than DEMA.
type TEMA struct {
	Period      int
	SourceFunc  SourceFunc
	valueNumber int
	ema1        *EMA
	ema2        *EMA
	ema3        *EMA
	out         []float64
}

// NewTEMA creates a new TEMA indicator with the given period.
func NewTEMA(period int) (*TEMA, error) {
	if period < 2 {
		return nil, fmt.Errorf("period should be greater than 1")
	}
	ema1, _ := NewEMA(period)
	ema2, _ := NewEMA(period)
	ema3, _ := NewEMA(period)
	return &TEMA{
		Period:     period,
		SourceFunc: SourceClose,
		ema1:       ema1,
		ema2:       ema2,
		ema3:       ema3,
		out:        make([]float64, 1),
	}, nil
}

// WithSource sets the price source used to extract values from candles.
func (tema *TEMA) WithSource(source SourceFunc) *TEMA {
	tema.SourceFunc = source
	return tema
}

func (tema *TEMA) String() string {
	return fmt.Sprintf("TEMA(%d)", tema.Period)
}

func (tema *TEMA) NextVal(value float64) float64 {
	tema.valueNumber++
	e1 := tema.ema1.NextVal(value)
	if tema.ema1.IsIdle() {
		return 0.0
	}
	e2 := tema.ema2.NextVal(e1)
	if tema.ema2.IsIdle() {
		return 0.0
	}
	e3 := tema.ema3.NextVal(e2)
	if tema.ema3.IsIdle() {
		return 0.0
	}
	return 3*e1 - 3*e2 + e3
}

func (tema *TEMA) CurrentVal(value float64) float64 {
	if tema.IsIdle() {
		return 0.0
	}
	e1 := tema.ema1.CurrentVal(value)
	e2 := tema.ema2.CurrentVal(e1)
	e3 := tema.ema3.CurrentVal(e2)
	return 3*e1 - 3*e2 + e3
}

func (tema *TEMA) Next(candle OHLCV) []float64 {
	tema.out[0] = tema.NextVal(tema.SourceFunc(candle))
	return tema.out
}

func (tema *TEMA) Current(candle OHLCV) []float64 {
	tema.out[0] = tema.CurrentVal(tema.SourceFunc(candle))
	return tema.out
}

func (tema *TEMA) IsIdle() bool {
	return tema.ema3.IsIdle()
}

func (tema *TEMA) IdlePeriod() int {
	return tema.ema1.IdlePeriod() + tema.ema2.IdlePeriod() + tema.ema3.IdlePeriod()
}

func (tema *TEMA) IsWarmedUp() bool {
	return tema.valueNumber > tema.WarmUpPeriod()
}

func (tema *TEMA) WarmUpPeriod() int {
	return tema.IdlePeriod() + 2*tema.Period
}
