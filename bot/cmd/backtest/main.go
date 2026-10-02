// backtest — прогон стратегии по истории из SQLite.
// Основной способ: YAML-конфиг.
//
//	go run ./cmd/backtest -config configs/trend_4h.yaml
//
// Быстрый режим без конфига (ema_cross):
//
//	go run ./cmd/backtest -tf 4h -strategy ema_cross -fast 20 -slow 50
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"bot/internal/backtest"
	"bot/internal/config"
	"bot/internal/experiment"
	"bot/internal/strategy"
)

func main() {
	configPath := flag.String("config", "", "путь к YAML-конфигу")

	// быстрый режим (без конфига) — обратная совместимость
	symbol := flag.String("symbol", "BTCUSDT", "торговая пара")
	tf := flag.String("tf", "1h", "таймфрейм")
	dbPath := flag.String("db", "history.db", "путь к SQLite базе")
	stratName := flag.String("strategy", "ema_cross", "стратегия: "+strings.Join(strategy.Names(), "|"))
	fast := flag.Int("fast", 20, "EMA fast")
	slow := flag.Int("slow", 50, "EMA slow")
	equity := flag.Float64("equity", 10000, "стартовый капитал USDT")
	risk := flag.Float64("risk", 0.01, "риск на сделку (доля капитала)")
	fee := flag.Float64("fee", 0.0005, "комиссия тейкера (доля)")
	slip := flag.Float64("slip", 0.0002, "проскальзывание (доля)")
	noFunding := flag.Bool("no-funding", false, "не учитывать funding rate")
	flag.Parse()

	var cc *config.Config
	var err error
	if *configPath != "" {
		cc, err = config.Load(*configPath)
		if err != nil {
			log.Fatal(err)
		}
		if len(cc.Symbols) > 0 {
			log.Fatal("use cmd/portfolio for symbols configuration")
		}
	} else {
		funding := !*noFunding
		cc = &config.Config{Symbol: strings.ToUpper(*symbol), TF: *tf, DB: *dbPath, Equity: *equity, Risk: *risk, Fee: *fee, Slip: *slip, Funding: &funding}
		cc.Strategy.Name = *stratName
		if *stratName == "ema_cross" || *stratName == "trend" {
			cc.Strategy.Params = map[string]any{"fast": *fast, "slow": *slow}
		}
		if err := cc.Validate(); err != nil {
			log.Fatal(err)
		}
	}
	ds, err := experiment.Open(cc)
	if err != nil {
		log.Fatal(err)
	}
	rep, err := experiment.Run(cc, ds, time.Time{})
	if err != nil {
		log.Fatal(err)
	}
	cs := ds[0].Candles
	printReport(rep, cs[0].C, cs[len(cs)-1].C)

}

func printReport(r *backtest.Report, firstClose, lastClose float64) {
	fmt.Println("========== BACKTEST ==========")
	fmt.Printf("%s %s | %s .. %s\n", r.Symbol, r.TF,
		r.StartTime.Format("2006-01-02"), r.EndTime.Format("2006-01-02"))
	fmt.Printf("Buy&Hold: %+.2f%% (%.0f → %.0f)\n",
		100*(lastClose/firstClose-1), firstClose, lastClose)
	fmt.Println("------------------------------")
	fmt.Printf("Equity:      %.2f → %.2f USDT (%+.2f%%)\n",
		r.StartEquity, r.FinalEquity, 100*r.TotalReturn)
	fmt.Printf("MaxDD:       %.2f%%\n", 100*r.MaxDrawdown)
	fmt.Printf("Sharpe:      %.2f\n", r.Sharpe)
	fmt.Printf("Exposure:    %.1f%%\n", r.ExposurePct)
	fmt.Println("------------------------------")
	fmt.Printf("Trades:      %d (win %d / loss %d, winrate %.1f%%)\n",
		len(r.Trades), r.Wins, r.Losses, r.WinRate)
	fmt.Printf("ProfitFactor: %.2f\n", r.ProfitFactor)
	fmt.Printf("Expectancy:  %.2f USDT/сделку\n", r.Expectancy)
	fmt.Printf("AvgWin/Loss: %.2f / %.2f\n", r.AvgWin, r.AvgLoss)
	fmt.Printf("Выходы:      signal=%d stop=%d take=%d end=%d\n",
		r.ExitReasons["signal"], r.ExitReasons["stop"],
		r.ExitReasons["take"], r.ExitReasons["end"])
	fmt.Println("------------------------------")
	fmt.Printf("Fees paid:   %.2f USDT\n", r.TotalFees)
	fmt.Printf("Funding:     %.2f USDT (отриц. = платили)\n", r.TotalFunding)
	fmt.Println("==============================")
}
