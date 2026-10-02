package internal

import (
	"context"
	"log/slog"
	"screener/internal/binance"
	"sync"
	"sync/atomic"
	"time"
)

type ScreenerWorker struct {
	log               *slog.Logger
	ctx               context.Context
	ready             atomic.Bool
	Symbol            string
	Screener          *Screener
	httpClient        *binance.HTTPClient
	wsLoadingBuffer   []binance.Kline
	lastProcessedTime time.Time
}

func NewScreenerWorker(
	ctx context.Context,
	log *slog.Logger,
	symbol string,
	client *binance.HTTPClient,
) *ScreenerWorker {
	return &ScreenerWorker{
		log:             log.With("symbol", symbol),
		ctx:             ctx,
		Symbol:          symbol,
		Screener:        NewScreener(log),
		httpClient:      client,
		wsLoadingBuffer: make([]binance.Kline, 0),
	}
}

func (w *ScreenerWorker) Start(klineCh <-chan binance.Kline, wg *sync.WaitGroup) {
	go w.fetchKlinesHistory()
	go func() {
		defer wg.Done()
		for {
			select {
			case <-w.ctx.Done():
				return
			case kline, ok := <-klineCh:
				if !ok {
					return
				}
				if w.ready.Load() {
					for _, bufKline := range w.wsLoadingBuffer {
						if bufKline.TimeStart.After(w.lastProcessedTime) {
							w.processKline(bufKline)
						}
					}
					w.wsLoadingBuffer = nil
					w.processKline(kline)
				} else {
					w.wsLoadingBuffer = append(w.wsLoadingBuffer, kline)
				}
			}
		}
	}()
}

func (w *ScreenerWorker) fetchKlinesHistory() {
	klines, err := w.httpClient.LastKlines(w.ctx, w.Symbol, w.Screener.MaxWarmUp()+1)
	if err != nil {
		w.log.Error("Error fetching LastKlines", "error", err)
		// NOTE: it's ok for example to go without history. Do not return
	}
	for _, kline := range klines {
		w.processKline(kline)
	}
	w.ready.Store(true)
}

func (w *ScreenerWorker) processKline(kline binance.Kline) {
	if kline.IsClosed {
		result := w.Screener.Next(&kline)
		w.log.Info("Screener result", "result", result, "receive->processed", time.Since(kline.TimeReceived))
	} else {
		result := w.Screener.Current(&kline)
		w.log.Debug("Screener result", "result", result, "receive->processed", time.Since(kline.TimeReceived))
	}
}
