package talive

import "fmt"

// MFI is a Money Flow Index indicator.
type MFI struct {
	Period           int
	valueNumber      int
	positiveMfBuffer *ringBuffer
	negativeMfBuffer *ringBuffer
	prevTypicalPrice float64
	out              []float64
}

// NewMFI creates a new MFI indicator with the given period.
func NewMFI(period int) (*MFI, error) {
	if period < 2 {
		return nil, fmt.Errorf("period should be greater than 1")
	}
	return &MFI{
		Period:           period,
		valueNumber:      0,
		positiveMfBuffer: newRingBuffer(period),
		negativeMfBuffer: newRingBuffer(period),
		prevTypicalPrice: 0.0,
		out:              make([]float64, 1),
	}, nil
}

func (mfi *MFI) String() string {
	return fmt.Sprintf("MFI(%d)", mfi.Period)
}

func (mfi *MFI) Next(candle OHLCV) []float64 {
	high, low, closeV, volume := candle.High(), candle.Low(), candle.Close(), candle.Volume()
	mfi.valueNumber++
	typicalPrice := (high + low + closeV) / 3.0
	if mfi.valueNumber == 1 {
		mfi.prevTypicalPrice = typicalPrice
		mfi.out[0] = 0.0
		return mfi.out
	}

	if typicalPrice-mfi.prevTypicalPrice > 0.0 {
		mfi.positiveMfBuffer.Push(typicalPrice * volume)
		mfi.negativeMfBuffer.Push(0.0)
	} else {
		mfi.positiveMfBuffer.Push(0.0)
		mfi.negativeMfBuffer.Push(typicalPrice * volume)
	}

	if mfi.IsIdle() {
		mfi.prevTypicalPrice = typicalPrice
		mfi.out[0] = 0.0
		return mfi.out
	}

	mfiRation := mfi.positiveMfBuffer.Sum / mfi.negativeMfBuffer.Sum
	mfi.prevTypicalPrice = typicalPrice
	mfi.out[0] = 100.0 - (100.0 / (1 + mfiRation))
	return mfi.out
}

func (mfi *MFI) Current(candle OHLCV) []float64 {
	high, low, closeV, volume := candle.High(), candle.Low(), candle.Close(), candle.Volume()
	mfi.valueNumber++
	if mfi.IsIdle() {
		mfi.valueNumber--
		mfi.out[0] = 0.0
		return mfi.out
	}

	positiveMf := 0.0
	negativeMf := 0.0
	typicalPrice := (high + low + closeV) / 3.0
	if typicalPrice-mfi.prevTypicalPrice > 0.0 {
		positiveMf = typicalPrice * volume
	} else {
		negativeMf = typicalPrice * volume
	}

	mfiRation := (mfi.positiveMfBuffer.SumExceptLast() + positiveMf) / (mfi.negativeMfBuffer.SumExceptLast() + negativeMf)
	mfi.valueNumber--
	mfi.out[0] = 100.0 - (100.0 / (1 + mfiRation))
	return mfi.out
}

func (mfi *MFI) IsIdle() bool {
	return mfi.valueNumber <= mfi.Period
}

func (mfi *MFI) IdlePeriod() int {
	return mfi.Period
}

func (mfi *MFI) IsWarmedUp() bool {
	return !mfi.IsIdle()
}

func (mfi *MFI) WarmUpPeriod() int {
	return mfi.IdlePeriod()
}
