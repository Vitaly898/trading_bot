package talive_test

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/MoroZvlg/talive"
)

func TestADLDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/adl/output_default.csv", []int{1}, 6)
	indicator, _ := talive.NewADL()
	for i, candle := range candles {
		result := roundFloat(indicator.Next(candle)[0], 6)
		if math.Abs(result-expectedParsedData[0][i]) > 0.0000011 {
			t.Fatalf("[ADL] value at %d = %f, expected %f", i, result, expectedParsedData[0][i])
		}
	}
}

func TestADLIdle(t *testing.T) {
	indicator, _ := talive.NewADL()
	var result []string
	for i := 0; i < 3; i++ {
		indicator.Next(&testCandle{high: 10, low: 0, close: float64(i), volume: 1})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"false", "false", "false"}) {
		t.Fatal(`[ADL] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[ADL] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestADLCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/adl/output_default.csv", []int{1}, 6)
	indicator, _ := talive.NewADL()
	for i := 0; i < 11; i++ {
		indicator.Next(candles[i])
	}
	currentVal := roundFloat(indicator.Current(candles[11])[0], 6)
	expectedValue := roundFloat(expectedParsedData[0][11], 6)
	if currentVal != expectedValue {
		t.Fatalf("[ADL] wrong Current value %f, expected %f", currentVal, expectedValue)
	}
	nextVal := roundFloat(indicator.Next(candles[11])[0], 6)
	if nextVal != currentVal {
		t.Fatalf("[ADL] Current value call broke Next value %f, expected %f", nextVal, expectedValue)
	}
}

func TestADLReset(t *testing.T) {
	indicator, _ := talive.NewADL()
	indicator.Next(&testCandle{high: 10, low: 0, close: 7, volume: 100})       // ADL = +40
	out := indicator.Next(&testCandle{high: 10, low: 0, close: 2, volume: 50}) // ADL = +40 - 30 = 10
	if out[0] != 10.0 {
		t.Fatalf("[ADL] pre-Reset value = %f, want 10.0", out[0])
	}
	indicator.Reset()
	if !indicator.IsIdle() {
		t.Fatal("[ADL] should be idle after Reset")
	}
	out = indicator.Next(&testCandle{high: 8, low: 4, close: 8, volume: 25})
	if out[0] != 25.0 {
		t.Fatalf("[ADL] post-Reset value = %f, want 25.0", out[0])
	}
}

func TestADLAnchor(t *testing.T) {
	indicator, _ := talive.NewADL()
	indicator.WithAnchor(talive.AnchorDaily)
	day1a := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)
	day1b := time.Date(2024, 1, 2, 14, 0, 0, 0, time.UTC)
	day2a := time.Date(2024, 1, 3, 10, 0, 0, 0, time.UTC)
	day2b := time.Date(2024, 1, 3, 14, 0, 0, 0, time.UTC)
	day3 := time.Date(2024, 1, 4, 10, 0, 0, 0, time.UTC)

	indicator.Next(&testCandle{high: 10, low: 0, close: 7, volume: 100, timestamp: day1a}) // +40
	out := indicator.Next(&testCandle{high: 10, low: 0, close: 2, volume: 50, timestamp: day1b})
	if out[0] != 10.0 {
		t.Fatalf("[ADL] day1 accumulated = %f, want 10.0 (40 + -30)", out[0])
	}

	out = indicator.Next(&testCandle{high: 10, low: 0, close: 7, volume: 100, timestamp: day2a})
	if out[0] != 40.0 {
		t.Fatalf("[ADL] day2 first candle = %f, want 40.0 (anchor reset)", out[0])
	}
	out = indicator.Next(&testCandle{high: 10, low: 0, close: 8, volume: 25, timestamp: day2b})
	if out[0] != 55.0 {
		t.Fatalf("[ADL] day2 accumulated = %f, want 55.0 (40 + 15)", out[0])
	}

	out = indicator.Next(&testCandle{high: 8, low: 4, close: 8, volume: 25, timestamp: day3})
	if out[0] != 25.0 {
		t.Fatalf("[ADL] day3 first candle = %f, want 25.0 (anchor reset)", out[0])
	}
}

func TestADLString(t *testing.T) {
	indicator, _ := talive.NewADL()
	if got := indicator.String(); got != "ADL" {
		t.Fatalf("ADL.String() default = %q, want %q", got, "ADL")
	}
	indicator.WithAnchor(talive.AnchorWeekly)
	if got := indicator.String(); got != "ADL(anchor=Weekly)" {
		t.Fatalf("ADL.String() weekly = %q, want %q", got, "ADL(anchor=Weekly)")
	}
}

func Benchmark_ADL_Init_Allocations(benchmark *testing.B) {
	for i := 0; i < benchmark.N; i++ {
		benchSink, _ = talive.NewADL()
	}
}

func Benchmark_ADL_Next_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	indicator, _ := talive.NewADL()
	dataIndex := 0
	benchmark.ResetTimer()
	for i := 0; i < benchmark.N; i++ {
		dataIndex = limitedDataIndex(dataIndex, dataLen)
		sliceDummy = indicator.Next(candles[dataIndex])
	}
}

func Benchmark_ADL_Current_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	indicator, _ := talive.NewADL()
	dataIndex := primeForCurrentBench(indicator, candles)
	benchmark.ResetTimer()
	for i := 0; i < benchmark.N; i++ {
		dataIndex = limitedDataIndex(dataIndex, dataLen)
		sliceDummy = indicator.Current(candles[dataIndex])
	}
}
