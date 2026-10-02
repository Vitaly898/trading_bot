package live

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

const filtersCachePath = "paper_exchange_info.json"

// loadFilters — кэш → fetch с ретраями → кэш.
func (e *TestnetExecutor) loadFilters(ctx context.Context) error {
	// 1. свежий кэш?
	if data, err := os.ReadFile(filtersCachePath); err == nil {
		var cached struct {
			SavedAt time.Time          `json:"saved_at"`
			Steps   map[string]float64 `json:"steps"`
			Ticks   map[string]float64 `json:"ticks"`
		}
		if json.Unmarshal(data, &cached) == nil && time.Since(cached.SavedAt) < 7*24*time.Hour {
			e.steps, e.ticks = cached.Steps, cached.Ticks
			e.log.Info("фильтры из кэша", "symbols", len(e.steps), "age", time.Since(cached.SavedAt).Round(time.Hour))
			return nil
		}
	}
	// 2. fetch с ретраями
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		steps, ticks, err := e.fetchFilters(ctx)
		if err == nil {
			e.steps, e.ticks = steps, ticks
			data, _ := json.MarshalIndent(struct {
				SavedAt time.Time          `json:"saved_at"`
				Steps   map[string]float64 `json:"steps"`
				Ticks   map[string]float64 `json:"ticks"`
			}{time.Now().UTC(), steps, ticks}, "", "  ")
			_ = os.WriteFile(filtersCachePath, data, 0o644)
			e.log.Info("фильтры получены и закэшированы", "symbols", len(steps))
			return nil
		}
		lastErr = err
		e.log.Warn("fetch filters", "attempt", attempt+1, "err", err)
		time.Sleep(3 * time.Second)
	}
	// 3. протухший кэш лучше ничего
	if data, err := os.ReadFile(filtersCachePath); err == nil {
		var cached struct {
			Steps map[string]float64 `json:"steps"`
			Ticks map[string]float64 `json:"ticks"`
		}
		if json.Unmarshal(data, &cached) == nil {
			e.steps, e.ticks = cached.Steps, cached.Ticks
			e.log.Warn("использую ПРОТУХШИЙ кэш фильтров")
			return nil
		}
	}
	return lastErr
}

func (e *TestnetExecutor) fetchFilters(ctx context.Context) (map[string]float64, map[string]float64, error) {
	client := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			DialContext:       (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		},
	}
	resp, err := client.Get(e.baseURL + "/fapi/v1/exchangeInfo")
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var info struct {
		Symbols []struct {
			Symbol  string `json:"symbol"`
			Filters []struct {
				FilterType string `json:"filterType"`
				StepSize   string `json:"stepSize"`
				TickSize   string `json:"tickSize"`
			} `json:"filters"`
		} `json:"symbols"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, nil, err
	}
	steps, ticks := map[string]float64{}, map[string]float64{}
	for _, s := range info.Symbols {
		for _, fl := range s.Filters {
			switch fl.FilterType {
			case "LOT_SIZE":
				steps[s.Symbol], _ = strconv.ParseFloat(fl.StepSize, 64)
			case "PRICE_FILTER":
				ticks[s.Symbol], _ = strconv.ParseFloat(fl.TickSize, 64)
			}
		}
	}
	return steps, ticks, nil
}

// fallbackFilters — последний рубеж: типичные точности для известных символов.
func (e *TestnetExecutor) fallbackFilters(symbols []string) {
	table := map[string][2]float64{
		// stepSize, tickSize (типичные значения Binance Futures)
		"BTCUSDT": {0.001, 0.1}, "ETHUSDT": {0.01, 0.01}, "SOLUSDT": {1, 0.01},
		"XRPUSDT": {1, 0.0001}, "BNBUSDT": {0.01, 0.01}, "DOGEUSDT": {1, 0.00001},
		"ADAUSDT": {1, 0.0001}, "TRXUSDT": {1, 0.00001}, "AVAXUSDT": {0.1, 0.01},
		"LINKUSDT": {0.1, 0.001}, "XLMUSDT": {1, 0.0001}, "HBARUSDT": {1, 0.00001},
		"LTCUSDT": {0.1, 0.01}, "BCHUSDT": {0.01, 0.1}, "DOTUSDT": {0.1, 0.001},
		"UNIUSDT": {0.1, 0.001}, "NEARUSDT": {0.1, 0.001}, "AAVEUSDT": {0.1, 0.01},
		"ATOMUSDT": {0.1, 0.001}, "1000PEPEUSDT": {1, 0.000001},
	}
	for _, sym := range symbols {
		if v, ok := table[sym]; ok {
			e.steps[sym], e.ticks[sym] = v[0], v[1]
		} else {
			e.steps[sym], e.ticks[sym] = 0.001, 0.01
		}
	}
}

func (e *TestnetExecutor) fmtQty(sym string, qty float64) string {
	step := e.steps[sym]
	if step <= 0 {
		step = 0.001
	}
	qty = math.Floor(qty/step+1e-9) * step
	if qty <= 0 {
		return ""
	}
	return strconv.FormatFloat(qty, 'f', decimals(step), 64)
}

func (e *TestnetExecutor) fmtPrice(sym string, price float64) string {
	tick := e.ticks[sym]
	if tick <= 0 {
		tick = 0.01
	}
	return strconv.FormatFloat(math.Round(price/tick)*tick, 'f', decimals(tick), 64)
}

func decimals(step float64) int {
	for d := 0; d < 12; d++ {
		nearest := math.Round(step)
		if nearest >= 1 && math.Abs(step-nearest) <= 1e-9*math.Max(1, math.Abs(step)) {
			return d
		}
		step *= 10
	}
	return 12
}
