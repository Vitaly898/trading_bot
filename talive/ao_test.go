package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestAoDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/ao/output_default.csv", []int{1}, 7)
	indicator, _ := talive.NewAO()
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[AO] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestAoIdle(t *testing.T) {
	indicator, _ := talive.NewAO()
	var result []string
	for i := 0; i < 35; i++ {
		indicator.Next(&testCandle{
			high: float64(i + 1),
			low:  float64(i),
		})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	expected := make([]string, 35)
	for i := range expected {
		if i < 33 {
			expected[i] = "true"
		} else {
			expected[i] = "false"
		}
	}
	if !reflect.DeepEqual(result, expected) {
		t.Fatal(`[AO] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[AO] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestAoCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/ao/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewAO()
	for i := 0; i < 34; i++ {
		indicator.Next(candles[i])
	}
	CurrentVal := roundFloat(indicator.Current(candles[34])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][34], 8)
	if CurrentVal != expectedValue {
		t.Fatalf("[AO] wrong Current value %f, expected %f", CurrentVal, expectedValue)
	}
	NextVal := roundFloat(indicator.Next(candles[34])[0], 8)
	if NextVal != CurrentVal {
		t.Fatalf("[AO] Current value call broke Next value %f, expected %f", NextVal, expectedValue)
	}
}

func Benchmark_Ao_Init_Allocations(benchmark *testing.B) {
	benchmark.Run("AO", func(benchmark *testing.B) {
		for i := 0; i < benchmark.N; i++ {
			benchSink, _ = talive.NewAO()
		}
	})
}

func Benchmark_Ao_Next_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	benchmark.Run("AO", func(benchmark *testing.B) {
		indicator, _ := talive.NewAO()
		dataIndex := 0
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Ao_Current_Allocations(benchmark *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	benchmark.Run("AO", func(benchmark *testing.B) {
		indicator, _ := talive.NewAO()
		dataIndex := primeForCurrentBench(indicator, candles)
		benchmark.ResetTimer()
		for i := 0; i < benchmark.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
