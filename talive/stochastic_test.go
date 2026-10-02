package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestStochasticDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/stochastic/output_default.csv", []int{1, 2}, 8)
	indicator, _ := talive.NewStochastic(14, 1, 3)
	result := [][]float64{
		make([]float64, len(candles)),
		make([]float64, len(candles)),
	}
	for i, candle := range candles {
		res := indicator.Next(candle)
		result[0][i] = roundFloat(res[0], 8)
		result[1][i] = roundFloat(res[1], 8)
	}
	if !(reflect.DeepEqual(result[0], expectedParsedData[0])) {
		t.Fatal(`[Stochastic(14, 1, 3)] values didn't match `, difference(result[0], expectedParsedData[0]))
	}
	if !(reflect.DeepEqual(result[1], expectedParsedData[1])) {
		t.Fatal(`[Stochastic(14, 1, 3)] values didn't match `, difference(result[1], expectedParsedData[1]))
	}
}

func TestStochasticMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/stochastic/output_min.csv", []int{1, 2}, 7)
	indicator, _ := talive.NewStochastic(2, 1, 2)
	result := [][]float64{
		make([]float64, len(candles)),
		make([]float64, len(candles)),
	}
	for i, candle := range candles {
		res := indicator.Next(candle)
		result[0][i] = roundFloat(res[0], 7)
		result[1][i] = roundFloat(res[1], 7)
	}
	if !(reflect.DeepEqual(result[0], expectedParsedData[0])) {
		t.Fatal(`[Stochastic(14, 1, 3)] values didn't match `, difference(result[0], expectedParsedData[0]))
	}
	if !(reflect.DeepEqual(result[1], expectedParsedData[1])) {
		t.Fatal(`[Stochastic(14, 1, 3)] values didn't match `, difference(result[1], expectedParsedData[1]))
	}
}

func TestStochasticIdle(t *testing.T) {
	indicator, _ := talive.NewStochastic(5, 3, 4)
	var result []string
	for i := 0; i < 12; i++ {
		indicator.Next(&testCandle{close: float64(i + 1)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "true", "true", "true", "true", "false", "false", "false", "false", "false"}) {
		t.Fatal(`[Stochastic(5,3,4)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[Stochastic(5,3,4)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestRsiCurrent(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/stochastic/output_default.csv", []int{1, 2}, 8)
	indicator, _ := talive.NewStochastic(14, 1, 3)
	for i := 0; i < 14; i++ {
		indicator.Next(candles[i])
	}
	CurrentValK := roundFloat(indicator.Current(candles[14])[0], 8)
	CurrentValD := roundFloat(indicator.Current(candles[14])[1], 8)
	expectedValueK := roundFloat(expectedParsedData[0][14], 8)
	expectedValueD := roundFloat(expectedParsedData[1][14], 8)

	if CurrentValK != expectedValueK {
		t.Fatalf("[Stochastic(14, 1 3)] wrong Current K value %f, expected %f", CurrentValK, expectedValueK)
	}
	if CurrentValD != expectedValueD {
		t.Fatalf("[Stochastic(14, 1 3)] wrong Current D value %f, expected %f", CurrentValD, expectedValueD)
	}

	NextVals := indicator.Next(candles[14])
	nextK := roundFloat(NextVals[0], 8)
	nextD := roundFloat(NextVals[1], 8)
	if nextK != expectedValueK {
		t.Fatalf("[Stochastic(14, 1 3)] Current value broke Next K value %f, expected %f", CurrentValK, expectedValueK)
	}
	if nextD != expectedValueD {
		t.Fatalf("[Stochastic(14, 1 3)] Current value broke  Next D value %f, expected %f", CurrentValD, expectedValueD)
	}
}

func Benchmark_Stochastic_Init_Allocations(b *testing.B) {
	b.Run("Stochastic(2, 1, 2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewStochastic(2, 1, 2)
		}
	})
	b.Run("Stochastic(14, 1, 3)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewStochastic(14, 1, 3)
		}
	})
	b.Run("Stochastic(50, 10, 20)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewStochastic(50, 10, 20)
		}
	})
}

func Benchmark_Stochastic_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("Stochastic(2, 1, 2)", func(b *testing.B) {
		indicator, _ := talive.NewStochastic(2, 1, 2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("Stochastic(14, 1, 3)", func(b *testing.B) {
		indicator, _ := talive.NewStochastic(14, 1, 3)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("Stochastic(50, 10, 20)", func(b *testing.B) {
		indicator, _ := talive.NewStochastic(50, 10, 20)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_Stochastic_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("Stochastic(2, 1, 2)", func(b *testing.B) {
		indicator, _ := talive.NewStochastic(2, 1, 2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("Stochastic(14, 1, 3)", func(b *testing.B) {
		indicator, _ := talive.NewStochastic(14, 1, 3)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("Stochastic(50, 10, 20)", func(b *testing.B) {
		indicator, _ := talive.NewStochastic(50, 10, 20)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
