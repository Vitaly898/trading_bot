package talive

import "fmt"

// Supertrend is an ATR-based trailing-stop trend indicator.
//
// Output layout: [Supertrend, Direction], where direction is +1 in uptrend
// (line below price, acting as support) and -1 in downtrend (line above price,
// acting as resistance).
type Supertrend struct {
	Period     int
	Multiplier float64
	SourceFunc SourceFunc

	atr *ATR

	prevClose      float64
	prevDirection  float64
	prevFinalUpper float64
	prevFinalLower float64

	out []float64
}

// NewSupertrend creates a new Supertrend indicator.
func NewSupertrend(period int, multiplier float64) (*Supertrend, error) {
	if multiplier <= 0 {
		return nil, fmt.Errorf("multiplier should be positive")
	}
	atr, err := NewATR(period)
	if err != nil {
		return nil, err
	}
	return &Supertrend{
		Period:     period,
		Multiplier: multiplier,
		SourceFunc: SourceHL2,
		atr:        atr,
		out:        make([]float64, 2),
	}, nil
}

// WithMA replaces the smoothing method used by the inner ATR.
func (s *Supertrend) WithMA(ma MaType) *Supertrend {
	s.atr.WithMA(ma)
	return s
}

// WithSource sets the price source used to extract values from candles.
func (s *Supertrend) WithSource(source SourceFunc) *Supertrend {
	s.SourceFunc = source
	return s
}

func (s *Supertrend) String() string {
	return fmt.Sprintf("Supertrend(%d,%.2f)", s.Period, s.Multiplier)
}

func (s *Supertrend) Next(candle OHLCV) []float64 {
	atrVal := s.atr.Next(candle)[0]

	if s.atr.IsIdle() {
		return s.out
	}

	source := s.SourceFunc(candle)
	basicUpper := source + s.Multiplier*atrVal
	basicLower := source - s.Multiplier*atrVal

	// Bootstrap on the first non-idle candle: no previous bands to trail,
	// default to downtrend per common Supertrend convention.
	if s.prevDirection == 0 {
		s.prevFinalUpper = basicUpper
		s.prevFinalLower = basicLower
		s.prevDirection = -1
		s.prevClose = candle.Close()
		s.out[0] = basicUpper
		s.out[1] = -1
		return s.out
	}

	finalUpper := s.prevFinalUpper
	if basicUpper < s.prevFinalUpper || s.prevClose > s.prevFinalUpper {
		finalUpper = basicUpper
	}
	finalLower := s.prevFinalLower
	if basicLower > s.prevFinalLower || s.prevClose < s.prevFinalLower {
		finalLower = basicLower
	}

	var supertrend, direction float64
	if s.prevDirection == -1 {
		if candle.Close() > finalUpper {
			direction = 1
			supertrend = finalLower
		} else {
			direction = -1
			supertrend = finalUpper
		}
	} else {
		if candle.Close() < finalLower {
			direction = -1
			supertrend = finalUpper
		} else {
			direction = 1
			supertrend = finalLower
		}
	}

	s.prevFinalUpper = finalUpper
	s.prevFinalLower = finalLower
	s.prevDirection = direction
	s.prevClose = candle.Close()
	s.out[0] = supertrend
	s.out[1] = direction
	return s.out
}

func (s *Supertrend) Current(candle OHLCV) []float64 {
	if s.IsIdle() {
		return s.out
	}
	atrVal := s.atr.Current(candle)[0]

	source := s.SourceFunc(candle)
	basicUpper := source + s.Multiplier*atrVal
	basicLower := source - s.Multiplier*atrVal

	if s.prevDirection == 0 {
		s.out[0] = basicUpper
		s.out[1] = -1
		return s.out
	}

	finalUpper := s.prevFinalUpper
	if basicUpper < s.prevFinalUpper || s.prevClose > s.prevFinalUpper {
		finalUpper = basicUpper
	}
	finalLower := s.prevFinalLower
	if basicLower > s.prevFinalLower || s.prevClose < s.prevFinalLower {
		finalLower = basicLower
	}

	if s.prevDirection == -1 {
		if candle.Close() > finalUpper {
			s.out[0] = finalLower
			s.out[1] = 1
		} else {
			s.out[0] = finalUpper
			s.out[1] = -1
		}
	} else {
		if candle.Close() < finalLower {
			s.out[0] = finalUpper
			s.out[1] = -1
		} else {
			s.out[0] = finalLower
			s.out[1] = 1
		}
	}
	return s.out
}

func (s *Supertrend) IsIdle() bool {
	return s.atr.IsIdle()
}

func (s *Supertrend) IdlePeriod() int {
	return s.atr.IdlePeriod()
}

func (s *Supertrend) IsWarmedUp() bool {
	return s.atr.IsWarmedUp()
}

func (s *Supertrend) WarmUpPeriod() int {
	return s.atr.WarmUpPeriod()
}
