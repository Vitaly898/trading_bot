// Package talive provides streaming technical analysis indicators with zero allocations.
package talive

import (
	"fmt"
	"time"
)

// OHLCV represents a single candlestick data point.
type OHLCV interface {
	Open() float64
	High() float64
	Low() float64
	Close() float64
	Volume() float64
	// Timestamp returns candle open time. Used by some Anchored indicators
	Timestamp() time.Time
}

// Indicator is the common interface for all technical indicators.
type Indicator interface {
	// Next feeds the next candle and advances the indicator state.
	// Returns zero values while IsIdle() is true.
	Next(candle OHLCV) []float64

	// Current calculates the indicator value for a candle without advancing state.
	// Returns zero values while IsIdle() is true.
	Current(candle OHLCV) []float64

	// IsIdle returns true while the indicator has not received enough candles
	// to produce meaningful output. All output values are zero during this phase.
	IsIdle() bool

	// IdlePeriod returns the number of candles that must be fed before the indicator
	// starts producing non-zero output.
	IdlePeriod() int

	// IsWarmedUp returns true when the indicator has received enough candles
	// for its output to be considered reliable. This requires more candles than
	// IdlePeriod for indicators with exponential memory (like EMA, RSI, MACD),
	// because early non-zero outputs still carry bias from limited history.
	// For indicators backed by fixed-size buffers (like SMA, MFI, StdDev),
	// IsWarmedUp is equivalent to !IsIdle. (see warmup_analysis_test.go).
	IsWarmedUp() bool

	// WarmUpPeriod returns the total number of candles that must be fed before
	// the indicator output is reliable. This value always includes IdlePeriod.
	WarmUpPeriod() int
}

// Scalar extends Indicator for single-value composable indicators.
// These operate on a raw float64 stream (pure math, no candle structure)
// and can be chained inside other indicators (e.g., RSI uses SMMA internally).
type Scalar interface {
	Indicator

	// NextVal feeds the next value and advances the indicator state.
	// Returns zero values while IsIdle() is true.
	NextVal(float64) float64

	// CurrentVal calculates the indicator value for a candle without advancing state.
	// Returns zero values while IsIdle() is true.
	CurrentVal(float64) float64
}

// Anchored extends Indicator for indicators whose accumulated state can be cleared
// and started fresh at a chosen point (e.g. VWAP, Pivot Points).
//
// IdlePeriod and WarmUpPeriod are not meaningful for Anchored indicators — when
// they exit idle depends on the anchor mode and candle frequency, not a fixed
// candle count.
type Anchored interface {
	Indicator

	// Reset triggers a manual anchor flip — the same operation auto-anchor
	// performs on a calendar boundary. Use for boundaries that can't be derived
	// from timestamps (session opens, ex-dividend dates, corporate actions).
	Reset()
}

// Anchor selects an auto-reset period for Anchored indicators. Boundaries are detected
// from candle.Timestamp() using the timestamp's own location — convert your timestamps
// to the desired timezone before feeding them in if needed. Use AnchorNone (default) to
// disable auto-detection and rely on manual Reset calls (e.g. for venue session boundaries
// or corporate-action anchors that can't be derived from time alone).
type Anchor int

// Supported anchor periods.
const (
	AnchorNone Anchor = iota
	AnchorDaily
	AnchorWeekly // ISO week (Monday-start)
	AnchorMonthly
	AnchorQuarterly
	AnchorYearly
)

func (a Anchor) String() string {
	switch a {
	case AnchorNone:
		return "None"
	case AnchorDaily:
		return "Daily"
	case AnchorWeekly:
		return "Weekly"
	case AnchorMonthly:
		return "Monthly"
	case AnchorQuarterly:
		return "Quarterly"
	case AnchorYearly:
		return "Yearly"
	}
	return "UnknownAnchor"
}

// anchorChanged reports whether prev and curr fall in different anchor periods.
// Shared internal helper for Anchored indicators that support auto-anchor modes.
func anchorChanged(prev, curr time.Time, mode Anchor) bool {
	switch mode {
	case AnchorDaily:
		return prev.Year() != curr.Year() || prev.YearDay() != curr.YearDay()
	case AnchorWeekly:
		py, pw := prev.ISOWeek()
		cy, cw := curr.ISOWeek()
		return py != cy || pw != cw
	case AnchorMonthly:
		return prev.Year() != curr.Year() || prev.Month() != curr.Month()
	case AnchorQuarterly:
		pq := (int(prev.Month()) - 1) / 3
		cq := (int(curr.Month()) - 1) / 3
		return prev.Year() != curr.Year() || pq != cq
	case AnchorYearly:
		return prev.Year() != curr.Year()
	case AnchorNone:
		return false
	default:
		return false
	}
}

// SourceFunc selects which price value an indicator reads from a candle.
// Pass one of the predefined sources or a custom function to derive the price series.
type SourceFunc func(OHLCV) float64

// SourceClose returns Candle's Close price
func SourceClose(candle OHLCV) float64 { return candle.Close() }

// SourceOpen returns Candle's Open price
func SourceOpen(candle OHLCV) float64 { return candle.Open() }

// SourceHigh returns Candle's High price
func SourceHigh(candle OHLCV) float64 { return candle.High() }

// SourceLow returns Candle's Low price
func SourceLow(candle OHLCV) float64 { return candle.Low() }

// SourceHLC3 returns the candle's typical price (high + low + close) / 3.
func SourceHLC3(candle OHLCV) float64 { return (candle.High() + candle.Low() + candle.Close()) / 3 }

// SourceHL2 returns the candle's median price (high + low) / 2.
func SourceHL2(candle OHLCV) float64 { return (candle.High() + candle.Low()) / 2 }

// MaType defines the type of moving average.
type MaType int

func (mt MaType) String() string {
	switch mt {
	case UseSMA:
		return "SMA"
	case UseEMA:
		return "EMA"
	case UseSMMA:
		return "SMMA"
	case UseWMA:
		return "WMA"
	case UseDEMA:
		return "DEMA"
	case UseTEMA:
		return "TEMA"
	}
	return "UnknownMA"
}

// New creates a new Scalar moving average of this type with the given period.
func (mt MaType) New(period int) (Scalar, error) {
	switch mt {
	case UseSMA:
		return NewSMA(period)
	case UseEMA:
		return NewEMA(period)
	case UseSMMA:
		return NewSMMA(period)
	case UseWMA:
		return NewWMA(period)
	case UseDEMA:
		return NewDEMA(period)
	case UseTEMA:
		return NewTEMA(period)
	}
	return nil, fmt.Errorf("unknown MA type: %s", mt)
}

// Supported moving average types.
const (
	UseSMA MaType = iota
	UseEMA
	UseSMMA
	UseWMA
	UseDEMA
	UseTEMA
)
