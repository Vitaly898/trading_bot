package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestHmaDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/hma/output_default.csv", []int{1}, 5)
	indicator, _ := talive.NewHMA(9)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 5)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[HMA(9)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestHmaMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/hma/output_min.csv", []int{1}, 7)
	indicator, _ := talive.NewHMA(2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[HMA(2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestHmaIdle(t *testing.T) {
	indicator, _ := talive.NewHMA(4)
	// fullWma(4) idle=3, sqrtWma(2) idle=1 -> total idle=4
	var result []string
	for i := 0; i < 6; i++ {
		indicator.Next(&testCandle{close: float64(i + 1)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "true", "false", "false"}) {
		t.Fatal(`[HMA(4)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[HMA(4)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestHmaCurrentValue(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/hma/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewHMA(9)
	for i := 0; i < 11; i++ {
		indicator.Next(candles[i])
	}
	currentValue := roundFloat(indicator.Current(candles[11])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][11], 8)
	if currentValue != expectedValue {
		t.Fatalf("[HMA(9)] wrong Current value %f, expected %f", currentValue, expectedValue)
	}
	nextValue := roundFloat(indicator.Next(candles[11])[0], 8)
	if nextValue != currentValue {
		t.Fatalf("[HMA(9)] Current value call broke Next value %f, expected %f", nextValue, expectedValue)
	}
}

func Benchmark_Hma_Init_Allocations(b *testing.B) {
	b.Run("HMA(2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewHMA(2)
		}
	})
	b.Run("HMA(50)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewHMA(50)
		}
	})
}

func Benchmark_Hma_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("HMA(2)", func(b *testing.B) {
		indicator, _ := talive.NewHMA(2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("HMA(50)", func(b *testing.B) {
		indicator, _ := talive.NewHMA(50)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Hma_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("HMA(2)", func(b *testing.B) {
		indicator, _ := talive.NewHMA(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("HMA(50)", func(b *testing.B) {
		indicator, _ := talive.NewHMA(50)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
