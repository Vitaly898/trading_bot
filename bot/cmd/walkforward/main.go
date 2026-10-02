// walkforward — честная проверка: скользящие фолды train → test.
//
//	mode=fixed:    замороженные параметры из base, прогон по test-окнам
//	mode=optimize: на каждом фолде сетка подбирается на train, лучшее — на test
//
// Пример:
//
//	go run ./cmd/walkforward -wf configs/walkforward/trend_fixed.yaml
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	"bot/internal/backtest"
	"bot/internal/candle"
	"bot/internal/config"
	"bot/internal/data"
	"bot/internal/experiment"
	"bot/internal/strategy"
)

type wfFile = config.WalkForward

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
	wfConfig, err := config.LoadWalkForward(*wfPath)
	if err != nil {
		log.Fatal(err)
	}
	wf := *wfConfig
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
		if err := config.ValidateGrid(base, grid); err != nil {
			log.Fatal(err)
		}
	}

	ds, bounds := loadData(base)

	folds := buildFolds(bounds, wf)
	if len(folds) == 0 {
		log.Fatal("history does not contain a complete test window")
	}
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
	if len(candles) == 0 || wf.StepMonths <= 0 {
		return nil
	}
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

// dataset is loaded once and reused by every fold.
type dataset struct{ single []backtest.SymbolData }

func loadData(cc *config.Config) (*dataset, []candle.Candle) {
	ds, err := experiment.Open(cc)
	if err != nil {
		log.Fatal(err)
	}
	return &dataset{single: ds}, ds[0].Candles
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
	trial, err := cc.WithParams(params)
	if err != nil {
		log.Fatal(err)
	}
	var win []backtest.SymbolData
	for _, sd := range ds.single {
		win = append(win, backtest.SymbolData{Symbol: sd.Symbol, Candles: sliceCandles(sd.Candles, winStart, winEnd), Funding: sliceFunding(sd.Funding, winStart, winEnd)})
	}
	rep, err := experiment.Run(trial, win, tradeStart)
	if err != nil {
		log.Fatal(err)
	}
	return rep

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
	if bestRep == nil {
		log.Fatal("no eligible optimization candidate: require at least 5 training trades")
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
