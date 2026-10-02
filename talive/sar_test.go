package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestSarDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/sar/output_default.csv", []int{1}, 7)
	indicator, _ := talive.NewSAR(0.02, 0.02, 0.2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[SAR(0.02,0.02,0.2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestSarMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/sar/output_min.csv", []int{1}, 7)
	indicator, _ := talive.NewSAR(0.01, 0.01, 0.01)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[SAR(0.01,0.01,0.01)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestSarIdle(t *testing.T) {
	indicator, _ := talive.NewSAR(0.02, 0.02, 0.2)
	var result []string
	for i := 0; i < 3; i++ {
		indicator.Next(&testCandle{high: float64(i + 2), low: float64(i), close: float64(i + 1)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "false", "false"}) {
		t.Fatal(`[SAR] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[SAR] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestSarCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/sar/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewSAR(0.02, 0.02, 0.2)
	for i := 0; i < 5; i++ {
		indicator.Next(candles[i])
	}
	CurrentVal := roundFloat(indicator.Current(candles[5])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][5], 8)
	if CurrentVal != expectedValue {
		t.Fatalf("[SAR] wrong Current value %f, expected %f", CurrentVal, expectedValue)
	}
	NextVal := roundFloat(indicator.Next(candles[5])[0], 8)
	if NextVal != CurrentVal {
		t.Fatalf("[SAR] Current value call broke Next value %f, expected %f", NextVal, expectedValue)
	}
}

func Benchmark_Sar_Init_Allocations(benchmark *testing.B) {
	benchmark.Run("SAR(0.01,0.01,0.01)", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewSAR(0.01, 0.01, 0.01)
		}
	})
	benchmark.Run("SAR(0.02,0.02,0.2)", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewSAR(0.02, 0.02, 0.2)
		}
	})
	benchmark.Run("SAR(0.05,0.05,0.5)", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewSAR(0.05, 0.05, 0.5)
		}
	})
}

func Benchmark_Sar_Next_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	benchmark.Run("SAR(0.01,0.01,0.01)", func(benchmark *testing.B) {
		indicator, _ := talive.NewSAR(0.01, 0.01, 0.01)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("SAR(0.02,0.02,0.2)", func(benchmark *testing.B) {
		indicator, _ := talive.NewSAR(0.02, 0.02, 0.2)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("SAR(0.05,0.05,0.5)", func(benchmark *testing.B) {
		indicator, _ := talive.NewSAR(0.05, 0.05, 0.5)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Sar_Current_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	benchmark.Run("SAR(0.01,0.01,0.01)", func(benchmark *testing.B) {
		indicator, _ := talive.NewSAR(0.01, 0.01, 0.01)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("SAR(0.02,0.02,0.2)", func(benchmark *testing.B) {
		indicator, _ := talive.NewSAR(0.02, 0.02, 0.2)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("SAR(0.05,0.05,0.5)", func(benchmark *testing.B) {
		indicator, _ := talive.NewSAR(0.05, 0.05, 0.5)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
