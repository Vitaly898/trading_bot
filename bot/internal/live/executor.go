package live

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Position — открытая позиция (единый вид для local и testnet).
type Position struct {
	Symbol        string    `json:"symbol"`
	Dir           int       `json:"dir"` // +1 long, -1 short
	Qty           float64   `json:"qty"`
	EntryPrice    float64   `json:"entry_price"`
	Stop          float64   `json:"stop"`
	Take          float64   `json:"take"`
	EntryTime     time.Time `json:"entry_time"`
	EntryOrderID  int64     `json:"entry_order_id,omitempty"`
	ClientOrderID string    `json:"client_order_id,omitempty"`
	PendingClose  string    `json:"pending_close,omitempty"`
	StopOrderID   int64     `json:"stop_order_id,omitempty"`
	TakeOrderID   int64     `json:"take_order_id,omitempty"`
	Unconfirmed   bool      `json:"unconfirmed,omitempty"`
}

// Executor — исполнение ордеров.
type Executor interface {
	// Open — вход маркетом + стоп/тейк. Возвращает фактическую цену входа.
	Open(ctx context.Context, sym string, dir int, qty, stop, take, refPrice float64) (float64, error)
	// Close — закрытие позиции по символу маркетом (+ отмена SL/TP ордеров).
	Close(ctx context.Context, sym string, refPrice float64) (float64, error)
	// Equity — текущий капитал в USDT.
	Equity(ctx context.Context) (float64, error)
	// Positions — локальный кэш открытых позиций.
	Positions() map[string]*Position
	// Name — режим.
	Name() string
}

// ---------- Local: виртуальное исполнение по реальным ценам ----------

type LocalExecutor struct {
	mu        sync.RWMutex
	log       *slog.Logger
	equity    float64
	feePct    float64
	slipPct   float64
	positions map[string]*Position
}

func NewLocalExecutor(log *slog.Logger, startEquity, feePct, slipPct float64) *LocalExecutor {
	return &LocalExecutor{
		log: log, equity: startEquity, feePct: feePct, slipPct: slipPct,
		positions: map[string]*Position{},
	}
}

func (e *LocalExecutor) Name() string { return "local" }

func (e *LocalExecutor) Equity(context.Context) (float64, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.equity, nil
}

func (e *LocalExecutor) Positions() map[string]*Position {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return copyPositions(e.positions)
}

func (e *LocalExecutor) Open(_ context.Context, sym string, dir int, qty, stop, take, refPrice float64) (float64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.positions[sym] != nil {
		return 0, fmt.Errorf("position already exists: %s", sym)
	}
	if err := validateEntry(dir, qty, refPrice); err != nil {
		return 0, err
	}
	fill := refPrice * (1 + float64(dir)*e.slipPct)
	fee := fill * qty * e.feePct
	e.equity -= fee
	e.positions[sym] = &Position{
		Symbol: sym, Dir: dir, Qty: qty, EntryPrice: fill,
		Stop: stop, Take: take, EntryTime: time.Now().UTC(),
	}
	e.log.Info("🟢 ОТКРЫТИЕ", "sym", sym, "dir", dir, "qty", round4(qty),
		"fill", round2(fill), "stop", round2(stop), "take", round2(take), "fee", round2(fee))
	return fill, nil
}

func (e *LocalExecutor) Close(_ context.Context, sym string, refPrice float64) (float64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	pos := e.positions[sym]
	if pos == nil {
		return 0, fmt.Errorf("нет позиции %s", sym)
	}
	fill := refPrice * (1 - float64(pos.Dir)*e.slipPct)
	fee := fill * pos.Qty * e.feePct
	pnl := float64(pos.Dir)*(fill-pos.EntryPrice)*pos.Qty - fee
	e.equity += pnl
	delete(e.positions, sym)
	e.log.Info("🔴 ЗАКРЫТИЕ", "sym", sym, "fill", round2(fill),
		"pnl", round2(pnl), "equity", round2(e.equity))
	return fill, nil
}

// CloseAt — закрытие по конкретной цене (стоп/тейк в local-режиме).
func (e *LocalExecutor) CloseAt(sym string, price float64, reason string) (float64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	pos := e.positions[sym]
	if pos == nil {
		return 0, fmt.Errorf("нет позиции %s", sym)
	}
	price *= 1 - float64(pos.Dir)*e.slipPct
	fee := price * pos.Qty * e.feePct
	pnl := float64(pos.Dir)*(price-pos.EntryPrice)*pos.Qty - fee
	e.equity += pnl
	delete(e.positions, sym)
	e.log.Info(fmt.Sprintf("🔴 ЗАКРЫТИЕ (%s)", reason), "sym", sym,
		"fill", round2(price), "pnl", round2(pnl), "equity", round2(e.equity))
	return price, nil
}

// ---------- Testnet: реальные ордера на Binance Futures Testnet ----------

const testnetURL = "https://testnet.binancefuture.com"

type TestnetExecutor struct {
	beforeSubmit func(map[string]*Position) error
	mu           sync.RWMutex
	client       *http.Client
	baseURL      string
	pollInterval time.Duration
	log          *slog.Logger
	apiKey       string
	apiSecret    string
	positions    map[string]*Position
	steps        map[string]float64 // symbol → stepSize
	ticks        map[string]float64 // symbol → tickSize
}

func NewTestnetExecutor(log *slog.Logger, apiKey, apiSecret string) *TestnetExecutor {
	return &TestnetExecutor{
		log: log, apiKey: apiKey, apiSecret: apiSecret,
		client: httpClient, baseURL: testnetURL, pollInterval: 400 * time.Millisecond,
		positions: map[string]*Position{},
		steps:     map[string]float64{}, ticks: map[string]float64{},
	}
}

func (e *TestnetExecutor) Name() string { return "testnet" }

func (e *TestnetExecutor) Positions() map[string]*Position {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return copyPositions(e.positions)
}

// Init — фильтры биржи (stepSize/tickSize) + плечо по символам.
// Фильтры кэшируются в paper_exchange_info.json: VPN периодически душит
// большие ответы, а точность символов меняется редко.
func (e *TestnetExecutor) Init(ctx context.Context, symbols []string, leverage int) error {
	if err := e.loadFilters(ctx); err != nil {
		e.log.Warn("фильтры из exchangeInfo не получены — использую fallback-таблицу", "err", err)
		e.fallbackFilters(symbols)
	}
	for _, sym := range symbols {
		if leverage <= 0 {
			leverage = 1
		}
		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			_, err := e.signedPost(ctx, "/fapi/v1/leverage", map[string]string{
				"symbol": sym, "leverage": strconv.Itoa(leverage),
			}, nil)
			if err == nil {
				lastErr = nil
				break
			}
			lastErr = err
			time.Sleep(2 * time.Second)
		}
		if lastErr != nil {
			e.log.Warn("leverage set", "sym", sym, "err", lastErr)
		}
	}
	return nil
}

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

func (e *TestnetExecutor) Equity(ctx context.Context) (float64, error) {
	var acc struct {
		TotalWalletBalance string `json:"totalWalletBalance"`
	}
	if err := e.signedGet(ctx, "/fapi/v2/account", nil, &acc); err != nil {
		return 0, err
	}
	v, err := strconv.ParseFloat(acc.TotalWalletBalance, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, fmt.Errorf("invalid wallet balance")
	}
	return v, nil
}

// LastPrice — последняя цена (публичный эндпоинт).
func (e *TestnetExecutor) LastPrice(ctx context.Context, sym string) (float64, error) {
	var t struct {
		Price string `json:"price"`
	}
	if err := e.publicGet(ctx, "/fapi/v1/ticker/price?symbol="+sym, &t); err != nil {
		return 0, err
	}
	v, err := strconv.ParseFloat(t.Price, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// --- HTTP helpers ---

func (e *TestnetExecutor) publicGet(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (e *TestnetExecutor) sign(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	q := strings.Join(parts, "&")
	mac := hmac.New(sha256.New, []byte(e.apiSecret))
	mac.Write([]byte(q))
	return q + "&signature=" + hex.EncodeToString(mac.Sum(nil))
}

func (e *TestnetExecutor) signedPost(ctx context.Context, path string, params map[string]string, _ map[string]string) (map[string]any, error) {
	if params == nil {
		params = map[string]string{}
	}
	params["timestamp"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+path+"?"+e.sign(params), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-MBX-APIKEY", e.apiKey)
	return e.do(req)
}

func (e *TestnetExecutor) signedGet(ctx context.Context, path string, params map[string]string, out any) error {
	if params == nil {
		params = map[string]string{}
	}
	params["timestamp"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.baseURL+path+"?"+e.sign(params), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-MBX-APIKEY", e.apiKey)
	resp, err := e.doRaw(req)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp, out)
}

func (e *TestnetExecutor) signedDelete(ctx context.Context, path string, params map[string]string) error {
	params["timestamp"] = strconv.FormatInt(time.Now().UnixMilli(), 10)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, e.baseURL+path+"?"+e.sign(params), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-MBX-APIKEY", e.apiKey)
	_, err = e.doRaw(req)
	return err
}

func (e *TestnetExecutor) do(req *http.Request) (map[string]any, error) {
	body, err := e.doRaw(req)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

func (e *TestnetExecutor) doRaw(req *http.Request) ([]byte, error) {
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		var apiErr exchangeError
		_ = json.Unmarshal(body, &apiErr)
		apiErr.HTTPStatus = resp.StatusCode
		return nil, &apiErr
	}
	return body, nil
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

func round2(x float64) float64 { return math.Round(x*100) / 100 }
func round4(x float64) float64 { return math.Round(x*10000) / 10000 }
