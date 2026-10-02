package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestADRDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/adr/output_default.csv", []int{1}, 7)
	indicator, _ := talive.NewADR(14)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[ADR(14)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestADRMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/adr/output_min.csv", []int{1}, 7)
	indicator, _ := talive.NewADR(2)
	result := make([]float64, len(candles))
	for i, candle := range candles {
		result[i] = roundFloat(indicator.Next(candle)[0], 7)
	}
	if !(reflect.DeepEqual(result, expectedParsedData[0])) {
		t.Fatal(`[ADR(2)] values didn't match `, difference(result, expectedParsedData[0]))
	}
}

func TestADRIdle(t *testing.T) {
	indicator, _ := talive.NewADR(3)
	var result []string
	for i := 0; i < 4; i++ {
		indicator.Next(&testCandle{high: float64(i + 2), low: float64(i)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "false", "false"}) {
		t.Fatal(`[ADR(3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[ADR(3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestADRCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/adr/output_default.csv", []int{1}, 8)
	indicator, _ := talive.NewADR(14)
	for i := 0; i < 14; i++ {
		indicator.Next(candles[i])
	}
	CurrentVal := roundFloat(indicator.Current(candles[14])[0], 8)
	expectedValue := roundFloat(expectedParsedData[0][14], 8)
	if CurrentVal != expectedValue {
		t.Fatalf("[ADR(14)] wrong Current value %f, expected %f", CurrentVal, expectedValue)
	}
	NextVal := roundFloat(indicator.Next(candles[14])[0], 8)
	if NextVal != CurrentVal {
		t.Fatalf("[ADR(14)] Current value call broke Next value %f, expected %f", NextVal, expectedValue)
	}
}

func TestADRInvalidPeriod(t *testing.T) {
	if _, err := talive.NewADR(0); err == nil {
		t.Fatal("NewADR(0) should return an error")
	}
}

func Benchmark_ADR_Init_Allocations(b *testing.B) {
	b.Run("ADR(2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewADR(2)
		}
	})
	b.Run("ADR(50)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewADR(50)
		}
	})
}

func Benchmark_ADR_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("ADR(2)", func(b *testing.B) {
		indicator, _ := talive.NewADR(2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("ADR(50)", func(b *testing.B) {
		indicator, _ := talive.NewADR(50)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_ADR_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("ADR(2)", func(b *testing.B) {
		indicator, _ := talive.NewADR(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("ADR(50)", func(b *testing.B) {
		indicator, _ := talive.NewADR(50)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
