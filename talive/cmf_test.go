package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestCMFDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/cmf/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewCMF(20)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 8)
	}
	if !reflect.DeepEqual(result, expectedParsedData[0]) {
		t.Fatal(`[CMF(20)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestCMFMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/cmf/output_min.csv", []int{1}, 8)
	indicator, _ := talive.NewCMF(2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 8)
	}
	if !reflect.DeepEqual(result, expectedParsedData[0]) {
		t.Fatal(`[CMF(2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestCMFIdle(t *testing.T) {
	indicator, _ := talive.NewCMF(3)
	var result []string
	for i := 0; i < 4; i++ {
		indicator.Next(&testCandle{
			high:   float64(i + 2),
			low:    float64(i),
			close:  float64(i + 1),
			volume: float64(i + 1),
		})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "false", "false"}) {
		t.Fatal(`[CMF(3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[CMF(3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestCMFCurrent(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/cmf/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewCMF(20)
	for i := 0; i < 20; i++ {
		indicator.Next(candles[i])
	}
	currentValue := roundFloat(indicator.Current(candles[20])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][20], 8)
	if currentValue != expectedValue {
		t.Fatalf("[CMF(20)] wrong Current value %f, expected %f", currentValue, expectedValue)
	}
	nextValue := roundFloat(indicator.Next(candles[20])[0], 8)
	if nextValue != currentValue {
		t.Fatalf("[CMF(20)] Current value call broke Next value %f, expected %f", nextValue, expectedValue)
	}
}

func TestCMFInvalidPeriod(t *testing.T) {
	if _, err := talive.NewCMF(0); err == nil {
		t.Fatal("NewCMF(0) should return an error")
	}
}

func Benchmark_CMF_Init_Allocations(b *testing.B) {
	b.Run("CMF(20)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewCMF(20)
		}
	})
	b.Run("CMF(2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewCMF(2)
		}
	})
	b.Run("CMF(50)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewCMF(50)
		}
	})
}

func Benchmark_CMF_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("CMF(20)", func(b *testing.B) {
		indicator, _ := talive.NewCMF(20)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("CMF(2)", func(b *testing.B) {
		indicator, _ := talive.NewCMF(2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("CMF(50)", func(b *testing.B) {
		indicator, _ := talive.NewCMF(50)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_CMF_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("CMF(20)", func(b *testing.B) {
		indicator, _ := talive.NewCMF(20)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("CMF(2)", func(b *testing.B) {
		indicator, _ := talive.NewCMF(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("CMF(50)", func(b *testing.B) {
		indicator, _ := talive.NewCMF(50)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
