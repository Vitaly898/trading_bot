package talive

import "fmt"

// DonchianChannels is a Donchian Channel indicator.
//
// Output layout: [Upper, Mid, Lower].
type DonchianChannels struct {
	Period      int
	valueNumber int
	highest     *ringBuffer
	lowest      *ringBuffer
	out         []float64
}

// NewDonchianChannels creates a new Donchian Channel indicator with the given period.
func NewDonchianChannels(period int) (*DonchianChannels, error) {
	if period < 1 {
		return nil, fmt.Errorf("period should be positive")
	}
	return &DonchianChannels{
		Period:  period,
		highest: newRingBuffer(period),
		lowest:  newRingBuffer(period),
		out:     make([]float64, 3),
	}, nil
}

func (dc *DonchianChannels) String() string {
	return fmt.Sprintf("DonchianChannels(%d)", dc.Period)
}

func (dc *DonchianChannels) Next(candle OHLCV) []float64 {
	dc.valueNumber++
	dc.highest.Push(candle.High())
	dc.lowest.Push(candle.Low())
	if dc.IsIdle() {
		return dc.out
	}

	upper := dc.highest.Max()
	lower := dc.lowest.Min()
	dc.out[0] = upper
	dc.out[1] = (upper + lower) / 2
	dc.out[2] = lower
	return dc.out
}

func (dc *DonchianChannels) Current(candle OHLCV) []float64 {
	if dc.IsIdle() {
		return dc.out
	}

	upper := max(dc.highest.MaxExceptLast(), candle.High())
	lower := min(dc.lowest.MinExceptLast(), candle.Low())
	dc.out[0] = upper
	dc.out[1] = (upper + lower) / 2
	dc.out[2] = lower
	return dc.out
}

func (dc *DonchianChannels) IsIdle() bool {
	return dc.valueNumber < dc.Period
}

func (dc *DonchianChannels) IdlePeriod() int {
	return dc.Period - 1
}

func (dc *DonchianChannels) IsWarmedUp() bool {
	return !dc.IsIdle()
}

func (dc *DonchianChannels) WarmUpPeriod() int {
	return dc.IdlePeriod()
}
