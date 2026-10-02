// sweep — тюнинг-сессия: прогон сетки параметров стратегии, таблица результатов.
//   go run ./cmd/sweep -sweep configs/sweeps/trend_4h.yaml
//
// Формат sweep-файла:
//   base: configs/trend_4h_baseline.yaml   # базовый конфиг прогона
//   top: 15                                # сколько строк показать (0 = все)
//   grid:                                  # декартово произведение значений
//     exit: [signal, supertrend]           # ключи = strategy.params.*
//     adx_min: [0, 20, 25]
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"

	"bot/internal/backtest"
	"bot/internal/candle"
	"bot/internal/config"
	"bot/internal/store"
	"bot/internal/strategy"
)

type sweepFile struct {
	Base string           `yaml:"base"`
	Top  int              `yaml:"top"`
	Grid map[string][]any `yaml:"grid"`
}

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

	raw, err := os.ReadFile(*sweepPath)
	if err != nil {
		log.Fatal(err)
	}
	var sf sweepFile
	if err := yaml.Unmarshal(raw, &sf); err != nil {
		log.Fatal(err)
	}
	if len(sf.Grid) == 0 {
		log.Fatal("пустая grid")
	}

	// данные грузим один раз (одиночный символ или портфель)
	base, err := config.Load(sf.Base)
	if err != nil {
		log.Fatal(err)
	}
	db, err := store.Open(base.DB)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	syms := base.Symbols
	portfolio := len(syms) > 0
	if !portfolio {
		syms = []string{base.Symbol}
	}
	var ds []backtest.SymbolData
	for _, s := range syms {
		sym := strings.ToUpper(s)
		candles, err := db.LoadKlines(sym, base.TF)
		if err != nil || len(candles) == 0 {
			log.Fatalf("нет свечей %s %s: %v", sym, base.TF, err)
		}
		funding, err := db.LoadFunding(sym)
		if err != nil {
			log.Fatal(err)
		}
		ds = append(ds, backtest.SymbolData{Symbol: sym, Candles: candles, Funding: funding})
	}

	combos := config.Cartesian(sf.Grid)
	log.Printf("Сетка: %d комбинаций по %s (%d символов)\n", len(combos), sf.Base, len(ds))

	results := make([]combo, 0, len(combos))
	for i, params := range combos {
		cc, err := config.Load(sf.Base) // свежий конфиг на каждый прогон
		if err != nil {
			log.Fatal(err)
		}
		for k, v := range params {
			if cc.Strategy.Params == nil {
				cc.Strategy.Params = map[string]any{}
			}
			cc.Strategy.Params[k] = v
		}
		cfg := backtest.Config{
			Symbol: ds[0].Symbol, TF: cc.TF,
			StartEquity: cc.Equity, RiskPct: cc.Risk,
			TakerFeePct: cc.Fee, SlippagePct: cc.Slip,
			UseFunding: cc.UseFunding(),
		}
		var rep *backtest.Report
		if portfolio {
			mk := func(string) backtest.Strategy { return stratFresh(cc) }
			rep = backtest.RunPortfolio(backtest.PortfolioConfig{
				Config:       cfg,
				MaxPositions: cc.MaxPositions,
				MaxTotalRisk: cc.MaxTotalRisk,
			}, mk, ds)
		} else {
			rep = backtest.Run(cfg, stratFresh(cc), ds[0].Candles, ds[0].Funding)
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

func stratFresh(cc *config.Config) backtest.Strategy {
	st, err := strategy.New(cc.Strategy.Name, strategy.Params(cc.Strategy.Params))
	if err != nil {
		log.Fatal(err)
	}
	return st
}

// cartesian — декартово произведение значений grid. Ключи сортируем для детерминизма.
func cartesian(grid map[string][]any) []map[string]any {
	keys := make([]string, 0, len(grid))
	for k := range grid {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := []map[string]any{{}}
	for _, k := range keys {
		var next []map[string]any
		for _, c := range out {
			for _, v := range grid[k] {
				nc := make(map[string]any, len(c)+1)
				for ck, cv := range c {
					nc[ck] = cv
				}
				nc[k] = v
				next = append(next, nc)
			}
		}
		out = next
	}
	return out
}

func printTable(results []combo, candles []candle.Candle) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Return%%\tPF\tTrades\tWinRate%%\tMaxDD%%\tSharpe\tParams\n")
	fmt.Fprintf(w, "-------\t--\t------\t--------\t------\t------\t------\n")
	for _, r := range results {
		fmt.Fprintf(w, "%+.2f\t%.2f\t%d\t%.1f\t%.1f\t%.2f\t%s\n",
			100*r.rep.TotalReturn, r.rep.ProfitFactor, len(r.rep.Trades),
			r.rep.WinRate, 100*r.rep.MaxDrawdown, r.rep.Sharpe, fmtParams(r.params))
	}
	w.Flush()
	bh := 100 * (candles[len(candles)-1].C/candles[0].C - 1)
	fmt.Printf("\nBuy&Hold за период: %+.2f%%\n", bh)
}

func fmtParams(p map[string]any) string {
	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, p[k]))
	}
	return strings.Join(parts, " ")
}
