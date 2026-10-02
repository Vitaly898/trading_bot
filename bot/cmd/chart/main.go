// chart — визуализация сделок бэктеста: свечи + маркеры входов/выходов + equity.
// Генерирует автономные HTML (Lightweight Charts от TradingView) в ./charts/.
//   go run ./cmd/chart -config configs/portfolio_4h.yaml
//   go run ./cmd/chart -config configs/walkforward/champion.yaml -symbol BTCUSDT
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"bot/internal/backtest"
	"bot/internal/candle"
	"bot/internal/config"
	"bot/internal/store"
	"bot/internal/strategy"
)

func main() {
	configPath := flag.String("config", "", "путь к YAML-конфигу")
	onlySym := flag.String("symbol", "", "только этот символ (иначе все из конфига)")
	outDir := flag.String("out", "charts", "каталог для HTML")
	flag.Parse()
	if *configPath == "" {
		log.Fatal("укажи -config")
	}
	cc, err := config.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatal(err)
	}

	db, err := store.Open(cc.DB)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	syms := cc.Symbols
	if len(syms) == 0 {
		syms = []string{cc.Symbol}
	}
	portfolio := len(cc.Symbols) > 0

	// готовим данные и единый прогон (портфельный или одиночный)
	var ds []backtest.SymbolData
	for _, s := range syms {
		sym := strings.ToUpper(s)
		if *onlySym != "" && sym != strings.ToUpper(*onlySym) {
			continue
		}
		candles, err := db.LoadKlines(sym, cc.TF)
		if err != nil || len(candles) == 0 {
			log.Printf("⚠️  %s: нет свечей, пропуск", sym)
			continue
		}
		funding, err := db.LoadFunding(sym)
		if err != nil {
			log.Fatal(err)
		}
		ds = append(ds, backtest.SymbolData{Symbol: sym, Candles: candles, Funding: funding})
	}

	cfg := backtest.Config{
		Symbol: syms[0], TF: cc.TF,
		StartEquity: cc.Equity, RiskPct: cc.Risk,
		TakerFeePct: cc.Fee, SlippagePct: cc.Slip,
		UseFunding: cc.UseFunding(),
	}
	mk := func(sym string) backtest.Strategy {
		st, err := strategy.New(cc.Strategy.Name, strategy.Params(cc.Strategy.Params))
		if err != nil {
			log.Fatal(err)
		}
		return st
	}

	var rep *backtest.Report
	if portfolio {
		rep = backtest.RunPortfolio(backtest.PortfolioConfig{
			Config:       cfg,
			MaxPositions: cc.MaxPositions,
			MaxTotalRisk: cc.MaxTotalRisk,
		}, mk, ds)
	} else {
		rep = backtest.Run(cfg, mk(ds[0].Symbol), ds[0].Candles, ds[0].Funding)
		rep.Symbol = ds[0].Symbol
	}

	// сделки по символам
	tradesBySym := map[string][]backtest.Trade{}
	for _, t := range rep.Trades {
		tradesBySym[t.Symbol] = append(tradesBySym[t.Symbol], t)
	}

	// полоса режимов для regime_switch: независимый прогон классификатора
	// (детерминированная функция от свечей — воспроизводит состояние стратегии)
	regimeBySym := map[string][]equityPoint{}
	if cc.Strategy.Name == "regime_switch" {
		for _, sd := range ds {
			clf := strategy.NewRegimeClassifier(strategy.Params(cc.Strategy.Params))
			var pts []equityPoint
			for _, c := range sd.Candles {
				pts = append(pts, equityPoint{Time: c.Time.Unix(), Value: float64(clf.Update(c))})
			}
			regimeBySym[sd.Symbol] = pts
		}
	}

	for _, sd := range ds {
		path := fmt.Sprintf("%s/%s_%s.html", *outDir, sd.Symbol, cc.TF)
		if err := writeChart(path, sd.Candles, tradesBySym[sd.Symbol], rep, sd.Symbol, cc.TF, regimeBySym[sd.Symbol]); err != nil {
			log.Printf("❌ %s: %v", sd.Symbol, err)
			continue
		}
		fmt.Printf("✅ %s → %s (%d сделок)\n", sd.Symbol, path, len(tradesBySym[sd.Symbol]))
	}
	fmt.Printf("\nОткрой в браузере: open %s/\n", *outDir)
}

// ---- HTML ----

type chartCandle struct {
	Time  int64   `json:"time"`
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}

type marker struct {
	Time     int64  `json:"time"`
	Position string `json:"position"`
	Color    string `json:"color"`
	Shape    string `json:"shape"`
	Text     string `json:"text"`
}

type equityPoint struct {
	Time  int64   `json:"time"`
	Value float64 `json:"value"`
}

func reasonColor(reason string) string {
	switch reason {
	case "take":
		return "#26a69a"
	case "stop":
		return "#ef5350"
	case "signal":
		return "#42a5f5"
	default:
		return "#9e9e9e"
	}
}

// reasonIcon — короткая иконка причины выхода для маркера.
func reasonIcon(reason string) string {
	switch reason {
	case "take":
		return "✅" // сработал тейк-профит
	case "stop":
		return "🛑" // сработал стоп-лосс
	case "signal":
		return "↩️" // выход по сигналу стратегии
	default:
		return "🏁" // конец данных
	}
}

func writeChart(path string, candles []candle.Candle, trades []backtest.Trade,
	rep *backtest.Report, sym, tf string, regime []equityPoint) error {

	cs := make([]chartCandle, 0, len(candles))
	for _, c := range candles {
		cs = append(cs, chartCandle{Time: c.Time.Unix(), Open: c.O, High: c.H, Low: c.L, Close: c.C})
	}

	var markers []marker
	for _, t := range trades {
		// вход
		if t.Dir > 0 {
			markers = append(markers, marker{t.EntryTime.Unix(), "belowBar", "#26a69a", "arrowUp", "LONG"})
		} else {
			markers = append(markers, marker{t.EntryTime.Unix(), "aboveBar", "#ab47bc", "arrowDown", "SHORT"})
		}
		// выход: иконка причины + PnL в USDT
		exitText := fmt.Sprintf("%s %+.0f$", reasonIcon(t.ExitReason), t.PnL)
		if t.Dir > 0 {
			markers = append(markers, marker{t.ExitTime.Unix(), "aboveBar", reasonColor(t.ExitReason), "circle", exitText})
		} else {
			markers = append(markers, marker{t.ExitTime.Unix(), "belowBar", reasonColor(t.ExitReason), "circle", exitText})
		}
	}

	// equity как «сделки накопленно» для этого символа
	var eq []equityPoint
	cum := 0.0
	for _, t := range trades {
		cum += t.PnL
		eq = append(eq, equityPoint{Time: t.ExitTime.Unix(), Value: cum})
	}

	candlesJSON, _ := json.Marshal(cs)
	markersJSON, _ := json.Marshal(markers)
	eqJSON, _ := json.Marshal(eq)

	title := fmt.Sprintf("%s %s | сделок: %d | PnL по символу: показан на графике | портфель: %+.1f%%",
		sym, tf, len(trades), 100*rep.TotalReturn)

	html := strings.ReplaceAll(pageTemplate, "__TITLE__", title)
	html = strings.ReplaceAll(html, "__CANDLES__", string(candlesJSON))
	html = strings.ReplaceAll(html, "__MARKERS__", string(markersJSON))
	html = strings.ReplaceAll(html, "__EQUITY__", string(eqJSON))
	return os.WriteFile(path, []byte(html), 0o644)
}

const pageTemplate = `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>__TITLE__</title>
<script src="https://unpkg.com/lightweight-charts@4.2.0/dist/lightweight-charts.standalone.production.js"></script>
<style>
  body { margin:0; background:#131722; color:#d1d4dc; font-family: -apple-system, sans-serif; }
  #header { padding: 10px 16px; font-size: 14px; }
  #chart { width: 100vw; height: 60vh; }
  #regime { width: 100vw; height: 8vh; border-top: 1px solid #2a2e39; }
  #equity { width: 100vw; height: 22vh; border-top: 1px solid #2a2e39; }
  .legend { padding: 4px 16px; font-size: 12px; opacity: .8; }
  .dot { display:inline-block; width:10px; height:10px; border-radius:50%; margin: 0 4px 0 12px;}
</style>
</head>
<body>
<div id="header">__TITLE__</div>
<div class="legend">
  <b>Входы:</b> ▲ LONG (под свечой, зелёная) / ▼ SHORT (над свечой, фиолетовая) &nbsp;|&nbsp;
  <b>Выходы</b> (точка на противоположной стороне): ✅ тейк-профит · 🛑 стоп-лосс · ↩️ разворот сигнала · 🏁 конец данных; рядом P&L сделки в $
</div>
<div id="chart"></div>
<div id="regime"></div>
<div id="equity"></div>
<script>
const chart = LightweightCharts.createChart(document.getElementById('chart'), {
  layout: { background: { color: '#131722' }, textColor: '#d1d4dc' },
  grid: { vertLines: { color: '#1e222d' }, horzLines: { color: '#1e222d' } },
  timeScale: { timeVisible: true, secondsVisible: false },
  crosshair: { mode: LightweightCharts.CrosshairMode.Normal },
});
const series = chart.addCandlestickSeries({
  upColor: '#26a69a', downColor: '#ef5350', borderVisible: false,
  wickUpColor: '#26a69a', wickDownColor: '#ef5350',
});
series.setData(__CANDLES__);
series.setMarkers(__MARKERS__);
chart.timeScale().fitContent();

const eqChart = LightweightCharts.createChart(document.getElementById('equity'), {
  layout: { background: { color: '#131722' }, textColor: '#d1d4dc' },
  grid: { vertLines: { color: '#1e222d' }, horzLines: { color: '#1e222d' } },
  timeScale: { timeVisible: true, secondsVisible: false },
});
const eqSeries = eqChart.addLineSeries({ color: '#42a5f5', lineWidth: 2 });
 eqSeries.setData(__EQUITY__);
eqChart.timeScale().fitContent();

// полоса режимов: 1 = TREND (зелёный), 0 = FLAT (серый)
const rgChart = LightweightCharts.createChart(document.getElementById('regime'), {
  layout: { background: { color: '#131722' }, textColor: '#d1d4dc' },
  grid: { vertLines: { color: '#1e222d' }, horzLines: { visible: false } },
  timeScale: { timeVisible: true, secondsVisible: false },
  rightPriceScale: { visible: false },
});
const rgSeries = rgChart.addAreaSeries({
  lineVisible: false, topColor: 'rgba(38,166,154,0.5)', bottomColor: 'rgba(38,166,154,0.05)',
  priceLineVisible: false, lastValueVisible: false,
});
rgSeries.setData(__REGIME__);
rgChart.priceScale('right').applyOptions({ scaleMargins: { top: 0, bottom: 0 } });
rgChart.timeScale().fitContent();

// синхронизация диапазонов (3 графика)
function sync(src, others) {
  src.timeScale().subscribeVisibleLogicalRangeChange(r => {
    if (!r) return;
    others.forEach(o => o.timeScale().setVisibleLogicalRange(r));
  });
}
sync(chart, [eqChart, rgChart]);
sync(eqChart, [chart, rgChart]);
sync(rgChart, [chart, eqChart]);
</script>
</body>
</html>
`
