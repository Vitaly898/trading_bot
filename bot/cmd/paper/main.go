// paper — paper trading: реальные данные Binance, исполнение local или testnet.
//
//	local:   go run ./cmd/paper -config configs/eth_sol_regime.yaml
//	testnet: BINANCE_TESTNET_KEY=... BINANCE_TESTNET_SECRET=... \
//	         go run ./cmd/paper -config configs/eth_sol_regime.yaml -exec testnet
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"bot/internal/backtest"
	"bot/internal/config"
	"bot/internal/live"
	"bot/internal/strategy"
)

func main() {
	configPath := flag.String("config", "", "путь к YAML-конфигу")
	execMode := flag.String("exec", "local", "local | testnet")
	warmupN := flag.Int("warmup", 300, "свечей прогрева")
	testOrder := flag.String("testorder", "", "тест исполнения: SYMBOL:QTY (напр. ETHUSDT:0.005) — открыть и закрыть, выйти")
	flag.Parse()
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
		ctx := context.Background()
		if err := te.Init(ctx, syms, cc.Leverage); err != nil {
			log.Fatalf("testnet init: %v", err)
		}
		if err := te.Reconcile(ctx, syms); err != nil {
			log.Fatalf("testnet reconcile: %v", err)
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// раннер
	runner := live.NewRunner(logger, live.RunnerConfig{
		Symbols: syms, TF: cc.TF,
		RiskPct: cc.Risk, MaxPositions: cc.MaxPositions, MaxTotalRisk: cc.MaxTotalRisk,
		StatePath: "paper_state.json",
	}, exec)

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
		for _, c := range candles {
			st.OnCandle(c)
		}
		runner.RegisterStrategy(sym, st)
		runner.MarkClosed(sym, candles[len(candles)-1].Time) // не скармливать повторно из reconcile
		logger.Info("стратегия прогрета", "sym", sym, "свечей", len(candles),
			"последняя", candles[len(candles)-1].Time.Format("2006-01-02 15:04"))
	}

	// heartbeat: сразу при старте, дальше раз в 15 минут
	go func() {
		beat := func() {
			eq, _ := exec.Equity(ctx)
			logger.Info("💓 alive", "equity", fmt.Sprintf("%.2f", eq),
				"positions", len(exec.Positions()))
		}
		beat()
		tick := time.NewTicker(15 * time.Minute)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				beat()
			}
		}
	}()

	// поток свечей + gap-recovery (раз в минуту сверка с REST)
	klineCh := make(chan live.KlineEvent, 64)
	go live.StreamKlines(ctx, logger, syms, cc.TF, klineCh)
	go live.ReconcileLoop(ctx, logger, syms, cc.TF, klineCh, time.Minute)
	go func() {
		for ev := range klineCh {
			runner.Handle(ctx, ev)
		}
	}()

	logger.Info("✅ в эфире. Жду закрытия свечей", "tf", cc.TF)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	logger.Info("остановка...")
	cancel()
	time.Sleep(500 * time.Millisecond)
	logger.Info("пока-пока")
}

var _ = backtest.Config{} // keep import
