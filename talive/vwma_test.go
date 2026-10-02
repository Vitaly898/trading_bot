package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestVwmaDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/vwma/output_default.csv", []int{1}, 6)
	indicator, _ := talive.NewVWMA(20)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 6)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[VWMA(20)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestVwmaMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/vwma/output_min.csv", []int{1}, 6)
	indicator, _ := talive.NewVWMA(2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 6)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[VWMA(2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestVwmaIdle(t *testing.T) {
	indicator, _ := talive.NewVWMA(3)
	var result []string
	for i := 0; i < 4; i++ {
		indicator.Next(&testCandle{open: float64(i + 1), high: float64(i + 1), low: float64(i + 1), close: float64(i + 1), volume: 1.0})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "false", "false"}) {
		t.Fatal(`[VWMA(3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[VWMA(3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestVwmaCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/vwma/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewVWMA(20)
	for i := 0; i < 20; i++ {
		indicator.Next(candles[i])
	}
	CurrentVal := roundFloat(indicator.Current(candles[20])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][20], 8)
	if CurrentVal != expectedValue {
		t.Fatalf("[VWMA(20)] wrong Current value %f, expected %f", CurrentVal, expectedValue)
	}
	NextVal := roundFloat(indicator.Next(candles[20])[0], 8)
	if NextVal != CurrentVal {
		t.Fatalf("[VWMA(20)] Current value call broke Next value %f, expected %f", NextVal, expectedValue)
	}
}

func Benchmark_Vwma_Init_Allocations(b *testing.B) {
	b.Run("VWMA(2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewVWMA(2)
		}
	})
	b.Run("VWMA(50)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewVWMA(50)
		}
	})
}

func Benchmark_Vwma_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("VWMA(2)", func(b *testing.B) {
		indicator, _ := talive.NewVWMA(2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("VWMA(50)", func(b *testing.B) {
		indicator, _ := talive.NewVWMA(50)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Vwma_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("VWMA(2)", func(b *testing.B) {
		indicator, _ := talive.NewVWMA(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("VWMA(50)", func(b *testing.B) {
		indicator, _ := talive.NewVWMA(50)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
