package talive

import (
	"fmt"
	"math"
)

// Aroon is a trend indicator measuring how recently the highest high and
// lowest low occurred within the lookback window.
//
// Output layout: [AroonUp, AroonDown].
type Aroon struct {
	Period      int
	valueNumber int
	highest     *ringBuffer
	lowest      *ringBuffer
	out         []float64
}

// NewAroon creates a new Aroon indicator with the given period.
func NewAroon(period int) (*Aroon, error) {
	if period < 2 {
		return nil, fmt.Errorf("period should be greater than 1")
	}
	return &Aroon{
		Period:  period,
		highest: newRingBuffer(period + 1),
		lowest:  newRingBuffer(period + 1),
		out:     make([]float64, 2),
	}, nil
}

func (aroon *Aroon) String() string {
	return fmt.Sprintf("Aroon(%d)", aroon.Period)
}

func (aroon *Aroon) Next(candle OHLCV) []float64 {
	aroon.valueNumber++
	aroon.highest.Push(candle.High())
	aroon.lowest.Push(candle.Low())
	if aroon.IsIdle() {
		return aroon.out
	}

	aroon.writeOut(aroon.extremePositions(false, candle))
	return aroon.out
}

func (aroon *Aroon) Current(candle OHLCV) []float64 {
	if aroon.IsIdle() {
		return aroon.out
	}

	aroon.writeOut(aroon.extremePositions(true, candle))
	return aroon.out
}

func (aroon *Aroon) extremePositions(forCurrent bool, candle OHLCV) (highestIdx, lowestIdx int) {
	highestV := math.Inf(-1)
	lowestV := math.Inf(1)
	start := 0
	incomingIdx := aroon.highest.Len()

	if forCurrent && aroon.highest.Len() == aroon.highest.Cap() {
		start = 1
		incomingIdx--
	}

	for i := start; i < aroon.highest.Len(); i++ {
		idx := i - start
		high := aroon.highest.At(i)
		low := aroon.lowest.At(i)

		if high > highestV {
			highestV = high
			highestIdx = idx
		}
		if low < lowestV {
			lowestV = low
			lowestIdx = idx
		}
	}

	if forCurrent {
		high := candle.High()
		low := candle.Low()

		if high > highestV {
			highestIdx = incomingIdx
		}
		if low < lowestV {
			lowestIdx = incomingIdx
		}
	}

	return highestIdx, lowestIdx
}

func (aroon *Aroon) writeOut(highestIdx, lowestIdx int) {
	factor := 100.0 / float64(aroon.Period)
	aroon.out[0] = float64(highestIdx) * factor
	aroon.out[1] = float64(lowestIdx) * factor
}

func (aroon *Aroon) IsIdle() bool {
	return aroon.valueNumber <= aroon.Period
}

func (aroon *Aroon) IdlePeriod() int {
	return aroon.Period
}

func (aroon *Aroon) IsWarmedUp() bool {
	return !aroon.IsIdle()
}

func (aroon *Aroon) WarmUpPeriod() int {
	return aroon.IdlePeriod()
}
