package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestUoDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/uo/output_default.csv", []int{1}, 7)
	indicator, _ := talive.NewUO(7, 14, 28)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[UO(7,14,28)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestUoMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/uo/output_min.csv", []int{1}, 7)
	indicator, _ := talive.NewUO(2, 3, 4)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[UO(2,3,4)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestUoIdle(t *testing.T) {
	indicator, _ := talive.NewUO(2, 3, 4)
	var result []string
	for i := 0; i < 6; i++ {
		indicator.Next(&testCandle{high: float64(i + 2), low: float64(i), close: float64(i + 1), volume: 1})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "true", "false", "false"}) {
		t.Fatal(`[UO(1,2,3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[UO(1,2,3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestUoCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/uo/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewUO(7, 14, 28)
	for i := 0; i < 29; i++ {
		indicator.Next(candles[i])
	}
	CurrentVal := roundFloat(indicator.Current(candles[29])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][29], 8)
	if CurrentVal != expectedValue {
		t.Fatalf("[UO(7,14,28)] wrong Current value %f, expected %f", CurrentVal, expectedValue)
	}
	NextVal := roundFloat(indicator.Next(candles[29])[0], 8)
	if NextVal != CurrentVal {
		t.Fatalf("[UO(7,14,28)] Current value call broke Next value %f, expected %f", NextVal, expectedValue)
	}
}

func Benchmark_Uo_Init_Allocations(benchmark *testing.B) {
	benchmark.Run("UO(2,3,4)", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewUO(2, 3, 4)
		}
	})
	benchmark.Run("UO(7,14,28)", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewUO(7, 14, 28)
		}
	})
	benchmark.Run("UO(14,28,56)", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewUO(14, 28, 56)
		}
	})
}

func Benchmark_Uo_Next_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	benchmark.Run("UO(2,3,4)", func(benchmark *testing.B) {
		indicator, _ := talive.NewUO(2, 3, 4)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("UO(7,14,28)", func(benchmark *testing.B) {
		indicator, _ := talive.NewUO(7, 14, 28)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("UO(14,28,56)", func(benchmark *testing.B) {
		indicator, _ := talive.NewUO(14, 28, 56)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Uo_Current_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	benchmark.Run("UO(2,3,4)", func(benchmark *testing.B) {
		indicator, _ := talive.NewUO(2, 3, 4)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("UO(7,14,28)", func(benchmark *testing.B) {
		indicator, _ := talive.NewUO(7, 14, 28)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("UO(14,28,56)", func(benchmark *testing.B) {
		indicator, _ := talive.NewUO(14, 28, 56)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
