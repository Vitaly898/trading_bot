package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestKamaDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/kama/output_default.csv", []int{1}, 6)
	indicator, _ := talive.NewKAMA(10, 2, 30)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 6)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[KAMA(10,2,30)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestKamaMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/kama/output_min.csv", []int{1}, 6)
	indicator, _ := talive.NewKAMA(2, 2, 4)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 6)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[KAMA(2,2,4)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestKamaIdle(t *testing.T) {
	indicator, _ := talive.NewKAMA(3, 2, 30)
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
		t.Fatal(`[KAMA(3,2,30)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[KAMA(3,2,30)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestKamaCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/kama/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewKAMA(10, 2, 30)
	for i := 0; i < 20; i++ {
		indicator.Next(candles[i])
	}
	currentValue := roundFloat(indicator.Current(candles[20])[0], 6)
	expectedValue := roundFloat(expectedParsedData[0][20], 6)
	if currentValue != expectedValue {
		t.Fatalf("[KAMA(10,2,30)] wrong Current value %f, expected %f", currentValue, expectedValue)
	}
	nextValue := roundFloat(indicator.Next(candles[20])[0], 6)
	if nextValue != currentValue {
		t.Fatalf("[KAMA(10,2,30)] Current value call broke Next value %f, expected %f", nextValue, expectedValue)
	}
}

func TestKamaInvalidParams(t *testing.T) {
	if _, err := talive.NewKAMA(1, 2, 30); err == nil {
		t.Fatal("expected error for period < 2")
	}
	if _, err := talive.NewKAMA(10, 0, 30); err == nil {
		t.Fatal("expected error for fastPeriod < 1")
	}
	if _, err := talive.NewKAMA(10, 2, 0); err == nil {
		t.Fatal("expected error for slowPeriod < 1")
	}
	if _, err := talive.NewKAMA(10, 30, 2); err == nil {
		t.Fatal("expected error when slowPeriod <= fastPeriod")
	}
}

func Benchmark_Kama_Init_Allocations(b *testing.B) {
	b.Run("KAMA(2,2,4)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewKAMA(2, 2, 4)
		}
	})
	b.Run("KAMA(10,2,30)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewKAMA(10, 2, 30)
		}
	})
	b.Run("KAMA(50,2,30)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewKAMA(50, 2, 30)
		}
	})
}

func Benchmark_Kama_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("KAMA(2,2,4)", func(b *testing.B) {
		indicator, _ := talive.NewKAMA(2, 2, 4)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("KAMA(10,2,30)", func(b *testing.B) {
		indicator, _ := talive.NewKAMA(10, 2, 30)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("KAMA(50,2,30)", func(b *testing.B) {
		indicator, _ := talive.NewKAMA(50, 2, 30)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Kama_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("KAMA(2,2,4)", func(b *testing.B) {
		indicator, _ := talive.NewKAMA(2, 2, 4)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("KAMA(10,2,30)", func(b *testing.B) {
		indicator, _ := talive.NewKAMA(10, 2, 30)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("KAMA(50,2,30)", func(b *testing.B) {
		indicator, _ := talive.NewKAMA(50, 2, 30)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
