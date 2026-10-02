package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestSupertrendDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/supertrend/output_default.csv", []int{1, 2}, 7)
	indicator, _ := talive.NewSupertrend(10, 3)
	supertrendCol := make([]float64, len(candles))
	directionCol := make([]float64, len(candles))
	for i, candle := range candles {
		out := indicator.Next(candle)
		supertrendCol[i] = roundFloat(out[0], 7)
		directionCol[i] = roundFloat(out[1], 7)
	}
	if !reflect.DeepEqual(supertrendCol, expectedParsedData[0]) {
		t.Fatal(`[Supertrend(10,3)] supertrend values didn't match `, difference(supertrendCol, expectedParsedData[0]))
	}
	if !reflect.DeepEqual(directionCol, expectedParsedData[1]) {
		t.Fatal(`[Supertrend(10,3)] direction values didn't match `, difference(directionCol, expectedParsedData[1]))
	}
}

func TestSupertrendMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/supertrend/output_min.csv", []int{1, 2}, 7)
	indicator, _ := talive.NewSupertrend(2, 1)
	supertrendCol := make([]float64, len(candles))
	directionCol := make([]float64, len(candles))
	for i, candle := range candles {
		out := indicator.Next(candle)
		supertrendCol[i] = roundFloat(out[0], 7)
		directionCol[i] = roundFloat(out[1], 7)
	}
	if !reflect.DeepEqual(supertrendCol, expectedParsedData[0]) {
		t.Fatal(`[Supertrend(2,1)] supertrend values didn't match `, difference(supertrendCol, expectedParsedData[0]))
	}
	if !reflect.DeepEqual(directionCol, expectedParsedData[1]) {
		t.Fatal(`[Supertrend(2,1)] direction values didn't match `, difference(directionCol, expectedParsedData[1]))
	}
}

func TestSupertrendIdle(t *testing.T) {
	indicator, _ := talive.NewSupertrend(3, 2)
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
		t.Fatal(`[Supertrend(3,2)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[Supertrend(3,2)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestSupertrendCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/supertrend/output_default.csv", []int{1, 2}, 8)
	indicator, _ := talive.NewSupertrend(10, 3)
	for i := 0; i < 10; i++ {
		indicator.Next(candles[i])
	}
	currentOut := indicator.Current(candles[10])
	currentSupertrend := roundFloat(currentOut[0], 8)
	currentDirection := roundFloat(currentOut[1], 8)
	expectedSupertrend := roundFloat(expectedParsedData[0][10], 8)
	expectedDirection := roundFloat(expectedParsedData[1][10], 8)
	if currentSupertrend != expectedSupertrend {
		t.Fatalf("[Supertrend(10,3)] wrong Current supertrend %f, expected %f", currentSupertrend, expectedSupertrend)
	}
	if currentDirection != expectedDirection {
		t.Fatalf("[Supertrend(10,3)] wrong Current direction %f, expected %f", currentDirection, expectedDirection)
	}
	nextOut := indicator.Next(candles[10])
	nextSupertrend := roundFloat(nextOut[0], 8)
	nextDirection := roundFloat(nextOut[1], 8)
	if nextSupertrend != currentSupertrend {
		t.Fatalf("[Supertrend(10,3)] Current call broke Next supertrend %f, expected %f", nextSupertrend, currentSupertrend)
	}
	if nextDirection != currentDirection {
		t.Fatalf("[Supertrend(10,3)] Current call broke Next direction %f, expected %f", nextDirection, currentDirection)
	}
}

func Benchmark_Supertrend_Init_Allocations(b *testing.B) {
	b.Run("Supertrend(2,1)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewSupertrend(2, 1)
		}
	})
	b.Run("Supertrend(10,3)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewSupertrend(10, 3)
		}
	})
}

func Benchmark_Supertrend_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("Supertrend(2,1)", func(b *testing.B) {
		indicator, _ := talive.NewSupertrend(2, 1)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("Supertrend(10,3)", func(b *testing.B) {
		indicator, _ := talive.NewSupertrend(10, 3)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Supertrend_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("Supertrend(2,1)", func(b *testing.B) {
		indicator, _ := talive.NewSupertrend(2, 1)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("Supertrend(10,3)", func(b *testing.B) {
		indicator, _ := talive.NewSupertrend(10, 3)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
