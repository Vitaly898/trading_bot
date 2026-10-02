//go:build analyze

// Warm-Up Period Analysis
//
// This test determines the minimum warm-up period (number of candles to feed before
// trusting indicator output) for a given indicator. Run it once per indicator to find
// the multiplier for WarmUpPeriod(), then delete or skip it.
//
// Algorithm:
//  1. For a range of indicator input parameters (e.g. period, length — depends on the
//     indicator), compute a reference by running the indicator over the FULL candle history.
//     Take the last N values as ground truth.
//  2. For a set of candidate multipliers (e.g. *4, *5, *6, *7, *8), compute the indicator
//     using only (N + IdlePeriod + param*multiplier) candles from the tail.
//     Skip the first (IdlePeriod + param*multiplier) outputs, then compare the remaining
//     N values against the reference using relative difference.
//  3. Count errors (values diverging beyond threshold) per parameter value per multiplier.
//  4. Output a table showing error counts, plus a summary of how many parameter values
//     have errors and the worst-case error count for each multiplier.
//
// How to use for a new indicator:
//  - Add a test function that calls runWarmUpAnalysis with appropriate factory/idlePeriod funcs.
//  - Run: go test -tags analyze -run Test_WarmUpAnalyze_<Name> -v
//  - Pick the multiplier that fits your tolerance.
//  - Set WarmUpPeriod() = param * chosen_multiplier in the indicator.

package talive_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/MoroZvlg/talive"
)

const (
	reliableResultsLen = 100
	diffThreshold      = 0.001 // 0.1% relative difference
)

// indicatorFactory creates a fresh indicator and returns its output count and idle period.
// The returned function feeds a candle and returns all output values.
type indicatorFactory func(param int) (nextFn func(c *testCandle) []float64, outputCount int, idlePeriod int)

func errorCount(candles []*testCandle, refTail [][]float64, factory indicatorFactory, param, warmup int) int {
	nextFn, outputCount, idlePeriod := factory(param)
	bufLen := reliableResultsLen + idlePeriod + warmup
	skip := idlePeriod + warmup

	buf := candles[len(candles)-bufLen:]
	errors := 0

	for i, c := range buf {
		vals := nextFn(c)
		if i < skip {
			continue
		}

		refIdx := i - skip
		for o := 0; o < outputCount; o++ {
			ref := refTail[o][refIdx]
			val := roundFloat(vals[o], 8)
			if ref == 0 {
				continue
			}
			if math.Abs(val-ref)/math.Abs(ref) > diffThreshold {
				errors++
				break // count at most one error per candle
			}
		}
	}
	return errors
}

func runWarmUpAnalysis(t *testing.T, name string, factory indicatorFactory, params []int, multipliers []int) {
	t.Helper()
	candles, err := readCandles("test_data/input_data.csv")
	if err != nil {
		t.Fatal(err)
	}

	// Per-param table
	fmt.Printf("param")
	for _, m := range multipliers {
		fmt.Printf("  | *%d errors", m)
	}
	fmt.Println()

	// Build reference for each param
	type paramRef struct {
		refTail    [][]float64
		idlePeriod int
	}
	refs := make(map[int]paramRef, len(params))
	for _, param := range params {
		nextFn, outputCount, idlePeriod := factory(param)
		outputs := make([][]float64, outputCount)
		for o := range outputs {
			outputs[o] = make([]float64, len(candles))
		}
		for i, c := range candles {
			vals := nextFn(c)
			for o := 0; o < outputCount; o++ {
				outputs[o][i] = roundFloat(vals[o], 8)
			}
		}
		// Take tail
		for o := range outputs {
			outputs[o] = outputs[o][len(outputs[o])-reliableResultsLen:]
		}
		refs[param] = paramRef{refTail: outputs, idlePeriod: idlePeriod}
	}

	for _, param := range params {
		ref := refs[param]
		fmt.Printf("%5d", param)
		for _, m := range multipliers {
			warmup := param * m
			maxWarmup := len(candles) - reliableResultsLen - ref.idlePeriod
			if warmup > maxWarmup {
				fmt.Printf("  | %9s", "n/a")
				continue
			}
			errors := errorCount(candles, ref.refTail, factory, param, warmup)
			fmt.Printf("  | %9d", errors)
		}
		fmt.Println()
	}

	// Summary
	fmt.Println()
	fmt.Printf("--- %s Summary ---\n", name)
	fmt.Printf("multiplier | params with errors | max errors in a param\n")
	for _, m := range multipliers {
		paramsWithErrors := 0
		maxErrors := 0
		for _, param := range params {
			ref := refs[param]
			warmup := param * m
			maxWarmup := len(candles) - reliableResultsLen - ref.idlePeriod
			if warmup > maxWarmup {
				continue
			}
			errors := errorCount(candles, ref.refTail, factory, param, warmup)
			if errors > 0 {
				paramsWithErrors++
			}
			if errors > maxErrors {
				maxErrors = errors
			}
		}
		fmt.Printf("    *%-6d | %18d | %d / %d\n", m, paramsWithErrors, maxErrors, reliableResultsLen)
	}
}

func periods2to99() []int {
	params := make([]int, 0, 98)
	for i := 2; i < 100; i++ {
		params = append(params, i)
	}
	return params
}

// ============================================================
// RSI
// ============================================================

func Test_WarmUpAnalyze_RSI(t *testing.T) {
	factory := func(period int) (func(c *testCandle) []float64, int, int) {
		ind, _ := talive.NewRSI(period)
		return func(c *testCandle) []float64 { return ind.Next(c) }, 1, ind.IdlePeriod()
	}
	runWarmUpAnalysis(t, "RSI", factory, periods2to99(), []int{4, 5, 6, 7, 8})
}

// ============================================================
// EMA
// ============================================================

func Test_WarmUpAnalyze_EMA(t *testing.T) {
	factory := func(period int) (func(c *testCandle) []float64, int, int) {
		ind, _ := talive.NewEMA(period)
		return func(c *testCandle) []float64 { return ind.Next(c) }, 1, ind.IdlePeriod()
	}
	runWarmUpAnalysis(t, "EMA", factory, periods2to99(), []int{1, 2, 3})
}

// ============================================================
// DEMA
// ============================================================

func Test_WarmUpAnalyze_DEMA(t *testing.T) {
	factory := func(period int) (func(c *testCandle) []float64, int, int) {
		ind, _ := talive.NewDEMA(period)
		return func(c *testCandle) []float64 { return ind.Next(c) }, 1, ind.IdlePeriod()
	}
	runWarmUpAnalysis(t, "DEMA", factory, periods2to99(), []int{1, 2, 3, 4})
}

// ============================================================
// TEMA
// ============================================================

func Test_WarmUpAnalyze_TEMA(t *testing.T) {
	factory := func(period int) (func(c *testCandle) []float64, int, int) {
		ind, _ := talive.NewTEMA(period)
		return func(c *testCandle) []float64 { return ind.Next(c) }, 1, ind.IdlePeriod()
	}
	runWarmUpAnalysis(t, "TEMA", factory, periods2to99(), []int{1, 2, 3, 4})
}

// ============================================================
// MACD (vary slowPeriod, fast=slowPeriod/2, signal=9)
// ============================================================

func Test_WarmUpAnalyze_MACD(t *testing.T) {
	factory := func(slowPeriod int) (func(c *testCandle) []float64, int, int) {
		fastPeriod := slowPeriod / 2
		if fastPeriod < 2 {
			fastPeriod = 2
		}
		signalPeriod := 9
		ind, _ := talive.NewMACD(fastPeriod, slowPeriod, signalPeriod)
		return func(c *testCandle) []float64 { return ind.Next(c) }, 3, ind.IdlePeriod()
	}
	// slowPeriod from 4 to 99
	params := make([]int, 0, 96)
	for i := 4; i < 100; i++ {
		params = append(params, i)
	}
	runWarmUpAnalysis(t, "MACD", factory, params, []int{5, 6, 7, 8, 9})
}

// ============================================================
// BBands with EMA (vary period, devUp=2.0, devDown=2.0)
// ============================================================

// ============================================================
// Stochastic (vary kLen, kSmooth=3, dSmooth=3)
// ============================================================

func Test_WarmUpAnalyze_Stochastic(t *testing.T) {
	factory := func(kLen int) (func(c *testCandle) []float64, int, int) {
		ind, _ := talive.NewStochastic(kLen, 3, 3)
		return func(c *testCandle) []float64 { return ind.Next(c) }, 2, ind.IdlePeriod()
	}
	runWarmUpAnalysis(t, "Stochastic", factory, periods2to99(), []int{1, 2, 3, 4, 5})
}

// ============================================================
// BBands with EMA (vary period, devUp=2.0, devDown=2.0)
// ============================================================

func Test_WarmUpAnalyze_BBands_EMA(t *testing.T) {
	factory := func(period int) (func(c *testCandle) []float64, int, int) {
		ind, _ := talive.NewBBands(period, 2.0, 2.0)
		ind.WithMA(talive.UseEMA)
		return func(c *testCandle) []float64 { return ind.Next(c) }, 3, ind.IdlePeriod()
	}
	runWarmUpAnalysis(t, "BBands(EMA)", factory, periods2to99(), []int{2, 3, 4, 5, 6})
}

// ============================================================
// CCI
// ============================================================

func Test_WarmUpAnalyze_CCI(t *testing.T) {
	factory := func(period int) (func(c *testCandle) []float64, int, int) {
		ind, _ := talive.NewCCI(period)
		return func(c *testCandle) []float64 { return ind.Next(c) }, 1, ind.IdlePeriod()
	}
	runWarmUpAnalysis(t, "CCI", factory, periods2to99(), []int{0, 1, 2, 3})
}

// ============================================================
// SMMA
// ============================================================

func Test_WarmUpAnalyze_SMMA(t *testing.T) {
	factory := func(period int) (func(c *testCandle) []float64, int, int) {
		ind, _ := talive.NewSMMA(period)
		return func(c *testCandle) []float64 { return ind.Next(c) }, 1, ind.IdlePeriod()
	}
	runWarmUpAnalysis(t, "SMMA", factory, periods2to99(), []int{5, 6, 7, 8, 9, 10})
}

// ============================================================
// ADX
// ============================================================

func Test_WarmUpAnalyze_ADX(t *testing.T) {
	factory := func(period int) (func(c *testCandle) []float64, int, int) {
		ind, _ := talive.NewADX(period)
		return func(c *testCandle) []float64 { return ind.Next(c) }, 1, ind.IdlePeriod()
	}
	runWarmUpAnalysis(t, "ADX", factory, periods2to99(), []int{8})
}

// ============================================================
// KAMA (vary slowPeriod, fix period=10, fastPeriod=2)
//
// SlowPeriod drives worst-case memory: in choppy markets ER->0 so the
// smoothing constant collapses to slowAlpha^2, giving an effective half-life
// of ~slowPeriod^2/4 candles. Multipliers on Period (the ER lookback) would
// understate this. We vary slowPeriod and report warmup as slowPeriod*M.
// ============================================================

func Test_WarmUpAnalyze_KAMA(t *testing.T) {
	factory := func(slowPeriod int) (func(c *testCandle) []float64, int, int) {
		ind, _ := talive.NewKAMA(10, 2, slowPeriod)
		return func(c *testCandle) []float64 { return ind.Next(c) }, 1, ind.IdlePeriod()
	}
	params := make([]int, 0, 51)
	for i := 10; i <= 60; i++ {
		params = append(params, i)
	}
	runWarmUpAnalysis(t, "KAMA", factory, params, []int{1, 2, 3, 4, 5, 6, 7})
}
