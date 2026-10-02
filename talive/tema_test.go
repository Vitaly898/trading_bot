package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestTemaDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/tema/output_default.csv", []int{1}, 5)
	indicator, _ := talive.NewTEMA(9)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 5)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[TEMA(9)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestTemaMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/tema/output_min.csv", []int{1}, 7)
	indicator, _ := talive.NewTEMA(2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[TEMA(2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestTemaIdle(t *testing.T) {
	indicator, _ := talive.NewTEMA(3)
	// ema1(3) idle=2, ema2(3) idle=2, ema3(3) idle=2 -> total idle=6
	var result []string
	for i := 0; i < 8; i++ {
		indicator.Next(&testCandle{close: float64(i + 1)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "true", "true", "true", "false", "false"}) {
		t.Fatal(`[TEMA(3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[TEMA(3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestTemaCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/tema/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewTEMA(9)
	// IdlePeriod = 24; first non-zero on candle 25 (index 24). Use index 26.
	for i := 0; i < 26; i++ {
		indicator.Next(candles[i])
	}
	currentValue := roundFloat(indicator.Current(candles[26])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][26], 8)
	if currentValue != expectedValue {
		t.Fatalf("[TEMA(9)] wrong Current value %f, expected %f", currentValue, expectedValue)
	}
	nextValue := roundFloat(indicator.Next(candles[26])[0], 8)
	if nextValue != currentValue {
		t.Fatalf("[TEMA(9)] Current value call broke Next value %f, expected %f", nextValue, expectedValue)
	}
}

func Benchmark_Tema_Init_Allocations(b *testing.B) {
	b.Run("TEMA(2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewTEMA(2)
		}
	})
	b.Run("TEMA(50)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewTEMA(50)
		}
	})
}

func Benchmark_Tema_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("TEMA(2)", func(b *testing.B) {
		indicator, _ := talive.NewTEMA(2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("TEMA(50)", func(b *testing.B) {
		indicator, _ := talive.NewTEMA(50)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Tema_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("TEMA(2)", func(b *testing.B) {
		indicator, _ := talive.NewTEMA(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("TEMA(50)", func(b *testing.B) {
		indicator, _ := talive.NewTEMA(50)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
