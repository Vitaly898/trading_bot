package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestMacdDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/macd/output_default.csv", []int{1, 2, 3}, 7)
	indicator, _ := talive.NewMACD(12, 26, 9)
	result := [][]float64{
		make([]float64, len(candles)),
		make([]float64, len(candles)),
		make([]float64, len(candles)),
	}
	for i, candle := range candles {
		res := indicator.Next(candle)
		result[0][i] = roundFloat(res[2], 7)
		result[1][i] = roundFloat(res[0], 7)
		result[2][i] = roundFloat(res[1], 7)
	}

	if !(reflect.DeepEqual(result[0], expectedParsedData[0])) {
		t.Fatal(`[MACD(12, 26, 9)] Hist values didn't match `, difference(result[0], expectedParsedData[0]))
	}
	if !(reflect.DeepEqual(result[1], expectedParsedData[1])) {
		t.Fatal(`[MACD(12, 26, 9)] MACD values didn't match `, difference(result[1], expectedParsedData[1]))
	}
	if !(reflect.DeepEqual(result[2], expectedParsedData[2])) {
		t.Fatal(`[MACD(12, 26, 9)] Signal values didn't match `, difference(result[2], expectedParsedData[2]))
	}
}

func TestMacdMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/macd/output_min.csv", []int{1, 2, 3}, 7)
	indicator, _ := talive.NewMACD(2, 3, 2)
	result := [][]float64{
		make([]float64, len(candles)),
		make([]float64, len(candles)),
		make([]float64, len(candles)),
	}
	for i, candle := range candles {
		res := indicator.Next(candle)
		result[0][i] = roundFloat(res[2], 7)
		result[1][i] = roundFloat(res[0], 7)
		result[2][i] = roundFloat(res[1], 7)
	}

	if !(reflect.DeepEqual(result[0], expectedParsedData[0])) {
		t.Fatal(`[MACD(2, 3, 2)] Hist values didn't match `, difference(result[0], expectedParsedData[0]))
	}
	if !(reflect.DeepEqual(result[1], expectedParsedData[1])) {
		t.Fatal(`[MACD(2, 3, 2)] MACD values didn't match `, difference(result[1], expectedParsedData[1]))
	}
	if !(reflect.DeepEqual(result[2], expectedParsedData[2])) {
		t.Fatal(`[MACD(2, 3, 2)] Signal values didn't match `, difference(result[2], expectedParsedData[2]))
	}
}

func TestMacdIdle(t *testing.T) {
	indicator, _ := talive.NewMACD(3, 4, 2)
	var result []string
	for i := 0; i < 6; i++ {
		indicator.Next(&testCandle{close: float64(i)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "true", "false", "false"}) {
		t.Fatal(`[MACD(2, 3, 2)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[MACD(3, 4, 2)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestMacdCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/macd/output_default.csv", []int{1, 2, 3}, 7)
	indicator, _ := talive.NewMACD(12, 26, 9)
	for i := 0; i < 34; i++ {
		indicator.Next(candles[i])
	}
	currResult := indicator.Current(candles[34])
	currMacd := currResult[0]
	currSignal := currResult[1]
	currHist := currResult[2]
	expectedHist := expectedParsedData[0][34]
	expectedMacd := expectedParsedData[1][34]
	expectedSignal := expectedParsedData[2][34]
	if roundFloat(currMacd, 7) != roundFloat(expectedMacd, 7) {
		t.Fatalf("[MACD(12, 26, 9)] wrong Current Macd value %f, expected %f", currMacd, expectedMacd)
	}
	if roundFloat(currSignal, 7) != roundFloat(expectedSignal, 7) {
		t.Fatalf("[MACD(12, 26, 9)] wrong Current Signal value %f, expected %f", currSignal, expectedSignal)
	}
	if roundFloat(currHist, 7) != roundFloat(expectedHist, 7) {
		t.Fatalf("[MACD(12, 26, 9)] wrong Current Hist value %f, expected %f", currHist, expectedHist)
	}
	nextResult := indicator.Next(candles[34])
	nextMacd := nextResult[0]
	nextSignal := nextResult[1]
	nextHist := nextResult[2]

	if roundFloat(nextMacd, 7) != roundFloat(expectedMacd, 7) {
		t.Fatalf("[MACD(12, 26, 9)] Current Macd value call broke Next Macd value %f, expected %f", nextMacd, expectedMacd)
	}
	if roundFloat(nextSignal, 7) != roundFloat(expectedSignal, 7) {
		t.Fatalf("[MACD(12, 26, 9)] Current Signal value call broke Next Signal value %f, expected %f", nextSignal, expectedSignal)
	}
	if roundFloat(nextHist, 7) != roundFloat(expectedHist, 7) {
		t.Fatalf("[MACD(12, 26, 9)] Current Hist value call broke Next Hist value %f, expected %f", nextHist, expectedHist)
	}
}

func Benchmark_MACD_Init_Allocations(benchmark *testing.B) {
	benchmark.Run("MACD (12, 26, 9)", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewMACD(12, 26, 9)
		}
	})
	benchmark.Run("MACD (2, 3, 2)", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewMACD(2, 3, 2)
		}
	})
	benchmark.Run("MACD (100, 200, 15)", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewMACD(100, 200, 15)
		}
	})
}

func Benchmark_MACD_Next_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data.csv")
	dataLen := len(candles)
	benchmark.Run("MACD (12, 26, 9)", func(benchmark *testing.B) {
		indicator, _ := talive.NewMACD(12, 26, 9)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("MACD (2, 3, 2)", func(benchmark *testing.B) {
		indicator, _ := talive.NewMACD(2, 3, 2)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("MACD (100, 200, 15)", func(benchmark *testing.B) {
		indicator, _ := talive.NewMACD(100, 200, 15)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_MACD_Current_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data.csv")
	dataLen := len(candles)
	benchmark.Run("MACD (12, 26, 9)", func(benchmark *testing.B) {
		indicator, _ := talive.NewMACD(12, 26, 9)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("MACD (2, 3, 2)", func(benchmark *testing.B) {
		indicator, _ := talive.NewMACD(2, 3, 2)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("MACD (100, 200, 15)", func(benchmark *testing.B) {
		indicator, _ := talive.NewMACD(100, 200, 15)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
