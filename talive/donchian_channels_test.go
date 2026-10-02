package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestDonchianChannelsDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/donchian_channels/output_default.csv", []int{1, 2, 3}, 4)
	indicator, _ := talive.NewDonchianChannels(20)
	result := [][]float64{
		make([]float64, len(candles)),
		make([]float64, len(candles)),
		make([]float64, len(candles)),
	}
	for i, candle := range candles {
		res := indicator.Next(candle)
		result[0][i] = roundFloat(res[1], 4)
		result[1][i] = roundFloat(res[0], 4)
		result[2][i] = roundFloat(res[2], 4)
	}

	if !reflect.DeepEqual(result[0], expectedParsedData[0]) {
		t.Fatal(`[DonchianChannels(20)] Mid values didn't match `, difference(result[0], expectedParsedData[0]))
	}
	if !reflect.DeepEqual(result[1], expectedParsedData[1]) {
		t.Fatal(`[DonchianChannels(20)] Upper values didn't match `, difference(result[1], expectedParsedData[1]))
	}
	if !reflect.DeepEqual(result[2], expectedParsedData[2]) {
		t.Fatal(`[DonchianChannels(20)] Lower values didn't match `, difference(result[2], expectedParsedData[2]))
	}
}

func TestDonchianChannelsMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/donchian_channels/output_min.csv", []int{1, 2, 3}, 5)
	indicator, _ := talive.NewDonchianChannels(2)
	result := [][]float64{
		make([]float64, len(candles)),
		make([]float64, len(candles)),
		make([]float64, len(candles)),
	}
	for i, candle := range candles {
		res := indicator.Next(candle)
		result[0][i] = roundFloat(res[1], 5)
		result[1][i] = roundFloat(res[0], 5)
		result[2][i] = roundFloat(res[2], 5)
	}

	if !reflect.DeepEqual(result[0], expectedParsedData[0]) {
		t.Fatal(`[DonchianChannels(2)] Mid values didn't match `, difference(result[0], expectedParsedData[0]))
	}
	if !reflect.DeepEqual(result[1], expectedParsedData[1]) {
		t.Fatal(`[DonchianChannels(2)] Upper values didn't match `, difference(result[1], expectedParsedData[1]))
	}
	if !reflect.DeepEqual(result[2], expectedParsedData[2]) {
		t.Fatal(`[DonchianChannels(2)] Lower values didn't match `, difference(result[2], expectedParsedData[2]))
	}
}

func TestDonchianChannelsIdle(t *testing.T) {
	indicator, _ := talive.NewDonchianChannels(5)
	var result []string
	for i := 0; i < 6; i++ {
		indicator.Next(&testCandle{high: float64(i + 2), low: float64(i)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "true", "false", "false"}) {
		t.Fatal(`[DonchianChannels(5)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[DonchianChannels(5)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestDonchianChannelsCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/donchian_channels/output_default.csv", []int{1, 2, 3}, 8)
	indicator, _ := talive.NewDonchianChannels(20)
	for i := 0; i < 22; i++ {
		indicator.Next(candles[i])
	}
	currResult := indicator.Current(candles[22])
	currUpper := currResult[0]
	currMid := currResult[1]
	currLower := currResult[2]
	expectedMid := expectedParsedData[0][22]
	expectedUpper := expectedParsedData[1][22]
	expectedLower := expectedParsedData[2][22]
	if roundFloat(currUpper, 8) != roundFloat(expectedUpper, 8) {
		t.Fatalf("[DonchianChannels(20)] wrong Current Upper value %f, expected %f", currUpper, expectedUpper)
	}
	if roundFloat(currMid, 8) != roundFloat(expectedMid, 8) {
		t.Fatalf("[DonchianChannels(20)] wrong Current Mid value %f, expected %f", currMid, expectedMid)
	}
	if roundFloat(currLower, 8) != roundFloat(expectedLower, 8) {
		t.Fatalf("[DonchianChannels(20)] wrong Current Lower value %f, expected %f", currLower, expectedLower)
	}
	nextResult := indicator.Next(candles[22])
	nextUpper := nextResult[0]
	nextMid := nextResult[1]
	nextLower := nextResult[2]
	if roundFloat(nextUpper, 8) != roundFloat(expectedUpper, 8) {
		t.Fatalf("[DonchianChannels(20)] Current Upper value call broke Next Upper value %f, expected %f", nextUpper, expectedUpper)
	}
	if roundFloat(nextMid, 8) != roundFloat(expectedMid, 8) {
		t.Fatalf("[DonchianChannels(20)] Current Mid value call broke Next Mid value %f, expected %f", nextMid, expectedMid)
	}
	if roundFloat(nextLower, 8) != roundFloat(expectedLower, 8) {
		t.Fatalf("[DonchianChannels(20)] Current Lower value call broke Next Lower value %f, expected %f", nextLower, expectedLower)
	}
}

func Benchmark_DonchianChannels_Init_Allocations(b *testing.B) {
	b.Run("DonchianChannels(20)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewDonchianChannels(20)
		}
	})
	b.Run("DonchianChannels(2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewDonchianChannels(2)
		}
	})
}

func Benchmark_DonchianChannels_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("DonchianChannels(20)", func(b *testing.B) {
		indicator, _ := talive.NewDonchianChannels(20)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("DonchianChannels(2)", func(b *testing.B) {
		indicator, _ := talive.NewDonchianChannels(2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_DonchianChannels_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)
	b.Run("DonchianChannels(20)", func(b *testing.B) {
		indicator, _ := talive.NewDonchianChannels(20)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("DonchianChannels(2)", func(b *testing.B) {
		indicator, _ := talive.NewDonchianChannels(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
