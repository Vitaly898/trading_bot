package talive

import (
	"fmt"
)

// RSI is a Relative Strength Index indicator.
type RSI struct {
	Period      int
	SourceFunc  SourceFunc
	valueNumber int
	prevPrice   float64
	gainMA      Scalar
	lossMA      Scalar
	out         []float64
}

// NewRSI creates a new RSI indicator with the given period.
func NewRSI(period int) (*RSI, error) {
	if period < 2 {
		return nil, fmt.Errorf("period should be greater than 1")
	}

	gainMA, _ := NewSMMA(period)
	lossMA, _ := NewSMMA(period)
	return &RSI{
		Period:     period,
		SourceFunc: SourceClose,
		gainMA:     gainMA,
		lossMA:     lossMA,
		out:        make([]float64, 1),
	}, nil
}

// WithMA replaces the internal smoothing method used for both gain and loss averages.
func (rsi *RSI) WithMA(ma MaType) *RSI {
	gainMA, _ := ma.New(rsi.Period)
	lossMA, _ := ma.New(rsi.Period)
	rsi.gainMA = gainMA
	rsi.lossMA = lossMA
	return rsi
}

// WithGain replaces the smoothing indicator used for gain averaging.
func (rsi *RSI) WithGain(gain Scalar) *RSI {
	rsi.gainMA = gain
	return rsi
}

// WithLoss replaces the smoothing indicator used for loss averaging.
func (rsi *RSI) WithLoss(loss Scalar) *RSI {
	rsi.lossMA = loss
	return rsi
}

// WithSource sets the price source used to extract values from candles.
func (rsi *RSI) WithSource(source SourceFunc) *RSI {
	rsi.SourceFunc = source
	return rsi
}

func (rsi *RSI) String() string {
	return fmt.Sprintf("RSI(%d)", rsi.Period)
}

func (rsi *RSI) NextVal(value float64) float64 {
	rsi.valueNumber++

	if rsi.valueNumber == 1 {
		rsi.prevPrice = value
		return 0
	}

	gain, loss := rsi.gainLoss(value)
	rsi.prevPrice = value

	avgGain := rsi.gainMA.NextVal(gain)
	avgLoss := rsi.lossMA.NextVal(loss)

	if rsi.IsIdle() {
		return 0
	}

	return 100.0 * avgGain / (avgGain + avgLoss)
}

func (rsi *RSI) CurrentVal(value float64) float64 {
	if rsi.IsIdle() {
		return 0
	}

	gain, loss := rsi.gainLoss(value)
	avgGain := rsi.gainMA.CurrentVal(gain)
	avgLoss := rsi.lossMA.CurrentVal(loss)

	return 100.0 * avgGain / (avgGain + avgLoss)
}

func (rsi *RSI) Next(candle OHLCV) []float64 {
	rsi.out[0] = rsi.NextVal(rsi.SourceFunc(candle))
	return rsi.out
}

func (rsi *RSI) Current(candle OHLCV) []float64 {
	rsi.out[0] = rsi.CurrentVal(rsi.SourceFunc(candle))
	return rsi.out
}

func (rsi *RSI) IsIdle() bool {
	return rsi.valueNumber <= rsi.Period
}

func (rsi *RSI) IsWarmedUp() bool {
	return rsi.valueNumber > rsi.WarmUpPeriod()
}

func (rsi *RSI) IdlePeriod() int {
	return rsi.Period
}

func (rsi *RSI) WarmUpPeriod() int {
	return max(rsi.gainMA.IdlePeriod(), rsi.lossMA.IdlePeriod()) + 1
}

func (rsi *RSI) gainLoss(price float64) (gain, loss float64) {
	change := price - rsi.prevPrice
	if change > 0 {
		gain = change
	} else {
		loss = -change
	}
	return gain, loss
}
