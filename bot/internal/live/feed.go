// Package live — live-данные с боевого Binance Futures: WS-стрим свечей + прогрев REST.
package live

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bot/internal/candle"
	"bot/internal/market"

	"github.com/gorilla/websocket"
)

const (
	wsURL   = "wss://fstream.binance.com/stream"
	restURL = "https://fapi.binance.com"
)

// Свежий TCP на каждый запрос: через нестабильные туннели (VPN/ISP-прокси)
// закэшированное keep-alive соединение глохнет, и все последующие запросы
// висят на трупе. Наш трафик — единицы запросов в минуту, keep-alive не нужен.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		DisableKeepAlives: true,
		DialContext:       (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
	},
}

// KlineEvent — свеча из потока (IsClosed=true на закрытии).
type KlineEvent struct {
	Symbol   string
	Candle   candle.Candle
	IsClosed bool
}

// Warmup — последние N закрытых свечей символа (прогрев индикаторов).
func Warmup(ctx context.Context, symbol, interval string, limit int) ([]candle.Candle, error) {
	url := fmt.Sprintf("%s/fapi/v1/klines?symbol=%s&interval=%s&limit=%d",
		restURL, strings.ToUpper(symbol), interval, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("warmup HTTP %d", resp.StatusCode)
	}
	var raw [][]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	var out []candle.Candle
	for _, rec := range raw {
		if len(rec) < 6 {
			continue
		}
		tsMs, _ := rec[0].(float64)
		out = append(out, candle.Candle{
			Time: time.UnixMilli(int64(tsMs)).UTC(),
			O:    f(rec[1]), H: f(rec[2]), L: f(rec[3]), C: f(rec[4]), V: f(rec[5]),
		})
	}
	// последняя свеча может быть незакрытой — отбрасываем, если её период ещё идёт
	if len(out) > 0 {
		last := out[len(out)-1]
		if time.Since(last.Time) < tfDur(interval) {
			out = out[:len(out)-1]
		}
	}
	return out, nil
}

func f(v any) float64 {
	s, _ := v.(string)
	fv, _ := strconv.ParseFloat(s, 64)
	return fv
}

func tfDur(interval string) time.Duration {
	duration, _ := market.Interval(interval) // caller validates config before starting feeds
	return duration
}

// StreamKlines — подписка на kline_<interval> по символам, шлёт в out.
// Реконнект с backoff. Блокируется до ctx.Done().
func StreamKlines(ctx context.Context, log *slog.Logger, symbols []string, interval string, out chan<- KlineEvent) {
	backoff := time.Second
	for {
		err := streamOnce(ctx, log, symbols, interval, out)
		if ctx.Err() != nil {
			return
		}
		log.Warn("WS переподключение", "ошибка", err, "backoff", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

// ReconcileLoop — gap-recovery: раз в interval сверяет с REST последние закрытые
// свечи и доигрывает их в out (на случай обрывов WS в момент закрытия).
// Дедупликация — на стороне Runner (lastClosed).
func ReconcileLoop(ctx context.Context, log *slog.Logger, symbols []string, interval string, out chan<- KlineEvent, every time.Duration) {
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			for _, sym := range symbols {
				var candles []candle.Candle
				var err error
				// ретраи: VPN/сеть может обрывать и REST; 3 попытки с паузой
				for attempt := 0; attempt < 3; attempt++ {
					candles, err = Warmup(ctx, sym, interval, 3)
					if err == nil {
						break
					}
					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Duration(attempt+1) * 2 * time.Second):
					}
				}
				if err != nil {
					log.Warn("reconcile: ошибка (3 попытки)", "sym", sym, "err", err)
					continue
				}
				for _, c := range candles {
					ev := KlineEvent{Symbol: sym, Candle: c, IsClosed: true}
					select {
					case out <- ev:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}
}

type wsStreamMsg struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

type wsKlineMsg struct {
	Symbol string `json:"s"`
	Kline  struct {
		StartTime int64  `json:"t"`
		Interval  string `json:"i"`
		Open      string `json:"o"`
		High      string `json:"h"`
		Low       string `json:"l"`
		Close     string `json:"c"`
		Volume    string `json:"v"`
		IsClosed  bool   `json:"x"`
	} `json:"k"`
}

func streamOnce(ctx context.Context, log *slog.Logger, symbols []string, interval string, out chan<- KlineEvent) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	params := make([]string, 0, len(symbols))
	for _, s := range symbols {
		params = append(params, fmt.Sprintf("%s@kline_%s", strings.ToLower(s), interval))
	}
	sub := map[string]any{"method": "SUBSCRIBE", "params": params, "id": 1}
	if err := conn.WriteJSON(sub); err != nil {
		return err
	}
	log.Info("WS подписка активна", "streams", params)

	// keepalive: клиентский ping раз в 60с (защита от NAT/firewall idle-таймаутов)
	pingDone := make(chan struct{})
	go func() {
		tick := time.NewTicker(60 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-pingDone:
				return
			case <-ctx.Done():
				return
			case <-tick.C:
				_ = conn.WriteMessage(websocket.PingMessage, nil)
			}
		}
	}()
	defer close(pingDone)

	// контекстная отмена чтения
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var sm wsStreamMsg
		if json.Unmarshal(msg, &sm) != nil || sm.Stream == "" {
			continue
		}
		var km wsKlineMsg
		if err := json.Unmarshal(sm.Data, &km); err != nil {
			continue
		}
		pf := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }
		select {
		case out <- KlineEvent{
			Symbol: km.Symbol,
			Candle: candle.Candle{
				Time: time.UnixMilli(km.Kline.StartTime).UTC(),
				O:    pf(km.Kline.Open), H: pf(km.Kline.High),
				L: pf(km.Kline.Low), C: pf(km.Kline.Close), V: pf(km.Kline.Volume),
			},
			IsClosed: km.Kline.IsClosed,
		}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
