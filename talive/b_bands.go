package talive

import "fmt"

// BBands is a Bollinger Bands indicator.
type BBands struct {
	Period         int
	DevUp, DevDown float64
	SourceFunc     SourceFunc
	ma             Scalar
	basicDeviation *StdDev
	out            []float64
}

// NewBBands creates a new Bollinger Bands indicator with the given parameters.
func NewBBands(period int, devUp, devDown float64) (*BBands, error) {
	ma, err := NewSMA(period)
	if err != nil {
		return nil, err
	}
	basicDeviation, err := NewStdDev(period, 1.0)
	if err != nil {
		return nil, err
	}
	return &BBands{
		Period:         period,
		DevUp:          devUp,
		DevDown:        devDown,
		SourceFunc:     SourceClose,
		ma:             ma,
		basicDeviation: basicDeviation,
		out:            make([]float64, 3),
	}, nil
}

// WithMA replaces the internal moving average type.
func (bb *BBands) WithMA(ma MaType) *BBands {
	bb.ma, _ = ma.New(bb.Period)
	return bb
}

// WithSource sets the price source used to extract values from candles.
func (bb *BBands) WithSource(source SourceFunc) *BBands {
	bb.SourceFunc = source
	return bb
}

func (bb *BBands) String() string {
	return fmt.Sprintf("BBands(%d,%.2f,%.2f,%s)", bb.Period, bb.DevUp, bb.DevDown, bb.ma)
}

func (bb *BBands) Next(candle OHLCV) []float64 {
	value := bb.SourceFunc(candle)
	ma := bb.ma.NextVal(value)
	devBase := bb.basicDeviation.NextVal(value)

	if bb.IsIdle() {
		bb.out[0] = 0.0
		bb.out[1] = 0.0
		bb.out[2] = 0.0
		return bb.out
	}

	bb.out[0] = ma + (devBase * bb.DevUp)
	bb.out[1] = ma
	bb.out[2] = ma - (devBase * bb.DevDown)
	return bb.out
}

func (bb *BBands) Current(candle OHLCV) []float64 {
	if bb.IsIdle() {
		return bb.out
	}

	value := bb.SourceFunc(candle)
	ma := bb.ma.CurrentVal(value)
	devBase := bb.basicDeviation.CurrentVal(value)

	bb.out[0] = ma + (devBase * bb.DevUp)
	bb.out[1] = ma
	bb.out[2] = ma - (devBase * bb.DevDown)
	return bb.out
}

func (bb *BBands) IsIdle() bool {
	return bb.ma.IsIdle()
}

func (bb *BBands) IdlePeriod() int {
	return bb.ma.IdlePeriod()
}

func (bb *BBands) IsWarmedUp() bool {
	return bb.ma.IsWarmedUp()
}

func (bb *BBands) WarmUpPeriod() int {
	return bb.ma.WarmUpPeriod()
}
