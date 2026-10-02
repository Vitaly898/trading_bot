// backtest — прогон стратегии по истории из SQLite.
// Основной способ: YAML-конфиг.
//   go run ./cmd/backtest -config configs/trend_4h.yaml
// Быстрый режим без конфига (ema_cross):
//   go run ./cmd/backtest -tf 4h -strategy ema_cross -fast 20 -slow 50
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"bot/internal/backtest"
	"bot/internal/config"
	"bot/internal/store"
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

	var (
		sym       string
		timeframe string
		dbFile    string
		cfg       backtest.Config
		strat     backtest.Strategy
		err       error
	)

	if *configPath != "" {
		var cc *config.Config
		cc, err = config.Load(*configPath)
		if err != nil {
			log.Fatal(err)
		}
		sym, timeframe, dbFile = strings.ToUpper(cc.Symbol), cc.TF, cc.DB
		cfg = backtest.Config{
			Symbol: sym, TF: timeframe,
			StartEquity: cc.Equity, RiskPct: cc.Risk,
			TakerFeePct: cc.Fee, SlippagePct: cc.Slip,
			UseFunding: cc.UseFunding(),
		}
		strat, err = strategy.New(cc.Strategy.Name, strategy.Params(cc.Strategy.Params))
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Конфиг: %s | стратегия %s %v", *configPath, cc.Strategy.Name, cc.Strategy.Params)
	} else {
		sym, timeframe, dbFile = strings.ToUpper(*symbol), *tf, *dbPath
		cfg = backtest.Config{
			Symbol: sym, TF: timeframe,
			StartEquity: *equity, RiskPct: *risk,
			TakerFeePct: *fee, SlippagePct: *slip,
			UseFunding: !*noFunding,
		}
		strat, err = strategy.New(*stratName, strategy.Params{
			"fast": *fast, "slow": *slow,
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	db, err := store.Open(dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	candles, err := db.LoadKlines(sym, timeframe)
	if err != nil {
		log.Fatal(err)
	}
	if len(candles) == 0 {
		log.Fatalf("нет свечей %s %s в %s — сначала запусти loader", sym, timeframe, dbFile)
	}
	funding, err := db.LoadFunding(sym)
	if err != nil {
		log.Fatal(err)
	}

	rep := backtest.Run(cfg, strat, candles, funding)
	printReport(rep, candles[0].C, candles[len(candles)-1].C)
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
