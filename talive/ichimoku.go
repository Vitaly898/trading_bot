package talive

import (
	"fmt"
	"math"
)

// Ichimoku is an Ichimoku Cloud indicator.
// Outputs: [Conversion Line, Base Line, Leading Span A, Leading Span B].
// Leading Spans are displaced forward by Shift-1 bars.
// Lagging Span (close) is trivial and omitted; the **consumer** may apply the offset if it's needed.
type Ichimoku struct {
	ConvPeriod  int
	BasePeriod  int
	SpanBPeriod int
	Shift       int

	valueNumber int
	convHigh    *ringBuffer
	convLow     *ringBuffer
	baseHigh    *ringBuffer
	baseLow     *ringBuffer
	spanBHigh   *ringBuffer
	spanBLow    *ringBuffer
	leadABuf    *ringBuffer
	leadBBuf    *ringBuffer
	out         []float64
}

// NewIchimoku creates a new Ichimoku Cloud indicator.
func NewIchimoku(convPeriod, basePeriod, spanBPeriod, shift int) (*Ichimoku, error) {
	ich := &Ichimoku{
		ConvPeriod:  convPeriod,
		BasePeriod:  basePeriod,
		SpanBPeriod: spanBPeriod,
		Shift:       shift,
		convHigh:    newRingBuffer(convPeriod),
		convLow:     newRingBuffer(convPeriod),
		baseHigh:    newRingBuffer(basePeriod),
		baseLow:     newRingBuffer(basePeriod),
		spanBHigh:   newRingBuffer(spanBPeriod),
		spanBLow:    newRingBuffer(spanBPeriod),
		out:         make([]float64, 4),
	}
	if shift > 1 {
		ich.leadABuf = newRingBuffer(shift - 1)
		ich.leadBBuf = newRingBuffer(shift - 1)
	}
	return ich, nil
}

func (ich *Ichimoku) String() string {
	return fmt.Sprintf("Ichimoku(%d,%d,%d,%d)", ich.ConvPeriod, ich.BasePeriod, ich.SpanBPeriod, ich.Shift)
}

func (ich *Ichimoku) Next(candle OHLCV) []float64 {
	ich.valueNumber++
	h, l := candle.High(), candle.Low()

	ich.convHigh.Push(h)
	ich.convLow.Push(l)
	ich.baseHigh.Push(h)
	ich.baseLow.Push(l)
	ich.spanBHigh.Push(h)
	ich.spanBLow.Push(l)

	var conv float64
	if ich.valueNumber >= ich.ConvPeriod {
		conv = (ich.convHigh.Max() + ich.convLow.Min()) / 2
	}
	ich.out[0] = conv

	var base float64
	if ich.valueNumber >= ich.BasePeriod {
		base = (ich.baseHigh.Max() + ich.baseLow.Min()) / 2
	}
	ich.out[1] = base

	var leadA float64
	if ich.valueNumber >= ich.BasePeriod {
		leadA = (conv + base) / 2
	}

	var leadB float64
	if ich.valueNumber >= ich.SpanBPeriod {
		leadB = (ich.spanBHigh.Max() + ich.spanBLow.Min()) / 2
	}

	if ich.leadABuf != nil {
		// delay values by buffer length. return Last() - value from the past
		ich.out[2] = ich.leadABuf.Last()
		ich.out[3] = ich.leadBBuf.Last()
		ich.leadABuf.Push(leadA)
		ich.leadBBuf.Push(leadB)
	} else {
		ich.out[2] = leadA
		ich.out[3] = leadB
	}

	return ich.out
}

func (ich *Ichimoku) Current(candle OHLCV) []float64 {
	if ich.IsIdle() {
		return ich.out
	}

	h, l := candle.High(), candle.Low()

	conv := (math.Max(ich.convHigh.MaxExceptLast(), h) + math.Min(ich.convLow.MinExceptLast(), l)) / 2
	ich.out[0] = conv

	base := (math.Max(ich.baseHigh.MaxExceptLast(), h) + math.Min(ich.baseLow.MinExceptLast(), l)) / 2
	ich.out[1] = base

	if ich.leadABuf != nil {
		ich.out[2] = ich.leadABuf.Last()
		ich.out[3] = ich.leadBBuf.Last()
	} else {
		ich.out[2] = (conv + base) / 2
		spanB := (math.Max(ich.spanBHigh.MaxExceptLast(), h) + math.Min(ich.spanBLow.MinExceptLast(), l)) / 2
		ich.out[3] = spanB
	}

	return ich.out
}

func (ich *Ichimoku) IsIdle() bool {
	return ich.valueNumber <= ich.IdlePeriod()
}

func (ich *Ichimoku) IdlePeriod() int {
	return max(ich.ConvPeriod, ich.BasePeriod, ich.SpanBPeriod) + ich.Shift - 2
}

func (ich *Ichimoku) IsWarmedUp() bool {
	return !ich.IsIdle()
}

func (ich *Ichimoku) WarmUpPeriod() int {
	return ich.IdlePeriod()
}
