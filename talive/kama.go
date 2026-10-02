package talive

import (
	"fmt"
	"math"
)

// KAMA is Kaufman's Adaptive Moving Average.
type KAMA struct {
	Period      int
	FastPeriod  int
	SlowPeriod  int
	SourceFunc  SourceFunc
	fastAlpha   float64
	slowAlpha   float64
	alphaDiff   float64
	valueNumber int
	closeBuf    *ringBuffer
	absDiffBuf  *ringBuffer
	prevClose   float64
	prevKama    float64
	out         []float64
}

// NewKAMA creates a new KAMA indicator.
func NewKAMA(period, fastPeriod, slowPeriod int) (*KAMA, error) {
	if period < 2 {
		return nil, fmt.Errorf("period should be greater than 1")
	}
	if fastPeriod < 1 || slowPeriod < 1 {
		return nil, fmt.Errorf("fastPeriod and slowPeriod should be positive")
	}
	if slowPeriod <= fastPeriod {
		return nil, fmt.Errorf("slowPeriod should be greater than fastPeriod")
	}
	fastAlpha := 2.0 / (float64(fastPeriod) + 1.0)
	slowAlpha := 2.0 / (float64(slowPeriod) + 1.0)
	return &KAMA{
		Period:     period,
		FastPeriod: fastPeriod,
		SlowPeriod: slowPeriod,
		SourceFunc: SourceClose,
		fastAlpha:  fastAlpha,
		slowAlpha:  slowAlpha,
		alphaDiff:  fastAlpha - slowAlpha,
		closeBuf:   newRingBuffer(period),
		absDiffBuf: newRingBuffer(period),
		out:        make([]float64, 1),
	}, nil
}

// WithSource sets the price source used to extract values from candles.
func (k *KAMA) WithSource(source SourceFunc) *KAMA {
	k.SourceFunc = source
	return k
}

func (k *KAMA) String() string {
	return fmt.Sprintf("KAMA(%d,%d,%d)", k.Period, k.FastPeriod, k.SlowPeriod)
}

func (k *KAMA) NextVal(value float64) float64 {
	k.valueNumber++

	if k.valueNumber <= k.Period {
		if k.valueNumber > 1 {
			k.absDiffBuf.Push(math.Abs(value - k.prevClose))
		}
		k.closeBuf.Push(value)
		k.prevClose = value
		return 0.0
	}

	if k.valueNumber == k.Period+1 {
		k.absDiffBuf.Push(math.Abs(value - k.prevClose))
		k.closeBuf.Push(value)
		k.prevClose = value
		k.prevKama = value
		return value
	}

	closeNAgo := k.closeBuf.Last()
	direction := math.Abs(value - closeNAgo)

	k.absDiffBuf.Push(math.Abs(value - k.prevClose))
	volatility := k.absDiffBuf.Sum

	er := 0.0
	if volatility != 0 {
		er = direction / volatility
	}
	scPre := er*k.alphaDiff + k.slowAlpha
	sc := scPre * scPre

	k.prevKama += sc * (value - k.prevKama)
	k.closeBuf.Push(value)
	k.prevClose = value
	return k.prevKama
}

func (k *KAMA) CurrentVal(value float64) float64 {
	if k.valueNumber < k.Period {
		return 0.0
	}
	if k.valueNumber == k.Period {
		return value
	}

	closeNAgo := k.closeBuf.Last()
	direction := math.Abs(value - closeNAgo)

	absDiff := math.Abs(value - k.prevClose)
	volatility := k.absDiffBuf.Sum - k.absDiffBuf.Last() + absDiff

	er := 0.0
	if volatility != 0 {
		er = direction / volatility
	}
	scPre := er*k.alphaDiff + k.slowAlpha
	sc := scPre * scPre

	return k.prevKama + sc*(value-k.prevKama)
}

func (k *KAMA) Next(candle OHLCV) []float64 {
	k.out[0] = k.NextVal(k.SourceFunc(candle))
	return k.out
}

func (k *KAMA) Current(candle OHLCV) []float64 {
	k.out[0] = k.CurrentVal(k.SourceFunc(candle))
	return k.out
}

func (k *KAMA) IsIdle() bool {
	return k.valueNumber <= k.Period
}

func (k *KAMA) IdlePeriod() int {
	return k.Period
}

func (k *KAMA) IsWarmedUp() bool {
	return k.valueNumber > k.WarmUpPeriod()
}

func (k *KAMA) WarmUpPeriod() int {
	return k.IdlePeriod() + k.SlowPeriod*4
}
