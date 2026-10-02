package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestBBandsDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/b_bands/output_default.csv", []int{1, 2, 3}, 4)
	indicator, _ := talive.NewBBands(20, 2.0, 2.0)
	result := [][]float64{
		make([]float64, len(candles)),
		make([]float64, len(candles)),
		make([]float64, len(candles)),
	}
	for i, candle := range candles {
		res := indicator.Next(candle)
		result[0][i] = roundFloat(res[1], 4)
		result[1][i] = roundFloat(res[0], 4)
		result[2][i] = roundFloat(res[2], 4)
	}

	if !(reflect.DeepEqual(result[0], expectedParsedData[0])) {
		t.Fatal(`[BBads(20, 2.0, 2.0, SMA)] Mid values didn't match `, difference(result[0], expectedParsedData[0]))
	}
	if !(reflect.DeepEqual(result[1], expectedParsedData[1])) {
		t.Fatal(`[BBads(20, 2.0, 2.0, SMA)] Upper values didn't match `, difference(result[1], expectedParsedData[1]))
	}
	if !(reflect.DeepEqual(result[2], expectedParsedData[2])) {
		t.Fatal(`[BBads(20, 2.0, 2.0, SMA)] Lower values didn't match `, difference(result[2], expectedParsedData[2]))
	}
}

func TestBBandsMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/b_bands/output_min.csv", []int{1, 2, 3}, 5)
	indicator, _ := talive.NewBBands(2, 0.1, 0.1)
	result := [][]float64{
		make([]float64, len(candles)),
		make([]float64, len(candles)),
		make([]float64, len(candles)),
	}
	for i, candle := range candles {
		res := indicator.Next(candle)
		result[0][i] = roundFloat(res[1], 5)
		result[1][i] = roundFloat(res[0], 5)
		result[2][i] = roundFloat(res[2], 5)
	}

	if !(reflect.DeepEqual(result[0], expectedParsedData[0])) {
		t.Fatal(`[BBads(2, 0.1, 0.1, SMA)] Mid values didn't match `, difference(result[0], expectedParsedData[0]))
	}
	if !(reflect.DeepEqual(result[1], expectedParsedData[1])) {
		t.Fatal(`[BBads(2, 0.1, 0.1, SMA)] Upper values didn't match `, difference(result[1], expectedParsedData[1]))
	}
	if !(reflect.DeepEqual(result[2], expectedParsedData[2])) {
		t.Fatal(`[BBads(2, 0.1, 0.1, SMA)] Lower values didn't match `, difference(result[2], expectedParsedData[2]))
	}
}

func TestBBandsIdle(t *testing.T) {
	indicator, _ := talive.NewBBands(5, 2.0, 2.0)
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
		t.Fatalf("[BBands(5)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestBBandsCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/b_bands/output_default.csv", []int{1, 2, 3}, 8)
	indicator, _ := talive.NewBBands(20, 2.0, 2.0)
	for i := 0; i < 22; i++ {
		indicator.Next(candles[i])
	}
	currResult := indicator.Current(candles[22])
	currUpper := currResult[0]
	currMid := currResult[1]
	currLower := currResult[2]
	expectedMid := expectedParsedData[0][22]
	expectedUpper := expectedParsedData[1][22]
	expectedLower := expectedParsedData[2][22]
	if roundFloat(currUpper, 8) != roundFloat(expectedUpper, 8) {
		t.Fatalf("[BBads(20, 2.0, 2.0, SMA)] wrong Current Upper value %f, expected %f", currUpper, expectedUpper)
	}
	if roundFloat(currMid, 8) != roundFloat(expectedMid, 8) {
		t.Fatalf("[BBads(20, 2.0, 2.0, SMA)] wrong Current Mid value %f, expected %f", currMid, expectedMid)
	}
	if roundFloat(currLower, 8) != roundFloat(expectedLower, 8) {
		t.Fatalf("[BBads(20, 2.0, 2.0, SMA)] wrong Current Lower value %f, expected %f", currLower, expectedLower)
	}
	nextResult := indicator.Next(candles[22])
	nextUpper := nextResult[0]
	nextMid := nextResult[1]
	nextLower := nextResult[2]
	if roundFloat(nextUpper, 8) != roundFloat(expectedUpper, 8) {
		t.Fatalf("[BBads(20, 2.0, 2.0, SMA)] Current Upper value call broke Next Upper value %f, expected %f", nextUpper, expectedUpper)
	}
	if roundFloat(nextMid, 8) != roundFloat(expectedMid, 8) {
		t.Fatalf("[BBads(20, 2.0, 2.0, SMA)] Current Mid value call broke Next Mid value %f, expected %f", nextMid, expectedMid)
	}
	if roundFloat(nextLower, 8) != roundFloat(expectedLower, 8) {
		t.Fatalf("[BBads(20, 2.0, 2.0, SMA)] Current Lower value call broke Next Lower value %f, expected %f", nextLower, expectedLower)
	}
}

func Benchmark_BBands_Init_Allocations(benchmark *testing.B) {
	benchmark.Run("BBands (20, 2.0, 2.0, SMA)", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewBBands(20, 2.0, 2.0)
		}
	})
	benchmark.Run("BBands (2, 0.1, 0.1, SMA)", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewBBands(2, 0.1, 0.1)
		}
	})
	benchmark.Run("BBands (30, 3.0, 3.0, EMA)", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			bb, _ := talive.NewBBands(30, 3.0, 3.0)
			bb.WithMA(talive.UseEMA)
			benchSink = bb
		}
	})
}

func Benchmark_BBands_Next_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data.csv")
	dataLen := len(candles)
	benchmark.Run("BBands (20, 2.0, 2.0, SMA)", func(benchmark *testing.B) {
		indicator, _ := talive.NewBBands(20, 2.0, 2.0)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("BBands (2, 0.1, 0.1, SMA)", func(benchmark *testing.B) {
		indicator, _ := talive.NewBBands(2, 0.1, 0.1)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	benchmark.Run("BBands (30, 3.0, 3.0, EMA)", func(benchmark *testing.B) {
		indicator, _ := talive.NewBBands(30, 3.0, 3.0)
		indicator.WithMA(talive.UseEMA)
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_BBands_Current_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data.csv")
	dataLen := len(candles)
	benchmark.Run("BBands (20, 2.0, 2.0, SMA)", func(benchmark *testing.B) {
		indicator, _ := talive.NewBBands(20, 2.0, 2.0)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("BBands (2, 0.1, 0.1, SMA)", func(benchmark *testing.B) {
		indicator, _ := talive.NewBBands(2, 0.1, 0.1)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	benchmark.Run("BBands (30, 3.0, 3.0, EMA)", func(benchmark *testing.B) {
		indicator, _ := talive.NewBBands(30, 3.0, 3.0)
		indicator.WithMA(talive.UseEMA)
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
