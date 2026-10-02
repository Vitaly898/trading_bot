package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestDmiDefault(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/dmi/output_default.csv", []int{1, 2, 3}, 7)
	indicator, _ := talive.NewDMI(14)
	adxResult := make([]float64, len(candles))
	plusDIResult := make([]float64, len(candles))
	minusDIResult := make([]float64, len(candles))
	for i, candle := range candles {
		out := indicator.Next(candle)
		adxResult[i] = roundFloat(out[0], 7)
		plusDIResult[i] = roundFloat(out[1], 7)
		minusDIResult[i] = roundFloat(out[2], 7)
	}
	if !(reflect.DeepEqual(adxResult, expectedParsedData[0])) {
		t.Fatal(`[DMI(14)] ADX values didn't match `, difference(adxResult, expectedParsedData[0]))
	}
	if !(reflect.DeepEqual(plusDIResult, expectedParsedData[1])) {
		t.Fatal(`[DMI(14)] +DI values didn't match `, difference(plusDIResult, expectedParsedData[1]))
	}
	if !(reflect.DeepEqual(minusDIResult, expectedParsedData[2])) {
		t.Fatal(`[DMI(14)] -DI values didn't match `, difference(minusDIResult, expectedParsedData[2]))
	}
}

func TestDmiMin(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/dmi/output_min.csv", []int{1, 2, 3}, 7)
	indicator, _ := talive.NewDMI(2)
	adxResult := make([]float64, len(candles))
	plusDIResult := make([]float64, len(candles))
	minusDIResult := make([]float64, len(candles))
	for i, candle := range candles {
		out := indicator.Next(candle)
		adxResult[i] = roundFloat(out[0], 7)
		plusDIResult[i] = roundFloat(out[1], 7)
		minusDIResult[i] = roundFloat(out[2], 7)
	}
	if !(reflect.DeepEqual(adxResult, expectedParsedData[0])) {
		t.Fatal(`[DMI(2)] ADX values didn't match `, difference(adxResult, expectedParsedData[0]))
	}
	if !(reflect.DeepEqual(plusDIResult, expectedParsedData[1])) {
		t.Fatal(`[DMI(2)] +DI values didn't match `, difference(plusDIResult, expectedParsedData[1]))
	}
	if !(reflect.DeepEqual(minusDIResult, expectedParsedData[2])) {
		t.Fatal(`[DMI(2)] -DI values didn't match `, difference(minusDIResult, expectedParsedData[2]))
	}
}

func TestDmiIdle(t *testing.T) {
	indicator, _ := talive.NewDMI(3)
	var result []string
	for i := 0; i < 5; i++ {
		indicator.Next(&testCandle{close: float64(i)})
		if indicator.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "false", "false"}) {
		t.Fatal(`[DMI(3)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != indicator.IdlePeriod() {
		t.Fatalf("[DMI(3)] IdlePeriod() = %d, but IsIdle() was true %d times", indicator.IdlePeriod(), trueCount)
	}
}

func TestDmiCurrentVal(t *testing.T) {
	candles, _ := readCandles("test_data/input_data2.csv")
	expectedParsedData, _ := readData("test_data/dmi/output_default.csv", []int{1, 2, 3}, 8)
	indicator, _ := talive.NewDMI(14)
	for i := 0; i < 28; i++ {
		indicator.Next(candles[i])
	}
	out := indicator.Current(candles[28])
	currentADX := roundFloat(out[0], 8)
	currentPlusDI := roundFloat(out[1], 8)
	currentMinusDI := roundFloat(out[2], 8)
	expectedADX := roundFloat(expectedParsedData[0][28], 8)
	expectedPlusDI := roundFloat(expectedParsedData[1][28], 8)
	expectedMinusDI := roundFloat(expectedParsedData[2][28], 8)
	if currentADX != expectedADX {
		t.Fatalf("[DMI(14)] wrong Current ADX value %f, expected %f", currentADX, expectedADX)
	}
	if currentPlusDI != expectedPlusDI {
		t.Fatalf("[DMI(14)] wrong Current +DI value %f, expected %f", currentPlusDI, expectedPlusDI)
	}
	if currentMinusDI != expectedMinusDI {
		t.Fatalf("[DMI(14)] wrong Current -DI value %f, expected %f", currentMinusDI, expectedMinusDI)
	}
	nextOut := indicator.Next(candles[28])
	nextADX := roundFloat(nextOut[0], 8)
	nextPlusDI := roundFloat(nextOut[1], 8)
	nextMinusDI := roundFloat(nextOut[2], 8)
	if nextADX != currentADX {
		t.Fatalf("[DMI(14)] Current call broke Next ADX value %f, expected %f", nextADX, expectedADX)
	}
	if nextPlusDI != currentPlusDI {
		t.Fatalf("[DMI(14)] Current call broke Next +DI value %f, expected %f", nextPlusDI, expectedPlusDI)
	}
	if nextMinusDI != currentMinusDI {
		t.Fatalf("[DMI(14)] Current call broke Next -DI value %f, expected %f", nextMinusDI, expectedMinusDI)
	}
}

func Benchmark_DMI_Init_Allocations(b *testing.B) {
	b.Run("DMI(2)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewDMI(2)
		}
	})
	b.Run("DMI(14)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewDMI(14)
		}
	})
	b.Run("DMI(50)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			benchSink, _ = talive.NewDMI(50)
		}
	})
}

func Benchmark_DMI_Next_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("DMI(2)", func(b *testing.B) {
		indicator, _ := talive.NewDMI(2)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("DMI(14)", func(b *testing.B) {
		indicator, _ := talive.NewDMI(14)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
	b.Run("DMI(50)", func(b *testing.B) {
		indicator, _ := talive.NewDMI(50)
		dataIndex := 0
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Next(candles[dataIndex])
		}
	})
}

func Benchmark_DMI_Current_Allocations(b *testing.B) {
	candles, _ := readCandles("test_data/input_data2.csv")
	dataLen := len(candles)

	b.Run("DMI(2)", func(b *testing.B) {
		indicator, _ := talive.NewDMI(2)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("DMI(14)", func(b *testing.B) {
		indicator, _ := talive.NewDMI(14)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
	b.Run("DMI(50)", func(b *testing.B) {
		indicator, _ := talive.NewDMI(50)
		dataIndex := primeForCurrentBench(indicator, candles)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dataIndex = limitedDataIndex(dataIndex, dataLen)
			sliceDummy = indicator.Current(candles[dataIndex])
		}
	})
}
