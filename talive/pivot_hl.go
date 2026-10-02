package talive

import "fmt"

// PivotHL detects swing pivots using pivotHigh / pivotLow semantics.
//
// Output layout: [LastPivotHigh, LastPivotLow, Event]
//
//	LastPivotHigh = price of the most recent confirmed swing-high pivot. Step value; holds across bars;
//	LastPivotLow  = same for swing-low pivots.
//	Event         = +1 high pivot just confirmed this bar
//	                -1 low pivot just confirmed this bar
//	                 0 no confirmation this bar
//	                When both confirm on the same bar (only possible if
//	                rightHigh != rightLow), +1 wins.
type PivotHL struct {
	LeftHigh, RightHigh int
	LeftLow, RightLow   int

	highBuf *ringBuffer
	lowBuf  *ringBuffer

	lastHigh      float64
	lastLow       float64
	firstLockSeen bool

	out []float64
}

// NewPivotHL creates a PivotHL with symmetric left/right windows applied to
// both high and low detection. Use WithAsymmetric to set them independently.
func NewPivotHL(leftBars, rightBars int) (*PivotHL, error) {
	if leftBars < 1 {
		return nil, fmt.Errorf("leftBars must be positive")
	}
	if rightBars < 1 {
		return nil, fmt.Errorf("rightBars must be positive")
	}
	p := &PivotHL{out: make([]float64, 3)}
	p.WithAsymmetric(leftBars, rightBars, leftBars, rightBars)
	return p, nil
}

// WithAsymmetric configures different left/right bar counts for pivot HIGH and pivot LOW detection
func (p *PivotHL) WithAsymmetric(leftHigh, rightHigh, leftLow, rightLow int) *PivotHL {
	p.LeftHigh, p.RightHigh = leftHigh, rightHigh
	p.LeftLow, p.RightLow = leftLow, rightLow
	p.highBuf = newRingBuffer(leftHigh + rightHigh + 1)
	p.lowBuf = newRingBuffer(leftLow + rightLow + 1)
	return p
}

func (p *PivotHL) String() string {
	return fmt.Sprintf("PivotHL(lH=%d,rH=%d,lL=%d,rL=%d)",
		p.LeftHigh, p.RightHigh, p.LeftLow, p.RightLow)
}

func (p *PivotHL) Next(candle OHLCV) []float64 {
	h, l := candle.High(), candle.Low()
	event := p.tryDetectPivot(h, l, &p.lastHigh, &p.lastLow)
	if event != 0 {
		p.firstLockSeen = true
	}
	p.highBuf.Push(h)
	p.lowBuf.Push(l)
	p.writeOut(p.lastHigh, p.lastLow, event)
	return p.out
}

func (p *PivotHL) Current(candle OHLCV) []float64 {
	h, l := candle.High(), candle.Low()
	lastH, lastL := p.lastHigh, p.lastLow
	event := p.tryDetectPivot(h, l, &lastH, &lastL)
	p.writeOut(lastH, lastL, event)
	return p.out
}

func (p *PivotHL) tryDetectPivot(h, l float64, lastH, lastL *float64) int {
	event := 0
	if v, ok := pivotPeek(p.highBuf, h, p.LeftHigh, true); ok {
		*lastH = v
		event = 1
	}
	if v, ok := pivotPeek(p.lowBuf, l, p.LeftLow, false); ok {
		*lastL = v
		if event == 0 {
			event = -1
		}
	}
	return event
}

func (p *PivotHL) writeOut(highVal, lowVal float64, event int) {
	p.out[0] = highVal
	p.out[1] = lowVal
	p.out[2] = float64(event)
}

// pivotPeek tells whether pushing newVal would confirm a pivot at the bar `rightBars` positions back.
// Never mutates buffer state — used by both Next (followed by Push) and Current (no Push)
func pivotPeek(buf *ringBuffer, newVal float64, leftBars int, isHigh bool) (float64, bool) {
	windowSize := buf.Cap()
	// Pivot check needs full neighbors (+1 incoming candle)
	if buf.Len() < windowSize-1 {
		return 0, false
	}

	preShift := 0
	if buf.Len() == windowSize {
		preShift = 1
	}
	candidateVal := buf.At(leftBars + preShift)
	for pos := 0; pos < windowSize; pos++ {
		if pos == leftBars {
			continue // the candidate is compared against neighbors only
		}
		var neighborVal float64
		if pos == windowSize-1 {
			// Last post-push slot = the incoming candle (not yet buffered).
			neighborVal = newVal
		} else {
			neighborVal = buf.At(pos + preShift)
		}
		if isHigh {
			if neighborVal >= candidateVal {
				return 0, false
			}
		} else {
			if neighborVal <= candidateVal {
				return 0, false
			}
		}
	}
	return candidateVal, true
}

func (p *PivotHL) IsIdle() bool {
	return !p.firstLockSeen
}

// IdlePeriod returns 0 — first pivot lock is data-dependent (a flat market may never produce one),
// so no fixed candle count guarantees a non-zero output.
func (p *PivotHL) IdlePeriod() int {
	return 0
}

func (p *PivotHL) IsWarmedUp() bool {
	return !p.IsIdle()
}

// WarmUpPeriod returns 0 for the same reason as IdlePeriod.
func (p *PivotHL) WarmUpPeriod() int {
	return 0
}
