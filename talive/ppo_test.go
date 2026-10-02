package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestPPODefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/ppo/output_default.csv", []int{1, 2, 3}, 7)
	indicator, _ := talive.NewPPO(12, 26, 9)
	result := [][]float64{
		make([]float64, len(candles)),
		make([]float64, len(candles)),
		make([]float64, len(candles)),
	}
	for i, candle := range candles {
		res := indicator.Next(candle)
		result[0][i] = roundFloat(res[2], 7)
		result[1][i] = roundFloat(res[0], 7)
		result[2][i] = roundFloat(res[1], 7)
	}

	if !reflect.DeepEqual(result[0], expectedParsedData[0]) {
		t.Fatal(`[PPO(12, 26, 9)] Hist values didn't match `, difference(result[0], expectedParsedData[0]))
	}
	if !reflect.DeepEqual(result[1], expectedParsedData[1]) {
		t.Fatal(`[PPO(12, 26, 9)] PPO values didn't match `, difference(result[1], expectedParsedData[1]))
	}
	if !reflect.DeepEqual(result[2], expectedParsedData[2]) {
		t.Fatal(`[PPO(12, 26, 9)] Signal values didn't match `, difference(result[2], expectedParsedData[2]))
	}
}

func TestPPOMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/ppo/output_min.csv", []int{1, 2, 3}, 7)
	indicator, _ := talive.NewPPO(2, 3, 2)
	result := [][]float64{
		make([]float64, len(candles)),
		make([]float64, len(candles)),
		make([]float64, len(candles)),
	}
	for i, candle := range candles {
		res := indicator.Next(candle)
		result[0][i] = roundFloat(res[2], 7)
		result[1][i] = roundFloat(res[0], 7)
		result[2][i] = roundFloat(res[1], 7)
	}

	if !reflect.DeepEqual(result[0], expectedParsedData[0]) {
		t.Fatal(`[PPO(2, 3, 2)] Hist values didn't match `, difference(result[0], expectedParsedData[0]))
	}
	if !reflect.DeepEqual(result[1], expectedParsedData[1]) {
		t.Fatal(`[PPO(2, 3, 2)] PPO values didn't match `, difference(result[1], expectedParsedData[1]))
	}
	if !reflect.DeepEqual(result[2], expectedParsedData[2]) {
		t.Fatal(`[PPO(2, 3, 2)] Signal values didn't match `, difference(result[2], expectedParsedData[2]))
	}
}

func TestPPOIdle(t *testing.T) {
	indicator, _ := talive.NewPPO(3, 4, 2)
	var result []string
	for i := 0; i < 6; i++ {
		indicator.Next(&testCandle{close: float64(i + 1)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "true", "false", "false"}) {
		t.Fatal(`[PPO(3, 4, 2)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[PPO(3, 4, 2)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestPPOCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/ppo/output_default.csv", []int{1, 2, 3}, 7)
	indicator, _ := talive.NewPPO(12, 26, 9)
	for i := 0; i < 34; i++ {
		indicator.Next(candles[i])
	}
	currResult := indicator.Current(candles[34])
	currPPO := currResult[0]
	currSignal := currResult[1]
	currHist := currResult[2]
	expectedHist := expectedParsedData[0][34]
	expectedPPO := expectedParsedData[1][34]
	expectedSignal := expectedParsedData[2][34]
	if roundFloat(currPPO, 7) != roundFloat(expectedPPO, 7) {
		t.Fatalf("[PPO(12, 26, 9)] wrong Current PPO value %f, expected %f", currPPO, expectedPPO)
	}
	if roundFloat(currSignal, 7) != roundFloat(expectedSignal, 7) {
		t.Fatalf("[PPO(12, 26, 9)] wrong Current Signal value %f, expected %f", currSignal, expectedSignal)
	}
	if roundFloat(currHist, 7) != roundFloat(expectedHist, 7) {
		t.Fatalf("[PPO(12, 26, 9)] wrong Current Hist value %f, expected %f", currHist, expectedHist)
	}
	nextResult := indicator.Next(candles[34])
	nextPPO := nextResult[0]
	nextSignal := nextResult[1]
	nextHist := nextResult[2]

	if roundFloat(nextPPO, 7) != roundFloat(expectedPPO, 7) {
		t.Fatalf("[PPO(12, 26, 9)] Current PPO value call broke Next PPO value %f, expected %f", nextPPO, expectedPPO)
	}
	if roundFloat(nextSignal, 7) != roundFloat(expectedSignal, 7) {
		t.Fatalf("[PPO(12, 26, 9)] Current Signal value call broke Next Signal value %f, expected %f", nextSignal, expectedSignal)
	}
	if roundFloat(nextHist, 7) != roundFloat(expectedHist, 7) {
		t.Fatalf("[PPO(12, 26, 9)] Current Hist value call broke Next Hist value %f, expected %f", nextHist, expectedHist)
	}
}

func TestPPOInvalidPeriod(t *testing.T) {
	if _, err := talive.NewPPO(1, 26, 9); err == nil {
		t.Fatal("NewPPO(1, 26, 9) should return an error")
	}
	if _, err := talive.NewPPO(12, 1, 9); err == nil {
		t.Fatal("NewPPO(12, 1, 9) should return an error")
	}
	if _, err := talive.NewPPO(12, 26, 1); err == nil {
		t.Fatal("NewPPO(12, 26, 1) should return an error")
	}
}

func Benchmark_PPO_Init_Allocations(b *testing.B) {
	b.Run("PPO(12,26,9)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewPPO(12, 26, 9)
		}
	})
	b.Run("PPO(2,3,2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewPPO(2, 3, 2)
		}
	})
	b.Run("PPO(100,200,15)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewPPO(100, 200, 15)
		}
	})
}

func Benchmark_PPO_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("PPO(12,26,9)", func(b *testing.B) {
		indicator, _ := talive.NewPPO(12, 26, 9)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("PPO(2,3,2)", func(b *testing.B) {
		indicator, _ := talive.NewPPO(2, 3, 2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("PPO(100,200,15)", func(b *testing.B) {
		indicator, _ := talive.NewPPO(100, 200, 15)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_PPO_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("PPO(12,26,9)", func(b *testing.B) {
		indicator, _ := talive.NewPPO(12, 26, 9)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("PPO(2,3,2)", func(b *testing.B) {
		indicator, _ := talive.NewPPO(2, 3, 2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("PPO(100,200,15)", func(b *testing.B) {
		indicator, _ := talive.NewPPO(100, 200, 15)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
