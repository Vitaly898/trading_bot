package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestRsiDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/rsi/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewRSI(14)
	indicator.WithMA(talive.UseSMMA)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 8)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[RSI(14)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestRsiMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/rsi/output_min.csv", []int{1}, 8)
	indicator, _ := talive.NewRSI(2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 8)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[RSI(2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestRsiIdle(t *testing.T) {
	indicator, _ := talive.NewRSI(3)
	var result []string
	for i := 0; i < 4; i++ {
		indicator.Next(&testCandle{close: float64(i)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "false"}) {
		t.Fatal(`[RSI(3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[RSI(3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestRsiCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/rsi/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewRSI(14)
	for i := 0; i < 15; i++ {
		indicator.Next(candles[i])
	}
	CurrentVal := roundFloat(indicator.Current(candles[15])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][15], 8)
	if CurrentVal != expectedValue {
		t.Fatalf("[RSI(14)] wrong Current value %f, expected %f", CurrentVal, expectedValue)
	}
	NextVal := roundFloat(indicator.Next(candles[15])[0], 8)
	if NextVal != CurrentVal {
		t.Fatalf("[RSI(14)] Current value call broke Next value %f, expected %f", NextVal, expectedValue)
	}
}

func Benchmark_Rsi_Init_Allocations(benchmark *testing.B) {
	benchmark.Run("RSI 2", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewRSI(2)
		}
	})
	benchmark.Run("RSI 14", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewRSI(14)
		}
	})
	benchmark.Run("RSI 100", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewRSI(100)
		}
	})
}

func Benchmark_Rsi_Next_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data.csv")
	dataLen := len(candles)
	benchmark.Run("RSI 2", func(benchmark *testing.B) {
		indicator, _ := talive.NewRSI(2)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			floatDummy = indicator.Next(candles[dataIndex])[0]
		}
	})
	benchmark.Run("RSI 14", func(benchmark *testing.B) {
		indicator, _ := talive.NewRSI(14)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			floatDummy = indicator.Next(candles[dataIndex])[0]
		}
	})
	benchmark.Run("RSI 100", func(benchmark *testing.B) {
		indicator, _ := talive.NewRSI(100)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			floatDummy = indicator.Next(candles[dataIndex])[0]
		}
	})
}

func Benchmark_Rsi_Current_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data.csv")
	dataLen := len(candles)
	benchmark.Run("RSI 2", func(benchmark *testing.B) {
		indicator, _ := talive.NewRSI(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			floatDummy = indicator.Current(candles[dataIndex])[0]
		}
	})
	benchmark.Run("RSI 14", func(benchmark *testing.B) {
		indicator, _ := talive.NewRSI(14)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			floatDummy = indicator.Current(candles[dataIndex])[0]
		}
	})
	benchmark.Run("RSI 100", func(benchmark *testing.B) {
		indicator, _ := talive.NewRSI(100)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			floatDummy = indicator.Current(candles[dataIndex])[0]
		}
	})
}
