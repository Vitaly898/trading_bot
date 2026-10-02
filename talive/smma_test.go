package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestSmmaDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/smma/output_default.csv", []int{1}, 7)
	indicator, _ := talive.NewSMMA(7)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[SMMA(7)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestSmmaMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/smma/output_min.csv", []int{1}, 6)
	indicator, _ := talive.NewSMMA(2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 6)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[SMMA(2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestSmmaIdle(t *testing.T) {
	indicator, _ := talive.NewSMMA(3)
	var result []string
	for i := 0; i < 4; i++ {
		indicator.Next(&testCandle{close: float64(i)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "false", "false"}) {
		t.Fatal(`[SMMA(3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[SMMA(3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestSmmaCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/smma/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewSMMA(7)
	for i := 0; i < 7; i++ {
		indicator.Next(candles[i])
	}
	CurrentVal := roundFloat(indicator.Current(candles[7])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][7], 8)
	if CurrentVal != expectedValue {
		t.Fatalf("[SMMA(7)] wrong Current value %f, expected %f", CurrentVal, expectedValue)
	}
	NextVal := roundFloat(indicator.Next(candles[7])[0], 8)
	if NextVal != CurrentVal {
		t.Fatalf("[SMMA(7)] Current value call broke Next value %f, expected %f", NextVal, expectedValue)
	}
}

func Benchmark_Smma_Init_Allocations(benchmark *testing.B) {
	benchmark.Run("SMMA 2", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewSMMA(2)
		}
	})
	benchmark.Run("SMMA 50", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewSMMA(50)
		}
	})
	benchmark.Run("SMMA 100", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewSMMA(100)
		}
	})
	benchmark.Run("SMMA 1000", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewSMMA(1000)
		}
	})
}

func Benchmark_Smma_Next_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	benchmark.Run("SMMA 2", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMMA(2)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("SMMA 50", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMMA(50)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("SMMA 100", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMMA(100)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("SMMA 1000", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMMA(1000)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Smma_Current_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	benchmark.Run("SMMA 2", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMMA(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("SMMA 50", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMMA(50)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("SMMA 100", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMMA(100)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("SMMA 1000", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMMA(1000)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
