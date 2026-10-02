package talive_test

import (
	"encoding/csv"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/MoroZvlg/talive"
)

var floatDummy float64
var sliceDummy []float64
var benchSink any

func roundFloat(number float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(number*pow) / pow
}

// primeForCurrentBench feeds IdlePeriod()+1 candles via Next so the indicator
// exits the idle short-circuit before benchmarking Current. Returns the index
// of the last fed candle so limitedDataIndex advances to the next fresh one.
func primeForCurrentBench(ind talive.Indicator, candles []*testCandle) int {
	idle := ind.IdlePeriod()
	for i := 0; i <= idle; i++ {
		ind.Next(candles[i])
	}
	return idle
}

func limitedDataIndex(dataIndex int, dataLen int) int {
	if dataIndex >= dataLen-1 {
		return 0
	}
	return dataIndex + 1
}

func difference(leftSlice, rightSlice []float64) map[int][]float64 {
	result := make(map[int][]float64, len(leftSlice))
	if len(leftSlice) != len(rightSlice) {
		result[-1] = []float64{float64(len(leftSlice)), float64(len(leftSlice))}
		return result
	}
	for i, leftElem := range leftSlice {
		rightElem := rightSlice[i]
		if leftElem != rightElem {
			result[i] = []float64{leftElem, rightElem}
		}
	}
	return result
}

func readCandles(filePath string) ([]*testCandle, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	csvReader := csv.NewReader(file)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}
	result := make([]*testCandle, 0, len(records)-1)
	for i, record := range records {
		if i == 0 {
			continue
		}
		ts, _ := strconv.ParseInt(record[0], 10, 64)
		open, _ := strconv.ParseFloat(record[1], 64)
		high, _ := strconv.ParseFloat(record[2], 64)
		low, _ := strconv.ParseFloat(record[3], 64)
		closeV, _ := strconv.ParseFloat(record[4], 64)
		volume, _ := strconv.ParseFloat(record[5], 64)
		result = append(result, &testCandle{
			open: open, high: high, low: low, close: closeV, volume: volume,
			timestamp: time.Unix(ts, 0).UTC(),
		})
	}
	return result, nil
}

func readData(filePath string, columnsIdx []int, roundDecimals int) ([][]float64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	csvReader := csv.NewReader(file)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}
	result := make([][]float64, len(records)-1) // -headers
	for recordIdx, record := range records {
		if recordIdx == 0 {
			continue // skip header
		}
		for i, columnIdx := range columnsIdx {
			value := record[columnIdx]
			floatValue := 0.0
			if value == "NaN" {
				result[i] = append(result[i], floatValue)
				continue
			}
			floatValue, _ = strconv.ParseFloat(value, 64)
			result[i] = append(result[i], roundFloat(floatValue, roundDecimals))
		}
	}
	return result, nil
}
