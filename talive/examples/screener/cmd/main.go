package main

import (
	"context"
	"os"
	"os/signal"
	"screener/internal"
	"screener/internal/binance"
	"sync"
	"syscall"
	"time"
)

func main() {
	log := internal.NewLogger()
	symbolsCtx, symbolsCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer symbolsCancel()
	httpClient := binance.NewHTTPClient(log)
	symbols, err := httpClient.TopVolumeSymbols(symbolsCtx, 1)
	if err != nil {
		log.Error("Error calling TopVolumeSymbols", "error", err)
		return
	}
	log.Info("Top Volume Symbols", "symbols", symbols)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	workerCtx, workerCancel := context.WithCancel(context.Background())
	ws := binance.NewWsClient(log)
	err = ws.Connect(errChan)
	if err != nil {
		log.Error("Error connection to WS", "error", err)
		return
	}
	workerMap := make(map[string]chan<- binance.Kline)
	var workerWg sync.WaitGroup
	for _, symbol := range symbols {
		klineCh := make(chan binance.Kline, 10)
		w := internal.NewScreenerWorker(workerCtx, log, symbol, httpClient)
		workerWg.Add(1)
		w.Start(klineCh, &workerWg)
		workerMap[symbol] = klineCh
	}

	ws.SetKlineHandler(func(kline binance.Kline) {
		klineCh, ok := workerMap[kline.Symbol]
		if !ok {
			log.Warn("Worker not found in workerMap", "symbol", kline.Symbol)
			return
		}
		klineCh <- kline
	})

	err = ws.SubscribeSymbols(symbols)

	if err != nil {
		workerCancel()
		log.Error("Error subscribing symbols", "error", err)
		return
	}

	select {
	case <-sigChan:
		workerCancel()
		log.Info("Received shutdown signal")
	case err = <-errChan:
		workerCancel()
		log.Error("Shutting down due to error", "error", err)
	}

	workerWg.Wait()
	err = ws.Close()
	if err != nil {
		log.Error("Error closing WS", "error", err)
	}
}
