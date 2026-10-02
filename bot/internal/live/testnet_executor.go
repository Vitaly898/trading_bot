package live

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

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
