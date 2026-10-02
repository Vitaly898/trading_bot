package talive

import "fmt"

// ZigZag is an extreme-tracker style ZigZag indicator.
//
// Output layout: [LastHigh, LastLow, Event]
//
//	LastHigh = price of the most recent LOCKED swing-high pivot. Step value;
//	           holds across bars; updates only when a HIGH pivot locks.
//	LastLow  = same for swing-low pivots.
//	Event    = lock event at this bar:
//	             +1 = HIGH pivot just locked
//	             -1 = LOW pivot just locked
//	              0 = no lock this bar
//
// A pivot is LOCKED when the next (opposite-type) pivot is appended. Until
// then the latest pivot is provisional and may shift bars/prices as new
// extremes arrive. Confirmation lag is data-dependent, not constant.
type ZigZag struct {
	Deviation      float64
	MinTrendLength int

	state zigzagState
	out   []float64
}

const (
	zzLow  = -1
	zzHigh = +1
)

type zigzagState struct {
	valueNumber     int
	provType        int // zzLow / zzHigh; 0 before seed
	provPrice       float64
	provValueNumber int
	lockedHigh      float64
	lockedLow       float64
	firstLockSeen   bool
}

// NewZigZag creates a ZigZag indicator.
func NewZigZag(deviation float64, minTrendLength int) (*ZigZag, error) {
	if deviation < 0 {
		return nil, fmt.Errorf("deviation must be non-negative")
	}
	if minTrendLength < 1 {
		return nil, fmt.Errorf("minTrendLength must be positive")
	}
	return &ZigZag{
		Deviation:      deviation,
		MinTrendLength: minTrendLength,
		out:            make([]float64, 3),
	}, nil
}

func (zz *ZigZag) String() string {
	return fmt.Sprintf("ZigZag(dev=%g,mtl=%d)", zz.Deviation, zz.MinTrendLength)
}

func (zz *ZigZag) Next(candle OHLCV) []float64 {
	zz.step(&zz.state, candle)
	return zz.out
}

func (zz *ZigZag) Current(candle OHLCV) []float64 {
	s := zz.state
	zz.step(&s, candle)
	return zz.out
}

// step applies one candle to state and writes zz.out. Mutates the state passed in;
func (zz *ZigZag) step(s *zigzagState, candle OHLCV) {
	s.valueNumber++

	if s.valueNumber == 1 {
		s.provType = zzLow
		s.provPrice = candle.Low()
		s.provValueNumber = s.valueNumber
		return
	}

	event := 0

	if s.provType == zzLow {
		threshold := s.provPrice * (1 + zz.Deviation)
		if candle.High() >= threshold && s.valueNumber-s.provValueNumber >= zz.MinTrendLength {
			s.lockedLow = s.provPrice
			s.firstLockSeen = true
			event = -1
			s.provType = zzHigh
			s.provPrice = candle.High()
			s.provValueNumber = s.valueNumber
		} else if candle.Low() <= s.provPrice {
			s.provPrice = candle.Low()
			s.provValueNumber = s.valueNumber
		}
	} else {
		threshold := s.provPrice * (1 - zz.Deviation)
		if candle.Low() <= threshold && s.valueNumber-s.provValueNumber >= zz.MinTrendLength {
			s.lockedHigh = s.provPrice
			s.firstLockSeen = true
			event = +1
			s.provType = zzLow
			s.provPrice = candle.Low()
			s.provValueNumber = s.valueNumber
		} else if candle.High() >= s.provPrice {
			s.provPrice = candle.High()
			s.provValueNumber = s.valueNumber
		}
	}

	if s.firstLockSeen {
		zz.out[0] = s.lockedHigh
		zz.out[1] = s.lockedLow
		zz.out[2] = float64(event)
	}
}

func (zz *ZigZag) IsIdle() bool {
	return !zz.state.firstLockSeen
}

// IdlePeriod returns 0; ZigZag idle duration depends on price action, not a fixed candle count.
func (zz *ZigZag) IdlePeriod() int {
	return 0
}

func (zz *ZigZag) IsWarmedUp() bool {
	return !zz.IsIdle()
}

// WarmUpPeriod returns 0 for the same reason as IdlePeriod.
func (zz *ZigZag) WarmUpPeriod() int {
	return 0
}
