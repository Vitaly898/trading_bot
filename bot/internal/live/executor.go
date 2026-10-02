package live

import (
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
	"time"
)

// Position — открытая позиция (единый вид для local и testnet).
type Position struct {
	Symbol     string    `json:"symbol"`
	Dir        int       `json:"dir"` // +1 long, -1 short
	Qty        float64   `json:"qty"`
	EntryPrice float64   `json:"entry_price"`
	Stop       float64   `json:"stop"`
	Take       float64   `json:"take"`
	EntryTime  time.Time `json:"entry_time"`
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
	log     *slog.Logger
	equity  float64
	feePct  float64
	slipPct float64
	positions map[string]*Position
}

func NewLocalExecutor(log *slog.Logger, startEquity, feePct, slipPct float64) *LocalExecutor {
	return &LocalExecutor{
		log: log, equity: startEquity, feePct: feePct, slipPct: slipPct,
		positions: map[string]*Position{},
	}
}

func (e *LocalExecutor) Name() string { return "local" }

func (e *LocalExecutor) Equity(context.Context) (float64, error) { return e.equity, nil }

func (e *LocalExecutor) Positions() map[string]*Position { return e.positions }

func (e *LocalExecutor) Open(_ context.Context, sym string, dir int, qty, stop, take, refPrice float64) (float64, error) {
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
	pos := e.positions[sym]
	if pos == nil {
		return 0, fmt.Errorf("нет позиции %s", sym)
	}
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
	log       *slog.Logger
	apiKey    string
	apiSecret string
	positions map[string]*Position
	steps     map[string]float64 // symbol → stepSize
	ticks     map[string]float64 // symbol → tickSize
}

func NewTestnetExecutor(log *slog.Logger, apiKey, apiSecret string) *TestnetExecutor {
	return &TestnetExecutor{
		log: log, apiKey: apiKey, apiSecret: apiSecret,
		positions: map[string]*Position{},
		steps:     map[string]float64{}, ticks: map[string]float64{},
	}
}

func (e *TestnetExecutor) Name() string { return "testnet" }

func (e *TestnetExecutor) Positions() map[string]*Position { return e.positions }

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
			SavedAt time.Time            `json:"saved_at"`
			Steps   map[string]float64   `json:"steps"`
			Ticks   map[string]float64   `json:"ticks"`
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
			DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		},
	}
	resp, err := client.Get(testnetURL + "/fapi/v1/exchangeInfo")
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
	v, _ := strconv.ParseFloat(acc.TotalWalletBalance, 64)
	return v, nil
}

func (e *TestnetExecutor) Open(ctx context.Context, sym string, dir int, qty, stop, take, refPrice float64) (float64, error) {
	side := "BUY"
	if dir < 0 {
		side = "SELL"
	}
	qtyStr := e.fmtQty(sym, qty)
	if qtyStr == "" {
		return 0, fmt.Errorf("%s: qty %.6f меньше stepSize", sym, qty)
	}
	resp, err := e.signedPost(ctx, "/fapi/v1/order", map[string]string{
		"symbol": sym, "side": side, "type": "MARKET", "quantity": qtyStr,
	}, nil)
	if err != nil {
		return 0, fmt.Errorf("entry order: %w", err)
	}
	fill := respAvgPrice(resp, refPrice)
	if fill <= 0 {
		// MARKET исполняется асинхронно — дочитываем ордер по id
		fill = e.fetchOrderPrice(ctx, sym, resp, refPrice)
	}

	// стоп и тейк — БИРЖЕВЫЕ algo-ордера (CONDITIONAL: STOP_MARKET / TAKE_PROFIT_MARKET).
	// С дек-2025 условные ордера обязаны идти через /fapi/v1/algoOrder (-4120 на старом).
	closeSide := "SELL"
	if dir < 0 {
		closeSide = "BUY"
	}
	placed := true
	if stop > 0 {
		if err := e.placeAlgo(ctx, sym, closeSide, "STOP_MARKET", stop); err != nil {
			placed = false
			e.log.Error("⚠️ стоп-ордер НЕ выставлен", "sym", sym, "err", err)
		}
	}
	if take > 0 {
		if err := e.placeAlgo(ctx, sym, closeSide, "TAKE_PROFIT_MARKET", take); err != nil {
			placed = false
			e.log.Error("⚠️ тейк-ордер НЕ выставлен", "sym", sym, "err", err)
		}
	}

	e.positions[sym] = &Position{
		Symbol: sym, Dir: dir, Qty: qty, EntryPrice: fill,
		Stop: stop, Take: take, EntryTime: time.Now().UTC(),
	}
	if !placed {
		// защита от голой позиции: SL/TP не встали — закрываем немедленно
		e.log.Error("❌ SL/TP не встали — закрываю позицию немедленно", "sym", sym)
		if _, cerr := e.signedPost(ctx, "/fapi/v1/order", map[string]string{
			"symbol": sym, "side": closeSide, "type": "MARKET",
			"quantity": qtyStr, "reduceOnly": "true",
		}, nil); cerr != nil {
			return fill, fmt.Errorf("naked position! close failed: %v (stop/take failed earlier)", cerr)
		}
		delete(e.positions, sym)
		return fill, fmt.Errorf("SL/TP не встали, позиция закрыта")
	}
	e.log.Info("🟢 ОТКРЫТИЕ (testnet)", "sym", sym, "dir", dir, "qty", qtyStr,
		"fill", round2(fill), "stop", round2(stop), "take", round2(take))
	return fill, nil
}

// placeAlgo — условный ордер через Algo API (CONDITIONAL).
func (e *TestnetExecutor) placeAlgo(ctx context.Context, symbol, side, algoType string, trigger float64) error {
	params := map[string]string{
		"symbol": symbol, "side": side, "type": algoType,
		"algoType": "CONDITIONAL", "triggerPrice": e.fmtPrice(symbol, trigger),
		"closePosition": "true", "workingType": "CONTRACT_PRICE",
	}
	resp, err := e.signedPost(ctx, "/fapi/v1/algoOrder", params, nil)
	if err != nil {
		return err
	}
	if code, ok := resp["code"].(float64); ok && code != 0 {
		return fmt.Errorf("algo order code %v: %v", code, resp["msg"])
	}
	e.log.Info("   📎 ордер выставлен", "sym", symbol, "type", algoType, "trigger", e.fmtPrice(symbol, trigger))
	return nil
}

// cancelAlgoOrders — снятие всех algo-ордеров по символу (best effort).
func (e *TestnetExecutor) cancelAlgoOrders(ctx context.Context, sym string) {
	params := map[string]string{"symbol": sym, "timestamp": strconv.FormatInt(time.Now().UnixMilli(), 10)}
	if err := e.signedDelete(ctx, "/fapi/v1/algoOpenOrders", params); err != nil {
		e.log.Warn("cancel algoOpenOrders", "sym", sym, "err", err)
	}
}

func (e *TestnetExecutor) Close(ctx context.Context, sym string, refPrice float64) (float64, error) {
	pos := e.positions[sym]
	if pos == nil {
		return 0, fmt.Errorf("нет позиции %s", sym)
	}
	// сначала снимаем SL/TP (обычные + algo), потом маркет-закрытие
	if err := e.signedDelete(ctx, "/fapi/v1/allOpenOrders", map[string]string{"symbol": sym}); err != nil {
		e.log.Warn("cancel allOpenOrders", "sym", sym, "err", err)
	}
	e.cancelAlgoOrders(ctx, sym)
	side := "SELL"
	if pos.Dir < 0 {
		side = "BUY"
	}
	resp, err := e.signedPost(ctx, "/fapi/v1/order", map[string]string{
		"symbol": sym, "side": side, "type": "MARKET",
		"quantity": e.fmtQty(sym, pos.Qty), "reduceOnly": "true",
	}, nil)
	if err != nil {
		return 0, fmt.Errorf("close order: %w", err)
	}
	fill := respAvgPrice(resp, refPrice)
	delete(e.positions, sym)
	e.log.Info("🔴 ЗАКРЫТИЕ (testnet)", "sym", sym, "fill", round2(fill))
	return fill, nil
}

// Reconcile — сверка локального кэша с биржей после рестарта.
// Запрос идёт по символам: полный positionRisk (700+ символов) режется VPN.
func (e *TestnetExecutor) Reconcile(ctx context.Context, symbols []string) error {
	seen := map[string]bool{}
	for _, sym := range symbols {
		var pr []struct {
			Symbol      string `json:"symbol"`
			PositionAmt string `json:"positionAmt"`
			EntryPrice  string `json:"entryPrice"`
		}
		var lastErr error
		for attempt := 0; attempt < 3; attempt++ {
			err := e.signedGet(ctx, "/fapi/v2/positionRisk", map[string]string{"symbol": sym}, &pr)
			if err == nil {
				lastErr = nil
				break
			}
			lastErr = err
			time.Sleep(2 * time.Second)
		}
		if lastErr != nil {
			return fmt.Errorf("reconcile %s: %w", sym, lastErr)
		}
		for _, p := range pr {
			seen[p.Symbol] = true
			amt, _ := strconv.ParseFloat(p.PositionAmt, 64)
			ep, _ := strconv.ParseFloat(p.EntryPrice, 64)
			if amt == 0 {
				delete(e.positions, p.Symbol)
				continue
			}
			dir := 1
			if amt < 0 {
				dir = -1
			}
			if e.positions[p.Symbol] == nil {
				e.log.Warn("позиция на бирже без локального состояния — подхватываю без SL/TP", "sym", p.Symbol)
			}
			e.positions[p.Symbol] = &Position{
				Symbol: p.Symbol, Dir: dir, Qty: math.Abs(amt), EntryPrice: ep,
				EntryTime: time.Now().UTC(),
			}
		}
	}
	// локальные позиции по символам, которых нет на бирже, — сбросить
	for sym := range e.positions {
		if !seen[sym] {
			delete(e.positions, sym)
		}
	}
	return nil
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
	resp, err := httpClient.Get(testnetURL + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, testnetURL+path+"?"+e.sign(params), nil)
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testnetURL+path+"?"+e.sign(params), nil)
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
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, testnetURL+path+"?"+e.sign(params), nil)
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
	if err := json.Unmarshal(body, &m); err != nil {
		return map[string]any{}, nil // DELETE и пустые ответы
	}
	return m, nil
}

func (e *TestnetExecutor) doRaw(req *http.Request) ([]byte, error) {
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (e *TestnetExecutor) fmtQty(sym string, qty float64) string {
	step := e.steps[sym]
	if step <= 0 {
		step = 0.001
	}
	qty = math.Floor(qty/step) * step
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
	return strconv.FormatFloat(price, 'f', decimals(tick), 64)
}

func decimals(step float64) int {
	d := 0
	for step < 1 {
		step *= 10
		d++
	}
	return d
}

func respAvgPrice(resp map[string]any, fallback float64) float64 {
	if s, ok := resp["avgPrice"].(string); ok {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
			return v
		}
	}
	return fallback
}

// fetchOrderPrice — цена исполнения MARKET-ордера через GET /fapi/v1/order.
func (e *TestnetExecutor) fetchOrderPrice(ctx context.Context, sym string, resp map[string]any, fallback float64) float64 {
	orderID, _ := resp["orderId"].(float64)
	if orderID == 0 {
		return fallback
	}
	for attempt := 0; attempt < 5; attempt++ {
		time.Sleep(400 * time.Millisecond)
		var ord struct {
			AvgPrice string `json:"avgPrice"`
		}
		err := e.signedGet(ctx, "/fapi/v1/order", map[string]string{
			"symbol": sym, "orderId": strconv.FormatInt(int64(orderID), 10),
		}, &ord)
		if err != nil {
			continue
		}
		if v, err := strconv.ParseFloat(ord.AvgPrice, 64); err == nil && v > 0 {
			return v
		}
	}
	return fallback
}

func round2(x float64) float64 { return math.Round(x*100) / 100 }
func round4(x float64) float64 { return math.Round(x*10000) / 10000 }
