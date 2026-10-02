// walkforward — честная проверка: скользящие фолды train → test.
//   mode=fixed:    замороженные параметры из base, прогон по test-окнам
//   mode=optimize: на каждом фолде сетка подбирается на train, лучшее — на test
// Пример:
//   go run ./cmd/walkforward -wf configs/walkforward/trend_fixed.yaml
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"gopkg.in/yaml.v3"

	"bot/internal/backtest"
	"bot/internal/candle"
	"bot/internal/config"
	"bot/internal/data"
	"bot/internal/store"
	"bot/internal/strategy"
)

type wfFile struct {
	Mode        string `yaml:"mode"` // fixed | optimize
	Base        string `yaml:"base"` // конфиг прогона (fixed-параметры / база для сетки)
	Grid        string `yaml:"grid"` // sweep-файл сетки (только optimize)
	TrainMonths int    `yaml:"train_months"`
	TestMonths  int    `yaml:"test_months"`
	StepMonths  int    `yaml:"step_months"`
	Metric      string `yaml:"metric"` // sharpe | return | pf
}

type foldResult struct {
	trainStart, testStart, testEnd time.Time
	bestParams                     map[string]any
	trainRep, testRep              *backtest.Report
}

func main() {
	wfPath := flag.String("wf", "", "путь к walkforward YAML")
	flag.Parse()
	if *wfPath == "" {
		log.Fatal("укажи -wf configs/walkforward/xxx.yaml")
	}
	raw, err := os.ReadFile(*wfPath)
	if err != nil {
		log.Fatal(err)
	}
	var wf wfFile
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		log.Fatal(err)
	}
	if wf.Metric == "" {
		wf.Metric = "sharpe"
	}
	if wf.StepMonths == 0 {
		wf.StepMonths = wf.TestMonths
	}

	base, err := config.Load(wf.Base)
	if err != nil {
		log.Fatal(err)
	}
	var grid []map[string]any
	if wf.Mode == "optimize" {
		sf, err := config.LoadSweep(wf.Grid)
		if err != nil {
			log.Fatal(err)
		}
		grid = config.Cartesian(sf.Grid)
	}

	ds, bounds := loadData(base)
	defer ds.db.Close()

	folds := buildFolds(bounds, wf)
	log.Printf("Walk-forward: mode=%s, %d фолдов, train=%dm test=%dm step=%dm\n",
		wf.Mode, len(folds), wf.TrainMonths, wf.TestMonths, wf.StepMonths)

	var results []foldResult
	for _, f := range folds {
		res := foldResult{trainStart: f[0], testStart: f[1], testEnd: f[2]}
		if wf.Mode == "optimize" {
			best, trainRep := optimizeFold(base, grid, ds, f[0], f[1], wf.Metric)
			res.bestParams = best
			res.trainRep = trainRep
		}
		res.testRep = runWindow(base, mergeParams(base, res.bestParams), ds, f[0], f[1], f[2])
		results = append(results, res)
		fmt.Printf("  фолд %s→%s готов\n", f[0].Format("2006-01"), f[2].Format("2006-01"))
	}
	printResults(results, wf)
}

// buildFolds — окна [trainStart, testStart, testEnd).
func buildFolds(candles []candle.Candle, wf wfFile) [][3]time.Time {
	first := time.Date(candles[0].Time.Year(), candles[0].Time.Month(), 1, 0, 0, 0, 0, time.UTC)
	last := candles[len(candles)-1].Time
	var folds [][3]time.Time
	for ts := first; ; ts = ts.AddDate(0, wf.StepMonths, 0) {
		testStart := ts.AddDate(0, wf.TrainMonths, 0)
		testEnd := testStart.AddDate(0, wf.TestMonths, 0)
		if !testStart.Before(last) {
			break
		}
		if testEnd.After(last) {
			testEnd = last
		}
		if testEnd.Sub(testStart) < 30*24*time.Hour {
			break
		}
		folds = append(folds, [3]time.Time{ts, testStart, testEnd})
	}
	return folds
}

func sliceCandles(candles []candle.Candle, from, to time.Time) []candle.Candle {
	var out []candle.Candle
	for _, c := range candles {
		if !c.Time.Before(from) && c.Time.Before(to) {
			out = append(out, c)
		}
	}
	return out
}

func sliceFunding(funding []data.Funding, from, to time.Time) []data.Funding {
	var out []data.Funding
	for _, f := range funding {
		if !f.CalcTime.Before(from) && f.CalcTime.Before(to) {
			out = append(out, f)
		}
	}
	return out
}

func makeCfg(cc *config.Config, startTrading time.Time) backtest.Config {
	return backtest.Config{
		Symbol: strings.ToUpper(cc.Symbol), TF: cc.TF,
		StartEquity: cc.Equity, RiskPct: cc.Risk,
		TakerFeePct: cc.Fee, SlippagePct: cc.Slip,
		UseFunding: cc.UseFunding(), StartTrading: startTrading,
	}
}

// dataset — данные прогона: одиночный символ или портфель.
type dataset struct {
	db        *store.DB
	single    []backtest.SymbolData // всегда как список; 1 элемент = одиночный
	portfolio bool
}

func loadData(cc *config.Config) (*dataset, []candle.Candle) {
	db, err := store.Open(cc.DB)
	if err != nil {
		log.Fatal(err)
	}
	syms := cc.Symbols
	if len(syms) == 0 {
		syms = []string{cc.Symbol}
	}
	d := &dataset{db: db, portfolio: len(cc.Symbols) > 0}
	var bounds []candle.Candle
	for _, s := range syms {
		sym := strings.ToUpper(s)
		candles, err := db.LoadKlines(sym, cc.TF)
		if err != nil || len(candles) == 0 {
			log.Fatalf("нет свечей %s %s: %v", sym, cc.TF, err)
		}
		funding, err := db.LoadFunding(sym)
		if err != nil {
			log.Fatal(err)
		}
		d.single = append(d.single, backtest.SymbolData{Symbol: sym, Candles: candles, Funding: funding})
		if bounds == nil {
			bounds = candles
		}
	}
	return d, bounds
}

func mergeParams(cc *config.Config, override map[string]any) strategy.Params {
	p := strategy.Params{}
	for k, v := range cc.Strategy.Params {
		p[k] = v
	}
	for k, v := range override {
		p[k] = v
	}
	return p
}

// runWindow — прогон [winStart..winEnd), торговля с tradeStart.
func runWindow(cc *config.Config, params strategy.Params, ds *dataset,
	winStart, tradeStart, winEnd time.Time) *backtest.Report {
	cfg := makeCfg(cc, tradeStart)
	if ds.portfolio {
		var win []backtest.SymbolData
		for _, sd := range ds.single {
			win = append(win, backtest.SymbolData{
				Symbol:  sd.Symbol,
				Candles: sliceCandles(sd.Candles, winStart, winEnd),
				Funding: sliceFunding(sd.Funding, winStart, winEnd),
			})
		}
		mk := func(sym string) backtest.Strategy {
			st, err := strategy.New(cc.Strategy.Name, params)
			if err != nil {
				log.Fatal(err)
			}
			return st
		}
		return backtest.RunPortfolio(backtest.PortfolioConfig{
			Config:       cfg,
			MaxPositions: cc.MaxPositions,
			MaxTotalRisk: cc.MaxTotalRisk,
		}, mk, win)
	}
	strat, err := strategy.New(cc.Strategy.Name, params)
	if err != nil {
		log.Fatal(err)
	}
	sd := ds.single[0]
	return backtest.Run(cfg, strat,
		sliceCandles(sd.Candles, winStart, winEnd),
		sliceFunding(sd.Funding, winStart, winEnd),
	)
}

// optimizeFold — сетка на train-окне, лучшее по метрике.
func optimizeFold(cc *config.Config, grid []map[string]any, ds *dataset,
	trainStart, trainEnd time.Time, metric string) (map[string]any, *backtest.Report) {
	var best map[string]any
	var bestRep *backtest.Report
	bestScore := -1e18
	for _, combo := range grid {
		rep := runWindow(cc, mergeParams(cc, combo), ds, trainStart, trainStart, trainEnd)
		score := scoreOf(rep, metric)
		if len(rep.Trades) < 5 { // мало сделок — метрике нельзя верить
			score = -1e18
		}
		if score > bestScore {
			bestScore, best, bestRep = score, combo, rep
		}
	}
	return best, bestRep
}

func scoreOf(r *backtest.Report, metric string) float64 {
	switch metric {
	case "return":
		return r.TotalReturn
	case "pf":
		return r.ProfitFactor
	default:
		return r.Sharpe
	}
}

func printResults(results []foldResult, wf wfFile) {
	fmt.Println("\n================ WALK-FORWARD ================")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Test-окно\tRet%\tPF\tTrades\tWinR%\tMaxDD%\tSharpe\tParams")
	for _, r := range results {
		params := "fixed"
		if r.bestParams != nil {
			params = config.FmtParams(r.bestParams)
		}
		fmt.Fprintf(w, "%s..%s\t%+.2f\t%.2f\t%d\t%.1f\t%.1f\t%.2f\t%s\n",
			r.testStart.Format("2006-01"), r.testEnd.Format("2006-01"),
			100*r.testRep.TotalReturn, r.testRep.ProfitFactor, len(r.testRep.Trades),
			r.testRep.WinRate, 100*r.testRep.MaxDrawdown, r.testRep.Sharpe, params)
		if r.trainRep != nil {
			fmt.Fprintf(w, "  (train: %+.2f%% PF %.2f)\t\t\t\t\t\t\t\n",
				100*r.trainRep.TotalReturn, r.trainRep.ProfitFactor)
		}
	}
	w.Flush()

	// агрегат OOS: компаунд доходности, суммарные сделки
	comp := 1.0
	trades, wins := 0, 0
	var grossWin, grossLoss float64
	for _, r := range results {
		comp *= 1 + r.testRep.TotalReturn
		trades += len(r.testRep.Trades)
		wins += r.testRep.Wins
		for _, t := range r.testRep.Trades {
			if t.PnL > 0 {
				grossWin += t.PnL
			} else {
				grossLoss += t.PnL
			}
		}
	}
	pf := 0.0
	if grossLoss < 0 {
		pf = grossWin / (-grossLoss)
	}
	wr := 0.0
	if trades > 0 {
		wr = 100 * float64(wins) / float64(trades)
	}
	fmt.Println("----------------------------------------------")
	fmt.Printf("OOS итого (компаунд): %+.2f%% | сделок: %d | winrate: %.1f%% | PF: %.2f\n",
		100*(comp-1), trades, wr, pf)
	fmt.Println("==============================================")
}
