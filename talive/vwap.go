package talive

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// VWAP is a Volume Weighted Average Price indicator with optional standard-deviation
// bands. Implements Anchored — use WithAnchor for time-based auto-reset, or call Reset
// manually at custom boundaries.
//
// Output layout: [vwap, upper1, lower1, upper2, lower2, ..., upperN, lowerN].
type VWAP struct {
	SourceFunc      SourceFunc
	BandMultipliers []float64
	AnchorMode      Anchor

	// Running state (Welford's online algorithm for weighted variance):
	//   sumW    = Σ volume                           (total weight)
	//   mean    = Σ(volume·price) / Σ volume          (the VWAP)
	//   sumSqW  = Σ volume·(price - mean)²            (weighted sum of squared deviations)
	// stddev  = √(sumSqW / sumW)
	//
	// Welford avoids the catastrophic cancellation of the textbook
	// E[X²] - E[X]² formula when sums grow large over thousands of candles.
	valueNumber int
	sumW        float64
	mean        float64
	sumSqW      float64
	prevTime    time.Time
	out         []float64
}

// NewVWAP creates a new VWAP indicator with HLC3 source and a single standard-deviation
// band pair at multiplier 1.0. Use WithBands to override the band multipliers.
func NewVWAP() (*VWAP, error) {
	multipliers := []float64{1.0}
	return &VWAP{
		SourceFunc:      SourceHLC3,
		BandMultipliers: multipliers,
		out:             make([]float64, 1+2*len(multipliers)),
	}, nil
}

// WithSource sets the price source used to extract values from candles.
func (v *VWAP) WithSource(source SourceFunc) *VWAP {
	v.SourceFunc = source
	return v
}

// WithBands replaces the standard-deviation band multipliers. Pass no arguments
// to disable bands entirely (output becomes [vwap]).
func (v *VWAP) WithBands(multipliers ...float64) *VWAP {
	v.BandMultipliers = multipliers
	v.out = make([]float64, 1+2*len(multipliers))
	return v
}

// WithAnchor configures automatic Reset at the start of each Anchor period.
func (v *VWAP) WithAnchor(mode Anchor) *VWAP {
	v.AnchorMode = mode
	return v
}

func (v *VWAP) String() string {
	parts := make([]string, 0, 2)
	if v.AnchorMode != AnchorNone {
		parts = append(parts, fmt.Sprintf("anchor=%s", v.AnchorMode))
	}
	if len(v.BandMultipliers) > 0 {
		bands := make([]string, len(v.BandMultipliers))
		for i, m := range v.BandMultipliers {
			bands[i] = fmt.Sprintf("%.2f", m)
		}
		parts = append(parts, fmt.Sprintf("bands=[%s]", strings.Join(bands, ",")))
	}
	if len(parts) == 0 {
		return "VWAP"
	}
	return fmt.Sprintf("VWAP(%s)", strings.Join(parts, ","))
}

func (v *VWAP) Next(candle OHLCV) []float64 {
	ts := candle.Timestamp()
	if v.valueNumber > 0 && anchorChanged(v.prevTime, ts, v.AnchorMode) {
		v.Reset()
	}
	v.prevTime = ts

	price := v.SourceFunc(candle)
	volume := candle.Volume()
	v.valueNumber++

	newSumW := v.sumW + volume
	if newSumW == 0 {
		return v.out
	}
	delta := price - v.mean
	v.sumW = newSumW
	v.mean += (volume / v.sumW) * delta
	v.sumSqW += volume * delta * (price - v.mean)

	v.computeOutput(v.mean, v.sumSqW, v.sumW)
	return v.out
}

func (v *VWAP) Current(candle OHLCV) []float64 {
	price := v.SourceFunc(candle)
	volume := candle.Volume()

	// If the candle starts a new anchor period, peek as if we'd reset first.
	resetPeek := v.valueNumber > 0 && anchorChanged(v.prevTime, candle.Timestamp(), v.AnchorMode)

	var sumW, mean, sumSqW float64
	if resetPeek {
		sumW = volume
		if sumW == 0 {
			return v.out
		}
		mean = price
		sumSqW = 0
	} else {
		sumW = v.sumW + volume
		if sumW == 0 {
			return v.out
		}
		delta := price - v.mean
		mean = v.mean + (volume/sumW)*delta
		sumSqW = v.sumSqW + volume*delta*(price-mean)
	}

	v.computeOutput(mean, sumSqW, sumW)
	return v.out
}

func (v *VWAP) computeOutput(mean, sumSqW, sumW float64) {
	v.out[0] = mean

	if len(v.BandMultipliers) == 0 {
		return
	}

	variance := max(sumSqW/sumW, 0)
	stdDev := math.Sqrt(variance)
	for i, mult := range v.BandMultipliers {
		v.out[1+2*i] = mean + mult*stdDev
		v.out[2+2*i] = mean - mult*stdDev
	}
}

func (v *VWAP) Reset() {
	v.valueNumber = 0
	v.sumW = 0
	v.mean = 0
	v.sumSqW = 0
	for i := range v.out {
		v.out[i] = 0
	}
}

func (v *VWAP) IsIdle() bool {
	return v.valueNumber == 0
}

func (v *VWAP) IdlePeriod() int {
	return 0
}

func (v *VWAP) IsWarmedUp() bool {
	return !v.IsIdle()
}

func (v *VWAP) WarmUpPeriod() int {
	return 0
}
