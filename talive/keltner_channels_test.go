package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestKeltnerChannelsDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/keltner_channels/output_default.csv", []int{1, 2, 3}, 4)
	indicator, _ := talive.NewKeltnerChannels(20, 10, 2.0)
	result := [][]float64{
		make([]float64, len(candles)),
		make([]float64, len(candles)),
		make([]float64, len(candles)),
	}
	for i, candle := range candles {
		res := indicator.Next(candle)
		result[0][i] = roundFloat(res[0], 4)
		result[1][i] = roundFloat(res[1], 4)
		result[2][i] = roundFloat(res[2], 4)
	}

	if !reflect.DeepEqual(result[0], expectedParsedData[0]) {
		t.Fatal(`[KeltnerChannels(20,10,2.0)] Upper values didn't match `, difference(result[0], expectedParsedData[0]))
	}
	if !reflect.DeepEqual(result[1], expectedParsedData[1]) {
		t.Fatal(`[KeltnerChannels(20,10,2.0)] Basis values didn't match `, difference(result[1], expectedParsedData[1]))
	}
	if !reflect.DeepEqual(result[2], expectedParsedData[2]) {
		t.Fatal(`[KeltnerChannels(20,10,2.0)] Lower values didn't match `, difference(result[2], expectedParsedData[2]))
	}
}

func TestKeltnerChannelsMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/keltner_channels/output_min.csv", []int{1, 2, 3}, 5)
	indicator, _ := talive.NewKeltnerChannels(2, 2, 1.0)
	result := [][]float64{
		make([]float64, len(candles)),
		make([]float64, len(candles)),
		make([]float64, len(candles)),
	}
	for i, candle := range candles {
		res := indicator.Next(candle)
		result[0][i] = roundFloat(res[0], 5)
		result[1][i] = roundFloat(res[1], 5)
		result[2][i] = roundFloat(res[2], 5)
	}

	if !reflect.DeepEqual(result[0], expectedParsedData[0]) {
		t.Fatal(`[KeltnerChannels(2,2,1.0)] Upper values didn't match `, difference(result[0], expectedParsedData[0]))
	}
	if !reflect.DeepEqual(result[1], expectedParsedData[1]) {
		t.Fatal(`[KeltnerChannels(2,2,1.0)] Basis values didn't match `, difference(result[1], expectedParsedData[1]))
	}
	if !reflect.DeepEqual(result[2], expectedParsedData[2]) {
		t.Fatal(`[KeltnerChannels(2,2,1.0)] Lower values didn't match `, difference(result[2], expectedParsedData[2]))
	}
}

func TestKeltnerChannelsIdle(t *testing.T) {
	indicator, _ := talive.NewKeltnerChannels(5, 5, 2.0)
	var result []string
	for i := 0; i < 6; i++ {
		indicator.Next(&testCandle{high: float64(i + 2), low: float64(i), close: float64(i + 1)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "true", "false", "false"}) {
		t.Fatal(`[KeltnerChannels(5,5,2.0)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[KeltnerChannels(5,5,2.0)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestKeltnerChannelsCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/keltner_channels/output_default.csv", []int{1, 2, 3}, 8)
	indicator, _ := talive.NewKeltnerChannels(20, 10, 2.0)
	for i := 0; i < 22; i++ {
		indicator.Next(candles[i])
	}
	currResult := indicator.Current(candles[22])
	currUpper := currResult[0]
	currBasis := currResult[1]
	currLower := currResult[2]
	expectedUpper := expectedParsedData[0][22]
	expectedBasis := expectedParsedData[1][22]
	expectedLower := expectedParsedData[2][22]
	if roundFloat(currUpper, 8) != roundFloat(expectedUpper, 8) {
		t.Fatalf("[KeltnerChannels(20,10,2.0)] wrong Current Upper value %f, expected %f", currUpper, expectedUpper)
	}
	if roundFloat(currBasis, 8) != roundFloat(expectedBasis, 8) {
		t.Fatalf("[KeltnerChannels(20,10,2.0)] wrong Current Basis value %f, expected %f", currBasis, expectedBasis)
	}
	if roundFloat(currLower, 8) != roundFloat(expectedLower, 8) {
		t.Fatalf("[KeltnerChannels(20,10,2.0)] wrong Current Lower value %f, expected %f", currLower, expectedLower)
	}
	nextResult := indicator.Next(candles[22])
	nextUpper := nextResult[0]
	nextBasis := nextResult[1]
	nextLower := nextResult[2]
	if roundFloat(nextUpper, 8) != roundFloat(expectedUpper, 8) {
		t.Fatalf("[KeltnerChannels(20,10,2.0)] Current Upper call broke Next Upper %f, expected %f", nextUpper, expectedUpper)
	}
	if roundFloat(nextBasis, 8) != roundFloat(expectedBasis, 8) {
		t.Fatalf("[KeltnerChannels(20,10,2.0)] Current Basis call broke Next Basis %f, expected %f", nextBasis, expectedBasis)
	}
	if roundFloat(nextLower, 8) != roundFloat(expectedLower, 8) {
		t.Fatalf("[KeltnerChannels(20,10,2.0)] Current Lower call broke Next Lower %f, expected %f", nextLower, expectedLower)
	}
}

func Benchmark_KeltnerChannels_Init_Allocations(b *testing.B) {
	b.Run("KeltnerChannels(20,10,2.0)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewKeltnerChannels(20, 10, 2.0)
		}
	})
	b.Run("KeltnerChannels(2,2,1.0)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewKeltnerChannels(2, 2, 1.0)
		}
	})
	b.Run("KeltnerChannels(20,10,2.0,SMA)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			kc, _ := talive.NewKeltnerChannels(20, 10, 2.0)
			kc.WithMA(talive.UseSMA)
			benchSink = kc
		}
	})
}

func Benchmark_KeltnerChannels_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("KeltnerChannels(20,10,2.0)", func(b *testing.B) {
		indicator, _ := talive.NewKeltnerChannels(20, 10, 2.0)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("KeltnerChannels(2,2,1.0)", func(b *testing.B) {
		indicator, _ := talive.NewKeltnerChannels(2, 2, 1.0)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("KeltnerChannels(20,10,2.0,SMA)", func(b *testing.B) {
		indicator, _ := talive.NewKeltnerChannels(20, 10, 2.0)
		indicator.WithMA(talive.UseSMA)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_KeltnerChannels_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("KeltnerChannels(20,10,2.0)", func(b *testing.B) {
		indicator, _ := talive.NewKeltnerChannels(20, 10, 2.0)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("KeltnerChannels(2,2,1.0)", func(b *testing.B) {
		indicator, _ := talive.NewKeltnerChannels(2, 2, 1.0)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
