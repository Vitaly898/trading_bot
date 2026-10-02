// sweep — тюнинг-сессия: прогон сетки параметров стратегии, таблица результатов.
//
//	go run ./cmd/sweep -sweep configs/sweeps/trend_4h.yaml
//
// Формат sweep-файла:
//
//	base: configs/trend_4h_baseline.yaml   # базовый конфиг прогона
//	top: 15                                # сколько строк показать (0 = все)
//	grid:                                  # декартово произведение значений
//	  exit: [signal, supertrend]           # ключи = strategy.params.*
//	  adx_min: [0, 20, 25]
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"bot/internal/backtest"
	"bot/internal/candle"
	"bot/internal/config"
	"bot/internal/experiment"
)

type combo struct {
	params map[string]any
	rep    *backtest.Report
}

func main() {
	sweepPath := flag.String("sweep", "", "путь к sweep YAML")
	flag.Parse()
	if *sweepPath == "" {
		log.Fatal("укажи -sweep configs/sweeps/xxx.yaml")
	}

	sf, err := config.LoadSweep(*sweepPath)
	if err != nil {
		log.Fatal(err)
	}
	base, err := config.Load(sf.Base)
	if err != nil {
		log.Fatal(err)
	}
	combos := config.Cartesian(sf.Grid)
	if err := config.ValidateGrid(base, combos); err != nil {
		log.Fatal(err)
	}
	ds, err := experiment.Open(base)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Сетка: %d комбинаций по %s (%d символов)\n", len(combos), sf.Base, len(ds))

	results := make([]combo, 0, len(combos))
	for i, params := range combos {
		cc, err := base.WithParams(params)
		if err != nil {
			log.Fatalf("trial %s: %v", config.FmtParams(params), err)
		}
		rep, err := experiment.Run(cc, ds, time.Time{})
		if err != nil {
			log.Fatal(err)
		}
		results = append(results, combo{params: params, rep: rep})
		fmt.Printf("\r  прогон %d/%d", i+1, len(combos))
	}
	fmt.Println()

	// сортировка по доходности
	sort.Slice(results, func(i, j int) bool {
		return results[i].rep.TotalReturn > results[j].rep.TotalReturn
	})

	top := sf.Top
	if top <= 0 || top > len(results) {
		top = len(results)
	}
	printTable(results[:top], ds[0].Candles)
	fmt.Printf("\n(худший в сетке: %+.2f%%, лучший: %+.2f%%)\n",
		100*results[len(results)-1].rep.TotalReturn, 100*results[0].rep.TotalReturn)
}

func printTable(results []combo, candles []candle.Candle) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Return%%\tPF\tTrades\tWinRate%%\tMaxDD%%\tSharpe\tParams\n")
	fmt.Fprintf(w, "-------\t--\t------\t--------\t------\t------\t------\n")
	for _, r := range results {
		fmt.Fprintf(w, "%+.2f\t%.2f\t%d\t%.1f\t%.1f\t%.2f\t%s\n",
			100*r.rep.TotalReturn, r.rep.ProfitFactor, len(r.rep.Trades),
			r.rep.WinRate, 100*r.rep.MaxDrawdown, r.rep.Sharpe, config.FmtParams(r.params))
	}
	w.Flush()
	bh := 100 * (candles[len(candles)-1].C/candles[0].C - 1)
	fmt.Printf("\nBuy&Hold за период: %+.2f%%\n", bh)
}
