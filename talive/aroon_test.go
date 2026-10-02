package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestAroonDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/aroon/output_default.csv", []int{1, 2}, 7)
	indicator, _ := talive.NewAroon(14)
	result := [][]float64{
		make([]float64, len(candles)),
		make([]float64, len(candles)),
	}
	for i, candle := range candles {
		res := indicator.Next(candle)
		result[0][i] = roundFloat(res[0], 7)
		result[1][i] = roundFloat(res[1], 7)
	}

	if !reflect.DeepEqual(result[0], expectedParsedData[0]) {
		t.Fatal(`[Aroon(14)] Up values didn't match `, difference(result[0], expectedParsedData[0]))
	}
	if !reflect.DeepEqual(result[1], expectedParsedData[1]) {
		t.Fatal(`[Aroon(14)] Down values didn't match `, difference(result[1], expectedParsedData[1]))
	}
}

func TestAroonMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/aroon/output_min.csv", []int{1, 2}, 7)
	indicator, _ := talive.NewAroon(2)
	result := [][]float64{
		make([]float64, len(candles)),
		make([]float64, len(candles)),
	}
	for i, candle := range candles {
		res := indicator.Next(candle)
		result[0][i] = roundFloat(res[0], 7)
		result[1][i] = roundFloat(res[1], 7)
	}

	if !reflect.DeepEqual(result[0], expectedParsedData[0]) {
		t.Fatal(`[Aroon(2)] Up values didn't match `, difference(result[0], expectedParsedData[0]))
	}
	if !reflect.DeepEqual(result[1], expectedParsedData[1]) {
		t.Fatal(`[Aroon(2)] Down values didn't match `, difference(result[1], expectedParsedData[1]))
	}
}

func TestAroonIdle(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	indicator, _ := talive.NewAroon(3)
	var result []string
	for i := 0; i < 5; i++ {
		indicator.Next(candles[i])
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "false", "false"}) {
		t.Fatal(`[Aroon(3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[Aroon(3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestAroonCurrent(t *testing.T) {
	candles, _ := readCandles("test_data/input_data.csv")
	expectedParsedData, _ := readData("test_data/aroon/output_default.csv", []int{1, 2}, 8)
	indicator, _ := talive.NewAroon(14)
	for i := 0; i < 15; i++ {
		indicator.Next(candles[i])
	}

	currResult := indicator.Current(candles[15])
	currentUp := roundFloat(currResult[0], 8)
	currentDown := roundFloat(currResult[1], 8)
	expectedUp := roundFloat(expectedParsedData[0][15], 8)
	expectedDown := roundFloat(expectedParsedData[1][15], 8)
	if currentUp != expectedUp {
		t.Fatalf("[Aroon(14)] wrong Current Up value %f, expected %f", currentUp, expectedUp)
	}
	if currentDown != expectedDown {
		t.Fatalf("[Aroon(14)] wrong Current Down value %f, expected %f", currentDown, expectedDown)
	}

	nextResult := indicator.Next(candles[15])
	nextUp := roundFloat(nextResult[0], 8)
	nextDown := roundFloat(nextResult[1], 8)
	if nextUp != currentUp {
		t.Fatalf("[Aroon(14)] Current call broke Next Up value %f, expected %f", nextUp, expectedUp)
	}
	if nextDown != currentDown {
		t.Fatalf("[Aroon(14)] Current call broke Next Down value %f, expected %f", nextDown, expectedDown)
	}
}

func TestAroonInvalidPeriod(t *testing.T) {
	if _, err := talive.NewAroon(1); err == nil {
		t.Fatal("NewAroon(1) should return an error")
	}
}

func Benchmark_Aroon_Init_Allocations(b *testing.B) {
	b.Run("Aroon(14)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewAroon(14)
		}
	})
	b.Run("Aroon(2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewAroon(2)
		}
	})
}

func Benchmark_Aroon_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data.csv")
	dataLen := len(candles)

	b.Run("Aroon(14)", func(b *testing.B) {
		indicator, _ := talive.NewAroon(14)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("Aroon(2)", func(b *testing.B) {
		indicator, _ := talive.NewAroon(2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Aroon_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data.csv")
	dataLen := len(candles)

	b.Run("Aroon(14)", func(b *testing.B) {
		indicator, _ := talive.NewAroon(14)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("Aroon(2)", func(b *testing.B) {
		indicator, _ := talive.NewAroon(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
