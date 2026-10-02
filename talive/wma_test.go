package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestWmaDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/wma/output_default.csv", []int{1}, 6)
	indicator, _ := talive.NewWMA(9)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 6)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[WMA(9)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestWmaMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/wma/output_min.csv", []int{1}, 7)
	indicator, _ := talive.NewWMA(2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[WMA(2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestWmaIdle(t *testing.T) {
	indicator, _ := talive.NewWMA(4)
	var result []string
	for i := 0; i < 6; i++ {
		indicator.Next(&testCandle{close: float64(i + 1)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "false", "false", "false"}) {
		t.Fatal(`[WMA(4)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[WMA(4)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestWmaCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/wma/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewWMA(9)
	for i := 0; i < 10; i++ {
		indicator.Next(candles[i])
	}
	CurrentVal := roundFloat(indicator.Current(candles[10])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][10], 8)
	if CurrentVal != expectedValue {
		t.Fatalf("[WMA(9)] wrong Current value %f, expected %f", CurrentVal, expectedValue)
	}
	NextVal := roundFloat(indicator.Next(candles[10])[0], 8)
	if NextVal != CurrentVal {
		t.Fatalf("[WMA(9)] Current value call broke Next value %f, expected %f", NextVal, expectedValue)
	}
}

func Benchmark_Wma_Init_Allocations(b *testing.B) {
	b.Run("WMA(2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewWMA(2)
		}
	})
	b.Run("WMA(50)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewWMA(50)
		}
	})
}

func Benchmark_Wma_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("WMA(2)", func(b *testing.B) {
		indicator, _ := talive.NewWMA(2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("WMA(50)", func(b *testing.B) {
		indicator, _ := talive.NewWMA(50)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Wma_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("WMA(2)", func(b *testing.B) {
		indicator, _ := talive.NewWMA(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("WMA(50)", func(b *testing.B) {
		indicator, _ := talive.NewWMA(50)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
