// paper — paper trading: реальные данные Binance, исполнение local или testnet.
//
//	local:   go run ./cmd/paper -config configs/eth_sol_regime.yaml
//	testnet: BINANCE_TESTNET_KEY=... BINANCE_TESTNET_SECRET=... \
//	         go run ./cmd/paper -config configs/eth_sol_regime.yaml -exec testnet
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"bot/internal/config"
	"bot/internal/live"
	"bot/internal/strategy"
)

func main() {
	configPath := flag.String("config", "", "путь к YAML-конфигу")
	execMode := flag.String("exec", "local", "local | testnet")
	warmupN := flag.Int("warmup", 300, "свечей прогрева")
	statePath := flag.String("state", "paper_state.json", "путь к снимку состояния")
	testOrder := flag.String("testorder", "", "тест исполнения: SYMBOL:QTY (напр. ETHUSDT:0.005) — открыть и закрыть, выйти")
	flag.Parse()
	if *execMode != "local" && *execMode != "testnet" {
		log.Fatal("exec: local | testnet")
	}
	if *warmupN <= 0 || *warmupN > 1500 {
		log.Fatal("warmup: 1..1500")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if *configPath == "" {
		log.Fatal("укажи -config")
	}
	cc, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	syms := cc.Symbols
	if len(syms) == 0 {
		syms = []string{cc.Symbol}
	}
	for i := range syms {
		syms[i] = strings.ToUpper(syms[i])
	}

	level := slog.LevelInfo
	if strings.EqualFold(os.Getenv("LOG_LEVEL"), "debug") {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	// исполнитель
	var exec live.Executor
	if *execMode == "testnet" {
		key, secret := os.Getenv("BINANCE_TESTNET_KEY"), os.Getenv("BINANCE_TESTNET_SECRET")
		if key == "" || secret == "" {
			log.Fatal("testnet: нужны env BINANCE_TESTNET_KEY и BINANCE_TESTNET_SECRET (testnet.binancefuture.com)")
		}
		te := live.NewTestnetExecutor(logger, key, secret)
		if err := te.Init(ctx, syms, cc.Leverage); err != nil {
			log.Fatalf("testnet init: %v", err)
		}
		exec = te

		// тест исполнения: открыть микро-позицию и закрыть
		if *testOrder != "" {
			parts := strings.Split(*testOrder, ":")
			if len(parts) != 2 {
				log.Fatal("формат: SYMBOL:QTY")
			}
			var qty float64
			fmt.Sscanf(parts[1], "%f", &qty)
			sym := strings.ToUpper(parts[0])
			if err := te.Reconcile(ctx, []string{sym}); err != nil {
				log.Fatal(err)
			}
			if te.Positions()[sym] != nil {
				log.Fatal("testorder: position already exists")
			}
			price, err := te.LastPrice(ctx, sym)
			if err != nil {
				log.Fatalf("testorder price: %v", err)
			}
			fill, err := te.Open(ctx, sym, 1, qty, price*0.97, price*1.03, price)
			if err != nil {
				log.Fatalf("❌ testorder open: %v", err)
			}
			logger.Info("✅ тестовая позиция открыта", "sym", sym, "fill", fill)
			time.Sleep(3 * time.Second)
			fillOut, err := te.Close(ctx, sym, fill)
			if err != nil {
				log.Fatalf("❌ testorder close: %v", err)
			}
			logger.Info("✅ тестовая позиция закрыта", "sym", sym, "fill", fillOut)
			eq, _ := te.Equity(ctx)
			logger.Info("✅ ИСПОЛНЕНИЕ РАБОТАЕТ", "equity", eq)
			return
		}
	} else {
		exec = live.NewLocalExecutor(logger, cc.Equity, cc.Fee, cc.Slip)
	}
	logger.Info("paper trading стартует",
		"mode", exec.Name(), "symbols", syms, "tf", cc.TF,
		"risk", cc.Risk, "max_pos", cc.MaxPositions)

	// Stable strategy identity protects against restoring incompatible ownership.
	strategyJSON, err := json.Marshal(cc.Strategy)
	if err != nil {
		log.Fatal(err)
	}
	strategyKey := fmt.Sprintf("%x", sha256.Sum256(strategyJSON))
	// раннер
	runner := live.NewRunner(logger, live.RunnerConfig{
		Symbols: syms, TF: cc.TF,
		RiskPct: cc.Risk, MaxPositions: cc.MaxPositions, MaxTotalRisk: cc.MaxTotalRisk,
		StatePath: *statePath, StrategyKey: strategyKey,
	}, exec)

	if err := runner.LoadState(); err != nil {
		log.Fatalf("load state: %v", err)
	}
	if err := runner.Reconcile(ctx); err != nil {
		log.Fatalf("initial reconcile: %v", err)
	}

	// прогрев стратегий историей
	for _, sym := range syms {
		candles, err := live.Warmup(ctx, sym, cc.TF, *warmupN)
		if err != nil {
			log.Fatalf("warmup %s: %v", sym, err)
		}
		st, err := strategy.New(cc.Strategy.Name, strategy.Params(cc.Strategy.Params))
		if err != nil {
			log.Fatal(err)
		}
		if err := runner.WarmupStrategy(sym, st, candles); err != nil {
			log.Fatalf("warmup %s: %v", sym, err)
		}
		logger.Info("стратегия прогрета", "sym", sym, "свечей", len(candles),
			"последняя", candles[len(candles)-1].Time.Format("2006-01-02 15:04"))
	}

	if err := runner.SaveState(); err != nil {
		log.Fatalf("save initial state: %v", err)
	}

	klineCh := make(chan live.KlineEvent, 64)
	var feeds sync.WaitGroup
	feeds.Add(2)
	go func() { defer feeds.Done(); live.StreamKlines(ctx, logger, syms, cc.TF, klineCh) }()
	go func() { defer feeds.Done(); live.ReconcileLoop(ctx, logger, syms, cc.TF, klineCh, time.Minute) }()
	heartbeat := time.NewTicker(15 * time.Minute)
	defer heartbeat.Stop()
	positionSync := time.NewTicker(30 * time.Second)
	defer positionSync.Stop()
	beat := func() {
		eq, err := exec.Equity(ctx)
		if err != nil {
			logger.Warn("heartbeat equity", "err", err)
			return
		}
		logger.Info("alive", "equity", eq, "positions", len(exec.Positions()))
	}
	beat()
	logger.Info("в эфире", "tf", cc.TF)
	// Execution, reconciliation and snapshots have one owner; no concurrent mutations.
	for {
		select {
		case <-ctx.Done():
			feeds.Wait()
			if err := runner.SaveState(); err != nil {
				logger.Error("save final state", "err", err)
			}
			logger.Info("остановлен")
			return
		case ev := <-klineCh:
			runner.Handle(ctx, ev)
		case <-positionSync.C:
			if err := runner.Reconcile(ctx); err != nil {
				logger.Error("position reconcile", "err", err)
			}
		case <-heartbeat.C:
			beat()
		}
	}
}
