package talive_test

import (
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/MoroZvlg/talive"
)

// VWAP accumulates volume*price sums over an entire anchor period without any decay,
// so floating-point drift compounds linearly with the number of bars. Other indicators
// in this package use sliding windows or exponential smoothing so old errors fade out;
// here we compare with absolute tolerance instead of exact rounded equality.
const vwapTolerance = 0.01

func differenceWithinTolerance(got, expected []float64, tolerance float64) map[int][]float64 {
	if len(got) != len(expected) {
		return map[int][]float64{-1: {float64(len(got)), float64(len(expected))}}
	}
	result := make(map[int][]float64)
	for i := range got {
		if math.Abs(got[i]-expected[i]) > tolerance {
			result[i] = []float64{got[i], expected[i]}
		}
	}
	return result
}

func TestVWAPWeek(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/vwap/output_week.csv", []int{1, 2, 3, 4, 5, 6, 7}, 8)
	indicator, _ := talive.NewVWAP()
	indicator.WithAnchor(talive.AnchorWeekly).WithBands(1.0, 2.0, 3.0)
	result := make([][]float64, 7)
	for c := range result {
		result[c] = make([]float64, len(candles))
	}
	for i, candle := range candles {
		out := indicator.Next(candle)
		for c := 0; c < 7; c++ {
			result[c][i] = out[c]
		}
	}
	for c := 0; c < 7; c++ {
		if diff := differenceWithinTolerance(result[c], expectedParsedData[c], vwapTolerance); len(diff) > 0 {
			t.Fatalf("[VWAP weekly] column %d values didn't match %v", c, diff)
		}
	}
}

func TestVWAPMonth(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/vwap/output_month.csv", []int{1, 2, 3, 4, 5, 6, 7}, 8)
	indicator, _ := talive.NewVWAP()
	indicator.WithAnchor(talive.AnchorMonthly).WithBands(1.0, 2.0, 3.0)
	result := make([][]float64, 7)
	for c := range result {
		result[c] = make([]float64, len(candles))
	}
	for i, candle := range candles {
		out := indicator.Next(candle)
		for c := 0; c < 7; c++ {
			result[c][i] = out[c]
		}
	}
	for c := 0; c < 7; c++ {
		if diff := differenceWithinTolerance(result[c], expectedParsedData[c], vwapTolerance); len(diff) > 0 {
			t.Fatalf("[VWAP monthly] column %d values didn't match %v", c, diff)
		}
	}
}

func TestVWAPIdle(t *testing.T) {
	indicator, _ := talive.NewVWAP()
	if !indicator.IsIdle() {
		t.Fatal("[VWAP] should be idle before any Next call")
	}
	indicator.Next(&testCandle{high: 10, low: 9, close: 9.5, volume: 1.0})
	if indicator.IsIdle() {
		t.Fatal("[VWAP] should not be idle after first Next")
	}
	if indicator.IdlePeriod() != 0 {
		t.Fatalf("[VWAP] IdlePeriod = %d, want 0", indicator.IdlePeriod())
	}
}

func TestVWAPReset(t *testing.T) {
	indicator, _ := talive.NewVWAP()
	indicator.Next(&testCandle{high: 10, low: 9, close: 9.5, volume: 1.0})
	indicator.Next(&testCandle{high: 12, low: 11, close: 11.5, volume: 2.0})
	indicator.Reset()
	if !indicator.IsIdle() {
		t.Fatal("[VWAP] should be idle after Reset")
	}
	out := indicator.Next(&testCandle{high: 20, low: 18, close: 19, volume: 5.0})
	expectedVWAP := (20.0 + 18.0 + 19.0) / 3.0
	if math.Abs(out[0]-expectedVWAP) > 1e-9 {
		t.Fatalf("[VWAP] post-Reset VWAP = %f, want %f", out[0], expectedVWAP)
	}
	for i := 1; i < len(out); i++ {
		if math.Abs(out[i]-expectedVWAP) > 1e-9 {
			t.Fatalf("[VWAP] post-Reset band[%d] = %f, want %f (single candle, stddev should be 0)", i, out[i], expectedVWAP)
		}
	}
}

func TestVWAPWithBands(t *testing.T) {
	indicator, _ := talive.NewVWAP()
	indicator.WithBands() // disable bands
	out := indicator.Next(&testCandle{high: 10, low: 9, close: 9.5, volume: 1.0})
	if len(out) != 1 {
		t.Fatalf("[VWAP] WithBands() output length = %d, want 1", len(out))
	}
}

func TestVWAPString(t *testing.T) {
	tests := []struct {
		name  string
		setup func(v *talive.VWAP) *talive.VWAP
		want  string
	}{
		{"default", func(v *talive.VWAP) *talive.VWAP { return v }, "VWAP(bands=[1.00])"},
		{"anchorOnly", func(v *talive.VWAP) *talive.VWAP { return v.WithAnchor(talive.AnchorWeekly).WithBands() }, "VWAP(anchor=Weekly)"},
		{"anchorAndBands", func(v *talive.VWAP) *talive.VWAP { return v.WithAnchor(talive.AnchorDaily).WithBands(1, 2, 3) }, "VWAP(anchor=Daily,bands=[1.00,2.00,3.00])"},
		{"noBandsNoAnchor", func(v *talive.VWAP) *talive.VWAP { return v.WithBands() }, "VWAP"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, _ := talive.NewVWAP()
			tc.setup(v)
			if got := v.String(); got != tc.want {
				t.Fatalf("VWAP.String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestVWAPCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	indicator, _ := talive.NewVWAP()
	for i := 0; i < 10; i++ {
		indicator.Next(candles[i])
	}
	currentOut := slices.Clone(indicator.Current(candles[10]))
	nextOut := indicator.Next(candles[10])
	if !reflect.DeepEqual(currentOut, nextOut) {
		t.Fatalf("[VWAP] Current and Next disagree: Current=%v Next=%v", currentOut, nextOut)
	}
}

func Benchmark_VWAP_Init_Allocations(benchmark *testing.B) {
	benchmark.Run("VWAP NoBands", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			indicator, _ := talive.NewVWAP()
			benchSink = indicator.WithBands()
		}
	})
	benchmark.Run("VWAP 1Band", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewVWAP()
		}
	})
	benchmark.Run("VWAP 3Bands", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			indicator, _ := talive.NewVWAP()
			benchSink = indicator.WithBands(1.0, 2.0, 3.0)
		}
	})
}

func Benchmark_VWAP_Next_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	benchmark.Run("VWAP NoBands", func(benchmark *testing.B) {
		indicator, _ := talive.NewVWAP()
		indicator.WithBands()
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("VWAP 1Band", func(benchmark *testing.B) {
		indicator, _ := talive.NewVWAP()
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("VWAP 3Bands", func(benchmark *testing.B) {
		indicator, _ := talive.NewVWAP()
		indicator.WithBands(1.0, 2.0, 3.0)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_VWAP_Current_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	benchmark.Run("VWAP NoBands", func(benchmark *testing.B) {
		indicator, _ := talive.NewVWAP()
		indicator.WithBands()
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("VWAP 1Band", func(benchmark *testing.B) {
		indicator, _ := talive.NewVWAP()
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("VWAP 3Bands", func(benchmark *testing.B) {
		indicator, _ := talive.NewVWAP()
		indicator.WithBands(1.0, 2.0, 3.0)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
