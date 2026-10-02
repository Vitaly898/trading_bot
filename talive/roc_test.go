package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestROCDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/roc/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewROC(10)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 8)
	}
	if !reflect.DeepEqual(result, expectedParsedData[0]) {
		t.Fatal(`[ROC(10)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestROCMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/roc/output_min.csv", []int{1}, 8)
	indicator, _ := talive.NewROC(2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 8)
	}
	if !reflect.DeepEqual(result, expectedParsedData[0]) {
		t.Fatal(`[ROC(2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestROCIdle(t *testing.T) {
	indicator, _ := talive.NewROC(3)
	var result []string
	for i := 0; i < 5; i++ {
		indicator.Next(&testCandle{close: float64(i + 1)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "false", "false"}) {
		t.Fatal(`[ROC(3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[ROC(3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestROCCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/roc/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewROC(10)
	for i := 0; i < 11; i++ {
		indicator.Next(candles[i])
	}
	currentValue := roundFloat(indicator.Current(candles[11])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][11], 8)
	if currentValue != expectedValue {
		t.Fatalf("[ROC(10)] wrong Current value %f, expected %f", currentValue, expectedValue)
	}
	nextValue := roundFloat(indicator.Next(candles[11])[0], 8)
	if nextValue != currentValue {
		t.Fatalf("[ROC(10)] Current value call broke Next value %f, expected %f", nextValue, expectedValue)
	}
}

func TestROCInvalidPeriod(t *testing.T) {
	if _, err := talive.NewROC(0); err == nil {
		t.Fatal("NewROC(0) should return an error")
	}
}

func Benchmark_ROC_Init_Allocations(b *testing.B) {
	b.Run("ROC(10)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewROC(10)
		}
	})
	b.Run("ROC(2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewROC(2)
		}
	})
	b.Run("ROC(50)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewROC(50)
		}
	})
}

func Benchmark_ROC_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("ROC(10)", func(b *testing.B) {
		indicator, _ := talive.NewROC(10)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("ROC(2)", func(b *testing.B) {
		indicator, _ := talive.NewROC(2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("ROC(50)", func(b *testing.B) {
		indicator, _ := talive.NewROC(50)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_ROC_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("ROC(10)", func(b *testing.B) {
		indicator, _ := talive.NewROC(10)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("ROC(2)", func(b *testing.B) {
		indicator, _ := talive.NewROC(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("ROC(50)", func(b *testing.B) {
		indicator, _ := talive.NewROC(50)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
