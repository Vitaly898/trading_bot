package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestStochasticRsiDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/stochastic_rsi/output_default.csv", []int{1, 2}, 7)
	indicator, _ := talive.NewStochasticRSI(14, 14, 3, 3)
	resultK := make([]float64, len(candles))
	resultD := make([]float64, len(candles))
	for i, candle := range candles {
		out := indicator.Next(candle)
		resultK[i] = roundFloat(out[0], 7)
		resultD[i] = roundFloat(out[1], 7)
	}
	if !(reflect.DeepEqual(resultK, expectedParsedData[0])) {
		t.Fatal(`[StochRSI(14,14,3,3)] K values didn't match `, difference(resultK, expectedParsedData[0]))
	}
	if !(reflect.DeepEqual(resultD, expectedParsedData[1])) {
		t.Fatal(`[StochRSI(14,14,3,3)] D values didn't match `, difference(resultD, expectedParsedData[1]))
	}
}

func TestStochasticRsiMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/stochastic_rsi/output_min.csv", []int{1, 2}, 7)
	indicator, _ := talive.NewStochasticRSI(2, 2, 1, 2)
	resultK := make([]float64, len(candles))
	resultD := make([]float64, len(candles))
	for i, candle := range candles {
		out := indicator.Next(candle)
		resultK[i] = roundFloat(out[0], 7)
		resultD[i] = roundFloat(out[1], 7)
	}
	if !(reflect.DeepEqual(resultK, expectedParsedData[0])) {
		t.Fatal(`[StochRSI(2,2,1,2)] K values didn't match `, difference(resultK, expectedParsedData[0]))
	}
	if !(reflect.DeepEqual(resultD, expectedParsedData[1])) {
		t.Fatal(`[StochRSI(2,2,1,2)] D values didn't match `, difference(resultD, expectedParsedData[1]))
	}
}

func TestStochasticRsiIdle(t *testing.T) {
	indicator, _ := talive.NewStochasticRSI(5, 4, 2, 3)
	var result []string
	for i := 0; i < 10; i++ {
		indicator.Next(&testCandle{close: float64(i + 1)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "true", "true", "true", "true", "true", "true", "false"}) {
		t.Fatal(`[StochRSI(3,3,1,2)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[StochRSI(3,3,1,2)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestStochasticRsiCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/stochastic_rsi/output_default.csv", []int{1, 2}, 8)
	indicator, _ := talive.NewStochasticRSI(14, 14, 3, 3)
	for i := 0; i < 20; i++ {
		indicator.Next(candles[i])
	}
	currentOut := indicator.Current(candles[20])
	currentK := roundFloat(currentOut[0], 8)
	currentD := roundFloat(currentOut[1], 8)
	expectedK := roundFloat(expectedParsedData[0][20], 8)
	expectedD := roundFloat(expectedParsedData[1][20], 8)
	if currentK != expectedK {
		t.Fatalf("[StochRSI(14,14,3,3)] wrong Current K value %f, expected %f", currentK, expectedK)
	}
	if currentD != expectedD {
		t.Fatalf("[StochRSI(14,14,3,3)] wrong Current D value %f, expected %f", currentD, expectedD)
	}
	nextOut := indicator.Next(candles[20])
	nextK := roundFloat(nextOut[0], 8)
	nextD := roundFloat(nextOut[1], 8)
	if nextK != currentK {
		t.Fatalf("[StochRSI(14,14,3,3)] Current call broke Next K value %f, expected %f", nextK, currentK)
	}
	if nextD != currentD {
		t.Fatalf("[StochRSI(14,14,3,3)] Current call broke Next D value %f, expected %f", nextD, currentD)
	}
}

func Benchmark_StochRsi_Init_Allocations(benchmark *testing.B) {
	benchmark.Run("StochRSI(14,14,3,3)", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewStochasticRSI(14, 14, 3, 3)
		}
	})
	benchmark.Run("StochRSI(50,50,10,10)", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewStochasticRSI(50, 50, 10, 10)
		}
	})
}

func Benchmark_StochRsi_Next_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	benchmark.Run("StochRSI(14,14,3,3)", func(benchmark *testing.B) {
		indicator, _ := talive.NewStochasticRSI(14, 14, 3, 3)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("StochRSI(50,50,10,10)", func(benchmark *testing.B) {
		indicator, _ := talive.NewStochasticRSI(50, 50, 10, 10)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_StochRsi_Current_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	benchmark.Run("StochRSI(14,14,3,3)", func(benchmark *testing.B) {
		indicator, _ := talive.NewStochasticRSI(14, 14, 3, 3)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("StochRSI(50,50,10,10)", func(benchmark *testing.B) {
		indicator, _ := talive.NewStochasticRSI(50, 50, 10, 10)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
