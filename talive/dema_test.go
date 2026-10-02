package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestDemaDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/dema/output_default.csv", []int{1}, 6)
	indicator, _ := talive.NewDEMA(9)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 6)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[DEMA(9)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestDemaMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/dema/output_min.csv", []int{1}, 7)
	indicator, _ := talive.NewDEMA(2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[DEMA(2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestDemaIdle(t *testing.T) {
	indicator, _ := talive.NewDEMA(3)
	// ema1(3) idle=2, ema2(3) idle=2 -> total idle=4
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
		t.Fatal(`[DEMA(3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[DEMA(3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestDemaCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/dema/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewDEMA(9)
	// IdlePeriod = 16; first non-zero on candle 17 (index 16). Use index 18.
	for i := 0; i < 18; i++ {
		indicator.Next(candles[i])
	}
	currentValue := roundFloat(indicator.Current(candles[18])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][18], 8)
	if currentValue != expectedValue {
		t.Fatalf("[DEMA(9)] wrong Current value %f, expected %f", currentValue, expectedValue)
	}
	nextValue := roundFloat(indicator.Next(candles[18])[0], 8)
	if nextValue != currentValue {
		t.Fatalf("[DEMA(9)] Current value call broke Next value %f, expected %f", nextValue, expectedValue)
	}
}

func Benchmark_Dema_Init_Allocations(b *testing.B) {
	b.Run("DEMA(2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewDEMA(2)
		}
	})
	b.Run("DEMA(50)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewDEMA(50)
		}
	})
}

func Benchmark_Dema_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("DEMA(2)", func(b *testing.B) {
		indicator, _ := talive.NewDEMA(2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("DEMA(50)", func(b *testing.B) {
		indicator, _ := talive.NewDEMA(50)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Dema_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("DEMA(2)", func(b *testing.B) {
		indicator, _ := talive.NewDEMA(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("DEMA(50)", func(b *testing.B) {
		indicator, _ := talive.NewDEMA(50)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
