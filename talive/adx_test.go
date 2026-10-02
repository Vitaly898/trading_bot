package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestAdxDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/adx/output_default.csv", []int{1}, 7)
	indicator, _ := talive.NewADX(14)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[ADX(14)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestAdxMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/adx/output_min.csv", []int{1}, 7)
	indicator, _ := talive.NewADX(2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[ADX(2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestAdxIdle(t *testing.T) {
	indicator, _ := talive.NewADX(3)
	var result []string
	for i := 0; i < 7; i++ {
		indicator.Next(&testCandle{close: float64(i)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	// Period=3: IdlePeriod = 2*3-1 = 5, first output at 6th call
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "true", "true", "false", "false"}) {
		t.Fatal(`[ADX(3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[ADX(3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestAdxCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/adx/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewADX(14)
	for i := 0; i < 28; i++ {
		indicator.Next(candles[i])
	}
	CurrentVal := roundFloat(indicator.Current(candles[28])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][28], 8)
	if CurrentVal != expectedValue {
		t.Fatalf("[ADX(14)] wrong Current value %f, expected %f", CurrentVal, expectedValue)
	}
	NextVal := roundFloat(indicator.Next(candles[28])[0], 8)
	if NextVal != CurrentVal {
		t.Fatalf("[ADX(14)] Current value call broke Next value %f, expected %f", NextVal, expectedValue)
	}
}

func Benchmark_ADX_Init_Allocations(b *testing.B) {
	b.Run("ADX(2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewADX(2)
		}
	})
	b.Run("ADX(14)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewADX(14)
		}
	})
	b.Run("ADX(50)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewADX(50)
		}
	})
}

func Benchmark_ADX_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("ADX(2)", func(b *testing.B) {
		indicator, _ := talive.NewADX(2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("ADX(14)", func(b *testing.B) {
		indicator, _ := talive.NewADX(14)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("ADX(50)", func(b *testing.B) {
		indicator, _ := talive.NewADX(50)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_ADX_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("ADX(2)", func(b *testing.B) {
		indicator, _ := talive.NewADX(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("ADX(14)", func(b *testing.B) {
		indicator, _ := talive.NewADX(14)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("ADX(50)", func(b *testing.B) {
		indicator, _ := talive.NewADX(50)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
