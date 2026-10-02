package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestSmaDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/sma/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewSMA(9)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 8)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[SMA(9)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestSmaMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/sma/output_min.csv", []int{1}, 8)
	indicator, _ := talive.NewSMA(2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 8)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[SMA(2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestSmaIdle(t *testing.T) {
	indicator, _ := talive.NewSMA(3)
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
		t.Fatal(`[SMA(3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[SMA(3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestSmaCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/sma/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewSMA(9)
	for i := 0; i < 9; i++ {
		indicator.Next(candles[i])
	}
	CurrentVal := roundFloat(indicator.Current(candles[9])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][9], 8)
	if CurrentVal != expectedValue {
		t.Fatalf("[SMA(9)] wrong Current value %f, expected %f", CurrentVal, expectedValue)
	}
	NextVal := roundFloat(indicator.Next(candles[9])[0], 8)
	if NextVal != CurrentVal {
		t.Fatalf("[SMA(9)] Current value call broke Next value %f, expected %f", NextVal, expectedValue)
	}
}

func Benchmark_Sma_Init_Allocations(benchmark *testing.B) {
	benchmark.Run("SMA 2", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewSMA(2)
		}
	})
	benchmark.Run("SMA 50", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewSMA(50)
		}
	})
	benchmark.Run("SMA 100", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewSMA(100)
		}
	})
	benchmark.Run("SMA 1000", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewSMA(1000)
		}
	})
}

func Benchmark_Sma_Next_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data.csv")
	dataLen := len(candles)
	benchmark.Run("SMA 2", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMA(2)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("SMA 50", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMA(50)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("SMA 100", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMA(100)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("SMA 1000", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMA(1000)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Sma_Current_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data.csv")
	dataLen := len(candles)
	benchmark.Run("SMA 2", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMA(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("SMA 50", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMA(50)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("SMA 100", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMA(100)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("SMA 1000", func(benchmark *testing.B) {
		indicator, _ := talive.NewSMA(1000)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
