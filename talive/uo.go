package talive

import (
	"fmt"
	"math"
)

// UO is an Ultimate Oscillator indicator.
type UO struct {
	PeriodMin   int
	PeriodMid   int
	PeriodMax   int
	valueNumber int
	prevClose   float64
	bpBuf       [3]*ringBuffer
	trBuf       [3]*ringBuffer
	out         []float64
}

// NewUO creates a new Ultimate Oscillator indicator.
func NewUO(periodMin, periodMid, periodMax int) (*UO, error) {
	if periodMin < 1 || periodMid < 1 || periodMax < 1 {
		return nil, fmt.Errorf("periods should be greater than 0")
	}
	return &UO{
		PeriodMin: periodMin,
		PeriodMid: periodMid,
		PeriodMax: periodMax,
		bpBuf:     [3]*ringBuffer{newRingBuffer(periodMin), newRingBuffer(periodMid), newRingBuffer(periodMax)},
		trBuf:     [3]*ringBuffer{newRingBuffer(periodMin), newRingBuffer(periodMid), newRingBuffer(periodMax)},
		out:       make([]float64, 1),
	}, nil
}

func (uo *UO) String() string {
	return fmt.Sprintf("UO(%d,%d,%d)", uo.PeriodMid, uo.PeriodMin, uo.PeriodMax)
}

func (uo *UO) Next(candle OHLCV) []float64 {
	uo.valueNumber++

	if uo.valueNumber == 1 {
		uo.prevClose = candle.Close()
		return uo.out
	}

	bp := candle.Close() - math.Min(candle.Low(), uo.prevClose)
	tr := math.Max(candle.High(), uo.prevClose) - math.Min(candle.Low(), uo.prevClose)
	uo.prevClose = candle.Close()

	for i := 0; i < 3; i++ {
		uo.bpBuf[i].Push(bp)
		uo.trBuf[i].Push(tr)
	}

	if uo.IsIdle() {
		return uo.out
	}

	avgMin := uo.bpBuf[0].Sum / uo.trBuf[0].Sum
	avgMid := uo.bpBuf[1].Sum / uo.trBuf[1].Sum
	avgMax := uo.bpBuf[2].Sum / uo.trBuf[2].Sum
	uo.out[0] = 100 * (4*avgMin + 2*avgMid + avgMax) / 7
	return uo.out
}

func (uo *UO) Current(candle OHLCV) []float64 {
	if uo.IsIdle() {
		return uo.out
	}

	bp := candle.Close() - math.Min(candle.Low(), uo.prevClose)
	tr := math.Max(candle.High(), uo.prevClose) - math.Min(candle.Low(), uo.prevClose)

	avgMin := (uo.bpBuf[0].SumExceptLast() + bp) / (uo.trBuf[0].SumExceptLast() + tr)
	avgMid := (uo.bpBuf[1].SumExceptLast() + bp) / (uo.trBuf[1].SumExceptLast() + tr)
	avgMax := (uo.bpBuf[2].SumExceptLast() + bp) / (uo.trBuf[2].SumExceptLast() + tr)

	uo.out[0] = 100 * (4*avgMin + 2*avgMid + avgMax) / 7
	return uo.out
}

func (uo *UO) IsIdle() bool {
	maxPeriod := max(uo.PeriodMin, max(uo.PeriodMid, uo.PeriodMax))
	return uo.valueNumber <= maxPeriod
}

func (uo *UO) IdlePeriod() int {
	return max(uo.PeriodMin, max(uo.PeriodMid, uo.PeriodMax))
}

func (uo *UO) IsWarmedUp() bool {
	return !uo.IsIdle()
}

func (uo *UO) WarmUpPeriod() int {
	return uo.IdlePeriod()
}
