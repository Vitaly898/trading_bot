package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestBullBearPowerDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/bull_bear_power/output_default.csv", []int{1}, 7)
	indicator, _ := talive.NewBullBearPower(13)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[BBPower(13)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestBullBearPowerMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/bull_bear_power/output_min.csv", []int{1}, 7)
	indicator, _ := talive.NewBullBearPower(2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[BBPower(2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestBullBearPowerIdle(t *testing.T) {
	indicator, _ := talive.NewBullBearPower(3)
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
		t.Fatal(`[BBPower(3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[BBPower(3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestBullBearPowerCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/bull_bear_power/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewBullBearPower(13)
	for i := 0; i < 13; i++ {
		indicator.Next(candles[i])
	}
	CurrentVal := roundFloat(indicator.Current(candles[13])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][13], 8)
	if CurrentVal != expectedValue {
		t.Fatalf("[BBPower(13)] wrong Current value %f, expected %f", CurrentVal, expectedValue)
	}
	NextVal := roundFloat(indicator.Next(candles[13])[0], 8)
	if NextVal != CurrentVal {
		t.Fatalf("[BBPower(13)] Current value call broke Next value %f, expected %f", NextVal, expectedValue)
	}
}

func Benchmark_BullBearPower_Init_Allocations(b *testing.B) {
	b.Run("BBPower(2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewBullBearPower(2)
		}
	})
	b.Run("BBPower(50)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewBullBearPower(50)
		}
	})
}

func Benchmark_BullBearPower_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("BBPower(2)", func(b *testing.B) {
		indicator, _ := talive.NewBullBearPower(2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("BBPower(50)", func(b *testing.B) {
		indicator, _ := talive.NewBullBearPower(50)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_BullBearPower_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("BBPower(2)", func(b *testing.B) {
		indicator, _ := talive.NewBullBearPower(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("BBPower(50)", func(b *testing.B) {
		indicator, _ := talive.NewBullBearPower(50)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
