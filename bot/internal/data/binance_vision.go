// Package data — загрузка публичных исторических данных Binance Futures (UM)
// с data.binance.vision: месячные CSV-архивы свечей и funding rate.
package data

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://data.binance.vision/data/futures/um/monthly"

// Kline — одна свеча OHLCV.
type Kline struct {
	OpenTime time.Time
	Open     float64
	High     float64
	Low      float64
	Close    float64
	Volume   float64
}

// Funding — одна запись funding rate.
type Funding struct {
	CalcTime time.Time
	Rate     float64
}

var httpClient = &http.Client{Timeout: 60 * time.Second}

// Month — год-месяц для адресации архивов.
type Month struct {
	Year  int
	Month time.Month
}

// MonthsBetween — все месяцы [from, to] включительно.
func MonthsBetween(from, to Month) []Month {
	var res []Month
	cur := time.Date(from.Year, from.Month, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(to.Year, to.Month, 1, 0, 0, 0, 0, time.UTC)
	for !cur.After(end) {
		res = append(res, Month{cur.Year(), cur.Month()})
		cur = cur.AddDate(0, 1, 0)
	}
	return res
}

func fetchZipCSV(url string) (*csv.Reader, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("bad zip from %s: %w", url, err)
	}
	if len(zr.File) == 0 {
		return nil, fmt.Errorf("empty zip from %s", url)
	}
	f, err := zr.File[0].Open()
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		return nil, err
	}
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1 // строки могут отличаться по ширине
	return r, nil
}

// parseTs — open_time/calc_time могут быть в миллисекундах или микросекундах
// (Binance менял формат в 2025). Определяем по порядку величины.
func parseTs(s string) (time.Time, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	if v > 1e15 { // микросекунды
		return time.UnixMicro(v).UTC(), nil
	}
	return time.UnixMilli(v).UTC(), nil
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

// FetchKlines — скачивает месячный архив свечей символа в заданном ТФ.
// interval: "15m", "1h", "4h", "1d" и т.д.
func FetchKlines(symbol, interval string, m Month) ([]Kline, error) {
	url := fmt.Sprintf("%s/klines/%s/%s/%s-%s-%d-%02d.zip",
		baseURL, strings.ToUpper(symbol), interval,
		strings.ToUpper(symbol), interval, m.Year, m.Month)
	r, err := fetchZipCSV(url)
	if err != nil {
		return nil, err
	}
	var out []Kline
	first := true
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if first { // пропускаем заголовок, если он текстовый
			first = false
			if _, perr := strconv.ParseInt(rec[0], 10, 64); perr != nil {
				continue
			}
		}
		if len(rec) < 6 {
			continue
		}
		ts, err := parseTs(rec[0])
		if err != nil {
			continue
		}
		out = append(out, Kline{
			OpenTime: ts,
			Open:     parseFloat(rec[1]),
			High:     parseFloat(rec[2]),
			Low:      parseFloat(rec[3]),
			Close:    parseFloat(rec[4]),
			Volume:   parseFloat(rec[5]),
		})
	}
	return out, nil
}

// FetchFunding — скачивает месячный архив funding rate символа.
func FetchFunding(symbol string, m Month) ([]Funding, error) {
	url := fmt.Sprintf("%s/fundingRate/%s/%s-fundingRate-%d-%02d.zip",
		baseURL, strings.ToUpper(symbol), strings.ToUpper(symbol), m.Year, m.Month)
	r, err := fetchZipCSV(url)
	if err != nil {
		return nil, err
	}
	var out []Funding
	first := true
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if first {
			first = false
			if _, perr := strconv.ParseInt(rec[0], 10, 64); perr != nil {
				continue
			}
		}
		if len(rec) < 3 {
			continue
		}
		ts, err := parseTs(rec[0])
		if err != nil {
			continue
		}
		out = append(out, Funding{CalcTime: ts, Rate: parseFloat(rec[2])})
	}
	return out, nil
}
