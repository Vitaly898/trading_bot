package talive

import "fmt"

// CMF is a Chaikin Money Flow indicator.
type CMF struct {
	Period      int
	valueNumber int
	mfvBuffer   *ringBuffer
	volBuffer   *ringBuffer
	out         []float64
}

// NewCMF creates a new Chaikin Money Flow indicator with the given period.
func NewCMF(period int) (*CMF, error) {
	if period < 1 {
		return nil, fmt.Errorf("period should be greater than 0")
	}
	return &CMF{
		Period:    period,
		mfvBuffer: newRingBuffer(period),
		volBuffer: newRingBuffer(period),
		out:       make([]float64, 1),
	}, nil
}

func (cmf *CMF) String() string {
	return fmt.Sprintf("CMF(%d)", cmf.Period)
}

func (cmf *CMF) Next(candle OHLCV) []float64 {
	cmf.valueNumber++
	cmf.mfvBuffer.Push(cmf.moneyFlowVolume(candle))
	cmf.volBuffer.Push(candle.Volume())
	if cmf.IsIdle() {
		return cmf.out
	}

	cmf.writeOut(cmf.mfvBuffer.Sum, cmf.volBuffer.Sum)
	return cmf.out
}

func (cmf *CMF) Current(candle OHLCV) []float64 {
	if cmf.IsIdle() {
		return cmf.out
	}

	mfvSum := cmf.mfvBuffer.SumExceptLast() + cmf.moneyFlowVolume(candle)
	volSum := cmf.volBuffer.SumExceptLast() + candle.Volume()
	cmf.writeOut(mfvSum, volSum)
	return cmf.out
}

func (cmf *CMF) writeOut(mfvSum, volSum float64) {
	if volSum == 0 {
		cmf.out[0] = 0
		return
	}
	cmf.out[0] = mfvSum / volSum
}

func (cmf *CMF) moneyFlowVolume(candle OHLCV) float64 {
	high, low := candle.High(), candle.Low()
	if high == low {
		return 0
	}
	closeV := candle.Close()
	multiplier := (2*closeV - low - high) / (high - low)
	return multiplier * candle.Volume()
}

func (cmf *CMF) IsIdle() bool {
	return cmf.valueNumber < cmf.Period
}

func (cmf *CMF) IdlePeriod() int {
	return cmf.Period - 1
}

func (cmf *CMF) IsWarmedUp() bool {
	return !cmf.IsIdle()
}

func (cmf *CMF) WarmUpPeriod() int {
	return cmf.IdlePeriod()
}
