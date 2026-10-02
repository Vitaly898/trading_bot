package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

// NOTE: My source of TA results data calculates value with Idx = Period-1.
// At that moment we have {Period-1} number of values in a buffer, but we need {Period} number of values for calcs.
// Most of the open source libraries that I saw also counts it as an Idle value.
// except this case all other results are matched.
func TestMfiDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/mfi/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewMFI(14)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 8)
	}
	expectedParsedData[0][13] = 0 // See NOTE
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[MFI(14)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestMfiMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/mfi/output_min.csv", []int{1}, 8)
	indicator, _ := talive.NewMFI(2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 8)
	}
	expectedParsedData[0][1] = 0 // See NOTE
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[MFI(2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestMfiIdle(t *testing.T) {
	indicator, _ := talive.NewMFI(3)
	var result []string
	for i := 0; i < 4; i++ {
		indicator.Next(&testCandle{
			high:   float64(i),
			low:    float64(i),
			close:  float64(i),
			volume: float64(i),
		})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "false"}) {
		t.Fatal(`[MFI(3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[MFI(3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestMfiCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/mfi/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewMFI(14)
	for i := 0; i < 15; i++ {
		indicator.Next(candles[i])
	}
	CurrentVal := roundFloat(indicator.Current(candles[15])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][15], 8)
	if CurrentVal != expectedValue {
		t.Fatalf("[MFI(14)] wrong Current value %f, expected %f", CurrentVal, expectedValue)
	}
	NextVal := roundFloat(indicator.Next(candles[15])[0], 8)
	if NextVal != CurrentVal {
		t.Fatalf("[MFI(14)] Current value call broke Next value %f, expected %f", NextVal, expectedValue)
	}
}

func Benchmark_Mfi_Init_Allocations(benchmark *testing.B) {
	benchmark.Run("MFI 14", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewMFI(14)
		}
	})
	benchmark.Run("MFI 2", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewMFI(2)
		}
	})
	benchmark.Run("MFI 1000", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewMFI(1000)
		}
	})
}

func Benchmark_Mfi_Next_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data.csv")
	dataLen := len(candles)
	benchmark.Run("MFI 2", func(benchmark *testing.B) {
		indicator, _ := talive.NewMFI(2)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("MFI 14", func(benchmark *testing.B) {
		indicator, _ := talive.NewMFI(14)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("MFI 1000", func(benchmark *testing.B) {
		indicator, _ := talive.NewMFI(1000)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Mfi_Current_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data.csv")
	dataLen := len(candles)
	benchmark.Run("MFI 2", func(benchmark *testing.B) {
		indicator, _ := talive.NewMFI(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("MFI 14", func(benchmark *testing.B) {
		indicator, _ := talive.NewMFI(14)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("MFI 1000", func(benchmark *testing.B) {
		indicator, _ := talive.NewMFI(1000)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
