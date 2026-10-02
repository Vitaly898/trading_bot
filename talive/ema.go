package talive

import "fmt"

// EMA is an Exponential Moving Average indicator.
type EMA struct {
	Period      int
	Alpha       float64
	SourceFunc  SourceFunc
	valueNumber int
	prevEma     float64
	out         []float64
}

// NewEMA creates a new EMA indicator with the given period.
func NewEMA(period int) (*EMA, error) {
	if period < 2 {
		return nil, fmt.Errorf("period should be greater than 1")
	}
	return &EMA{
		Period:     period,
		Alpha:      2.0 / float64(period+1),
		SourceFunc: SourceClose,
		out:        make([]float64, 1),
	}, nil
}

// WithSource sets the price source used to extract values from candles.
func (ema *EMA) WithSource(source SourceFunc) *EMA {
	ema.SourceFunc = source
	return ema
}

func (ema *EMA) String() string {
	return fmt.Sprintf("EMA(%d)", ema.Period)
}

func (ema *EMA) NextVal(value float64) float64 {
	ema.valueNumber++
	if ema.IsIdle() {
		// first EMA value = avg of close prices. We need to save them
		ema.prevEma += value
		return 0.0
	}
	if ema.valueNumber == ema.Period {
		ema.prevEma = (ema.prevEma + value) / float64(ema.Period)
		return ema.prevEma
	}

	currentEma := value*ema.Alpha + ema.prevEma*(1-ema.Alpha)
	ema.prevEma = currentEma
	return currentEma
}

func (ema *EMA) CurrentVal(value float64) float64 {
	if ema.IsIdle() {
		return 0.0
	}
	if ema.valueNumber+1 == ema.Period {
		result := (ema.prevEma + value) / float64(ema.Period)
		return result
	}
	result := value*ema.Alpha + ema.prevEma*(1-ema.Alpha)
	return result
}

func (ema *EMA) Next(candle OHLCV) []float64 {
	ema.out[0] = ema.NextVal(ema.SourceFunc(candle))
	return ema.out
}

func (ema *EMA) Current(candle OHLCV) []float64 {
	ema.out[0] = ema.CurrentVal(ema.SourceFunc(candle))
	return ema.out
}

func (ema *EMA) IsIdle() bool {
	return ema.valueNumber < ema.Period
}

func (ema *EMA) IdlePeriod() int {
	return ema.Period - 1
}

func (ema *EMA) IsWarmedUp() bool {
	return ema.valueNumber > ema.WarmUpPeriod()
}

func (ema *EMA) WarmUpPeriod() int {
	return ema.IdlePeriod() + ema.Period*2
}
