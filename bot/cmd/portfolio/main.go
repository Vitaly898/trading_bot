// portfolio — прогон стратегии на корзине символов с общим капиталом.
//   go run ./cmd/portfolio -config configs/portfolio_4h.yaml
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"bot/internal/backtest"
	"bot/internal/config"
	"bot/internal/store"
	"bot/internal/strategy"
)

func main() {
	configPath := flag.String("config", "", "путь к YAML-конфигу портфеля")
	flag.Parse()
	if *configPath == "" {
		log.Fatal("укажи -config configs/portfolio_4h.yaml")
	}
	cc, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	if len(cc.Symbols) == 0 {
		log.Fatal("в конфиге нет symbols")
	}

	db, err := store.Open(cc.DB)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var ds []backtest.SymbolData
	for _, s := range cc.Symbols {
		sym := strings.ToUpper(s)
		candles, err := db.LoadKlines(sym, cc.TF)
		if err != nil {
			log.Fatal(err)
		}
		if len(candles) == 0 {
			log.Printf("⚠️  %s: нет свечей %s — пропускаю", sym, cc.TF)
			continue
		}
		funding, err := db.LoadFunding(sym)
		if err != nil {
			log.Fatal(err)
		}
		ds = append(ds, backtest.SymbolData{Symbol: sym, Candles: candles, Funding: funding})
		log.Printf("%s: %d свечей (%s .. %s), funding %d",
			sym, len(candles),
			candles[0].Time.Format("2006-01-02"), candles[len(candles)-1].Time.Format("2006-01-02"),
			len(funding))
	}

	cfg := backtest.PortfolioConfig{
		Config: backtest.Config{
			Symbol: "PORTFOLIO", TF: cc.TF,
			StartEquity: cc.Equity, RiskPct: cc.Risk,
			TakerFeePct: cc.Fee, SlippagePct: cc.Slip,
			UseFunding: cc.UseFunding(),
		},
		MaxPositions: cc.MaxPositions,
		MaxTotalRisk: cc.MaxTotalRisk,
	}

	mk := func(sym string) backtest.Strategy {
		st, err := strategy.New(cc.Strategy.Name, strategy.Params(cc.Strategy.Params))
		if err != nil {
			log.Fatal(err)
		}
		return st
	}

	var rep *backtest.Report
	if cc.Mode == "split" {
		rep = backtest.RunSplit(cfg.Config, mk, ds)
	} else {
		rep = backtest.RunPortfolio(cfg, mk, ds)
	}
	printReport(rep, ds)
}

func printReport(r *backtest.Report, ds []backtest.SymbolData) {
	fmt.Println("\n========= PORTFOLIO BACKTEST =========")
	fmt.Printf("%d символов | %s .. %s\n", len(ds),
		r.StartTime.Format("2006-01-02"), r.EndTime.Format("2006-01-02"))
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
	fmt.Printf("Fees paid:   %.2f | Funding: %.2f USDT\n", r.TotalFees, r.TotalFunding)
	fmt.Println("-------------- по символам ---------------")

	type symStat struct {
		trades, wins int
		pnl, funding float64
	}
	stats := map[string]*symStat{}
	for _, t := range r.Trades {
		s := stats[t.Symbol]
		if s == nil {
			s = &symStat{}
			stats[t.Symbol] = s
		}
		s.trades++
		if t.PnL > 0 {
			s.wins++
		}
		s.pnl += t.PnL
		s.funding += t.Funding
	}
	syms := make([]string, 0, len(stats))
	for s := range stats {
		syms = append(syms, s)
	}
	sort.Strings(syms)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Symbol\tTrades\tWinR%\tPnL USDT\tFunding")
	for _, s := range syms {
		st := stats[s]
		wr := 0.0
		if st.trades > 0 {
			wr = 100 * float64(st.wins) / float64(st.trades)
		}
		fmt.Fprintf(w, "%s\t%d\t%.1f\t%+.2f\t%.2f\n", s, st.trades, wr, st.pnl, st.funding)
	}
	w.Flush()
	fmt.Println("=========================================")
}
