package talive_test

import (
	"reflect"
	"testing"

	"github.com/MoroZvlg/talive"
)

func TestStdDev(t *testing.T) {
	inputData := []float64{1, 2, 7, 3, 12, 9}
	variance, _ := talive.NewStdDev(5, 1.0)
	var result []float64
	for _, data := range inputData {
		result = append(result, roundFloat(variance.Next(&testCandle{close: data})[0], 2))
	}
	expectedData := []float64{0.0, 0.0, 0.0, 0.0, 4.05, 3.72}
	if !(reflect.DeepEqual(result, expectedData)) {
		t.Fatal(`[Variance(5)] values didn't match `, difference(result, expectedData))
	}
}

func TestVarianceNext(t *testing.T) {
	inputData := []float64{1, 2, 7, 3, 12, 9}
	variance, _ := talive.NewVariance(5)
	var result []float64
	for _, data := range inputData {
		result = append(result, roundFloat(variance.Next(&testCandle{close: data})[0], 2))
	}
	expectedData := []float64{0.0, 0.0, 0.0, 0.0, 16.4, 13.84}
	if !(reflect.DeepEqual(result, expectedData)) {
		t.Fatal(`[Variance(5)] values didn't match `, difference(result, expectedData))
	}
}

func TestVarianceIsIdle(t *testing.T) {
	variance, _ := talive.NewVariance(4)
	var result []string
	for i := 0; i < 6; i++ {
		variance.Next(&testCandle{close: float64(i)})
		if variance.IsIdle() {
			result = append(result, "true")
		} else {
			result = append(result, "false")
		}
	}
	if !reflect.DeepEqual(result, []string{"true", "true", "true", "false", "false", "false"}) {
		t.Fatal(`[Variance(5)] wrong idle value `, result)
	}
	trueCount := 0
	for _, v := range result {
		if v == "true" {
			trueCount++
		}
	}
	if trueCount != variance.IdlePeriod() {
		t.Fatalf("[Variance(4)] IdlePeriod() = %d, but IsIdle() was true %d times", variance.IdlePeriod(), trueCount)
	}
}

func TestVarianceCurrent(t *testing.T) {
	inputParsedData := []float64{1, 2, 7, 3, 12, 9}
	expectedParsedData := []float64{0.0, 0.0, 0.0, 0.0, 16.4, 13.84}
	indicator, _ := talive.NewVariance(5)
	for i := 0; i < 5; i++ {
		indicator.Next(&testCandle{close: inputParsedData[i]})
	}
	CurrentVal := roundFloat(indicator.Current(&testCandle{close: inputParsedData[5]})[0], 8)
	expectedValue := roundFloat(expectedParsedData[5], 8)
	if CurrentVal != expectedValue {
		t.Fatalf("[Variance(5)] wrong Current value %f, expected %f", CurrentVal, expectedValue)
	}
	NextVal := roundFloat(indicator.Next(&testCandle{close: inputParsedData[5]})[0], 8)
	if NextVal != CurrentVal {
		t.Fatalf("[Variance(5)] Current value call broke Next value %f, expected %f", NextVal, expectedValue)
	}
}
