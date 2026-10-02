package talive

import "fmt"

// KeltnerChannels is a Keltner Channels indicator: a moving-average basis with
// upper/lower bands offset by a multiple of ATR.
//
// Output layout: [Upper, Basis, Lower].
type KeltnerChannels struct {
	Period     int
	AtrPeriod  int
	Multiplier float64
	SourceFunc SourceFunc

	basis Scalar
	atr   *ATR
	out   []float64
}

// NewKeltnerChannels creates a new Keltner Channels indicator with the given parameters.
func NewKeltnerChannels(period, atrPeriod int, multiplier float64) (*KeltnerChannels, error) {
	if multiplier <= 0 {
		return nil, fmt.Errorf("multiplier should be positive")
	}
	basis, err := NewEMA(period)
	if err != nil {
		return nil, err
	}
	atr, err := NewATR(atrPeriod)
	if err != nil {
		return nil, err
	}
	return &KeltnerChannels{
		Period:     period,
		AtrPeriod:  atrPeriod,
		Multiplier: multiplier,
		SourceFunc: SourceClose,
		basis:      basis,
		atr:        atr,
		out:        make([]float64, 3),
	}, nil
}

// WithMA replaces the basis moving average type.
func (kc *KeltnerChannels) WithMA(ma MaType) *KeltnerChannels {
	kc.basis, _ = ma.New(kc.Period)
	return kc
}

// WithATRMA replaces the smoothing method of the inner ATR.
func (kc *KeltnerChannels) WithATRMA(ma MaType) *KeltnerChannels {
	kc.atr.WithMA(ma)
	return kc
}

// WithSource sets the price source used to extract basis values from candles.
func (kc *KeltnerChannels) WithSource(source SourceFunc) *KeltnerChannels {
	kc.SourceFunc = source
	return kc
}

func (kc *KeltnerChannels) String() string {
	return fmt.Sprintf("KeltnerChannels(%d,%d,%.2f)", kc.Period, kc.AtrPeriod, kc.Multiplier)
}

func (kc *KeltnerChannels) Next(candle OHLCV) []float64 {
	basisVal := kc.basis.NextVal(kc.SourceFunc(candle))
	atrVal := kc.atr.Next(candle)[0]

	if kc.IsIdle() {
		return kc.out
	}

	width := atrVal * kc.Multiplier
	kc.out[0] = basisVal + width
	kc.out[1] = basisVal
	kc.out[2] = basisVal - width
	return kc.out
}

func (kc *KeltnerChannels) Current(candle OHLCV) []float64 {
	if kc.IsIdle() {
		return kc.out
	}

	basisVal := kc.basis.CurrentVal(kc.SourceFunc(candle))
	atrVal := kc.atr.Current(candle)[0]

	width := atrVal * kc.Multiplier
	kc.out[0] = basisVal + width
	kc.out[1] = basisVal
	kc.out[2] = basisVal - width
	return kc.out
}

func (kc *KeltnerChannels) IsIdle() bool {
	return kc.basis.IsIdle() || kc.atr.IsIdle()
}

func (kc *KeltnerChannels) IdlePeriod() int {
	return max(kc.basis.IdlePeriod(), kc.atr.IdlePeriod())
}

func (kc *KeltnerChannels) IsWarmedUp() bool {
	return kc.basis.IsWarmedUp() && kc.atr.IsWarmedUp()
}

func (kc *KeltnerChannels) WarmUpPeriod() int {
	return max(kc.basis.WarmUpPeriod(), kc.atr.WarmUpPeriod())
}
