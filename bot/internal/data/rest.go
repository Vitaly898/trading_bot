package data

import (
	"encoding/json"
	"fmt"
	"time"
)

const restBase = "https://fapi.binance.com"

// FetchKlinesREST — история свечей через REST API Binance Futures
// с пагинацией по 1500 свечей. [startMs, endMs) — unix ms UTC.
func FetchKlinesREST(symbol, interval string, startMs, endMs int64) ([]Kline, error) {
	var out []Kline
	cursor := startMs
	for cursor < endMs {
		url := fmt.Sprintf("%s/fapi/v1/klines?symbol=%s&interval=%s&startTime=%d&endTime=%d&limit=1500",
			restBase, symbol, interval, cursor, endMs-1)
		resp, err := httpClient.Get(url)
		if err != nil {
			return out, err
		}
		var raw [][]any
		err = json.NewDecoder(resp.Body).Decode(&raw)
		resp.Body.Close()
		if err != nil {
			return out, fmt.Errorf("klines decode: %w", err)
		}
		if len(raw) == 0 {
			break
		}
		for _, rec := range raw {
			if len(rec) < 6 {
				continue
			}
			tsMs, ok := rec[0].(float64)
			if !ok {
				continue
			}
			out = append(out, Kline{
				OpenTime: time.UnixMilli(int64(tsMs)).UTC(),
				Open:     parseFloat(str(rec[1])),
				High:     parseFloat(str(rec[2])),
				Low:      parseFloat(str(rec[3])),
				Close:    parseFloat(str(rec[4])),
				Volume:   parseFloat(str(rec[5])),
			})
		}
		lastTs := int64(raw[len(raw)-1][0].(float64))
		if lastTs <= cursor { // защита от зацикливания
			break
		}
		cursor = lastTs + 1
		time.Sleep(150 * time.Millisecond) // бережём rate limit (weight 10/запрос)
	}
	return out, nil
}

// FetchFundingREST — история funding rate через REST API. [startMs, endMs) — unix ms UTC.
func FetchFundingREST(symbol string, startMs, endMs int64) ([]Funding, error) {
	var out []Funding
	cursor := startMs
	for cursor < endMs {
		url := fmt.Sprintf("%s/fapi/v1/fundingRate?symbol=%s&startTime=%d&endTime=%d&limit=1000",
			restBase, symbol, cursor, endMs-1)
		resp, err := httpClient.Get(url)
		if err != nil {
			return out, err
		}
		var raw []struct {
			FundingTime int64  `json:"fundingTime"`
			FundingRate string `json:"fundingRate"`
		}
		err = json.NewDecoder(resp.Body).Decode(&raw)
		resp.Body.Close()
		if err != nil {
			return out, fmt.Errorf("funding decode: %w", err)
		}
		if len(raw) == 0 {
			break
		}
		for _, rec := range raw {
			out = append(out, Funding{
				CalcTime: time.UnixMilli(rec.FundingTime).UTC(),
				Rate:     parseFloat(rec.FundingRate),
			})
		}
		lastTs := raw[len(raw)-1].FundingTime
		if lastTs <= cursor {
			break
		}
		cursor = lastTs + 1
		time.Sleep(150 * time.Millisecond)
	}
	return out, nil
}

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
