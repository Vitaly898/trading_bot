package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestWilliamsDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/williams/output_default.csv", []int{1}, 7)
	indicator, _ := talive.NewWilliams(14)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[Williams(14)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestWilliamsMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/williams/output_min.csv", []int{1}, 7)
	indicator, _ := talive.NewWilliams(2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[Williams(2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestWilliamsIdle(t *testing.T) {
	indicator, _ := talive.NewWilliams(3)
	var result []string
	for i := 0; i < 4; i++ {
		indicator.Next(&testCandle{high: float64(i + 2), low: float64(i), close: float64(i + 1)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "false", "false"}) {
		t.Fatal(`[Williams(3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[Williams(3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestWilliamsCurrent(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/williams/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewWilliams(14)
	for i := 0; i < 14; i++ {
		indicator.Next(candles[i])
	}
	CurrentVal := roundFloat(indicator.Current(candles[14])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][14], 8)
	if CurrentVal != expectedValue {
		t.Fatalf("[Williams(14)] wrong Current value %f, expected %f", CurrentVal, expectedValue)
	}
	NextVal := roundFloat(indicator.Next(candles[14])[0], 8)
	if NextVal != CurrentVal {
		t.Fatalf("[Williams(14)] Current value broke Next value %f, expected %f", NextVal, expectedValue)
	}
}

func Benchmark_Williams_Init_Allocations(b *testing.B) {
	b.Run("Williams(2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewWilliams(2)
		}
	})
	b.Run("Williams(50)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewWilliams(50)
		}
	})
}

func Benchmark_Williams_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("Williams(2)", func(b *testing.B) {
		indicator, _ := talive.NewWilliams(2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("Williams(50)", func(b *testing.B) {
		indicator, _ := talive.NewWilliams(50)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Williams_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("Williams(2)", func(b *testing.B) {
		indicator, _ := talive.NewWilliams(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("Williams(50)", func(b *testing.B) {
		indicator, _ := talive.NewWilliams(50)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
