package bench

import (
	"encoding/csv"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/MoroZvlg/talive"
	"github.com/markcheno/go-talib"
)

// sinks prevent compiler from optimizing away benchmark results.
var (
	sinkF64s  []float64
	sinkF64s2 [][2]float64
	sinkF64s3 [][3]float64
)

type candle struct {
	open, high, low, close, volume float64
	timestamp                      time.Time
}

func (c *candle) Open() float64        { return c.open }
func (c *candle) High() float64        { return c.high }
func (c *candle) Low() float64         { return c.low }
func (c *candle) Close() float64       { return c.close }
func (c *candle) Volume() float64      { return c.volume }
func (c *candle) Timestamp() time.Time { return c.timestamp }

var testCandles []candle

func init() {
	testCandles = loadCandles("../../test_data/input_data2.csv")
}

func loadCandles(path string) []candle {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		panic(err)
	}

	candles := make([]candle, 0, len(records)-1)
	for _, row := range records[1:] {
		ts, _ := strconv.ParseInt(row[0], 10, 64)
		o, _ := strconv.ParseFloat(row[1], 64)
		h, _ := strconv.ParseFloat(row[2], 64)
		l, _ := strconv.ParseFloat(row[3], 64)
		c, _ := strconv.ParseFloat(row[4], 64)
		v, _ := strconv.ParseFloat(row[5], 64)
		candles = append(candles, candle{open: o, high: h, low: l, close: c, volume: v, timestamp: time.Unix(ts, 0).UTC()})
	}
	return candles
}

func closes(candles []candle) []float64 {
	out := make([]float64, len(candles))
	for i := range candles {
		out[i] = candles[i].close
	}
	return out
}

func highs(candles []candle) []float64 {
	out := make([]float64, len(candles))
	for i := range candles {
		out[i] = candles[i].high
	}
	return out
}

func lows(candles []candle) []float64 {
	out := make([]float64, len(candles))
	for i := range candles {
		out[i] = candles[i].low
	}
	return out
}

var benchLengths = []int{100, 200, 500, 1000, 2000}

// --- EMA ---

func BenchmarkEMA(b *testing.B) {
	const period = 14
	for _, benchLen := range benchLengths {
		data := testCandles[:benchLen]

		b.Run("talib/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				closeData := closes(data)
				sinkF64s = talib.Ema(closeData, period)
			}
		})

		b.Run("talive/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				ema, _ := talive.NewEMA(period)
				results := make([]float64, benchLen)
				for j := range data {
					results[j] = ema.NextVal(data[j].close)
				}
				sinkF64s = results
			}
		})
	}
}

// --- SMA ---

func BenchmarkSMA(b *testing.B) {
	const period = 14
	for _, benchLen := range benchLengths {
		data := testCandles[:benchLen]

		b.Run("talib/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				closeData := closes(data)
				sinkF64s = talib.Sma(closeData, period)
			}
		})

		b.Run("talive/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				sma, _ := talive.NewSMA(period)
				results := make([]float64, benchLen)
				for j := range data {
					results[j] = sma.NextVal(data[j].close)
				}
				sinkF64s = results
			}
		})
	}
}

// --- RSI ---

func BenchmarkRSI(b *testing.B) {
	const period = 14
	for _, benchLen := range benchLengths {
		data := testCandles[:benchLen]

		b.Run("talib/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				closeData := closes(data)
				sinkF64s = talib.Rsi(closeData, period)
			}
		})

		b.Run("talive/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				rsi, _ := talive.NewRSI(period)
				results := make([]float64, benchLen)
				for j := range data {
					results[j] = rsi.NextVal(data[j].close)
				}
				sinkF64s = results
			}
		})
	}
}

// --- MACD ---

func BenchmarkMACD(b *testing.B) {
	for _, benchLen := range benchLengths {
		data := testCandles[:benchLen]

		b.Run("talib/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				closeData := closes(data)
				a, b2, c := talib.Macd(closeData, 12, 26, 9)
				sinkF64s = a
				sinkF64s = b2
				sinkF64s = c
			}
		})

		b.Run("talive/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				macd, _ := talive.NewMACD(12, 26, 9)
				results := make([][3]float64, benchLen)
				for j := range data {
					c := &data[j]
					out := macd.Next(c)
					results[j] = [3]float64{out[0], out[1], out[2]}
				}
				sinkF64s3 = results
			}
		})
	}
}

// --- Bollinger Bands ---

func BenchmarkBBands(b *testing.B) {
	const period = 20
	for _, benchLen := range benchLengths {
		data := testCandles[:benchLen]

		b.Run("talib/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				closeData := closes(data)
				a, b2, c := talib.BBands(closeData, period, 2.0, 2.0, talib.SMA)
				sinkF64s = a
				sinkF64s = b2
				sinkF64s = c
			}
		})

		b.Run("talive/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				bb, _ := talive.NewBBands(period, 2.0, 2.0)
				results := make([][3]float64, benchLen)
				for j := range data {
					c := &data[j]
					out := bb.Next(c)
					results[j] = [3]float64{out[0], out[1], out[2]}
				}
				sinkF64s3 = results
			}
		})
	}
}

// --- CCI ---

func BenchmarkCCI(b *testing.B) {
	const period = 20
	for _, benchLen := range benchLengths {
		data := testCandles[:benchLen]

		b.Run("talib/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				highData := highs(data)
				lowData := lows(data)
				closeData := closes(data)
				sinkF64s = talib.Cci(highData, lowData, closeData, period)
			}
		})

		b.Run("talive/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				cci, _ := talive.NewCCI(period)
				results := make([]float64, benchLen)
				for j := range data {
					c := &data[j]
					out := cci.Next(c)
					results[j] = out[0]
				}
				sinkF64s = results
			}
		})
	}
}

// --- ADX ---

func BenchmarkADX(b *testing.B) {
	const period = 14
	for _, benchLen := range benchLengths {
		data := testCandles[:benchLen]

		b.Run("talib/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				highData := highs(data)
				lowData := lows(data)
				closeData := closes(data)
				sinkF64s = talib.Adx(highData, lowData, closeData, period)
			}
		})

		b.Run("talive/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				adx, _ := talive.NewADX(period)
				results := make([]float64, benchLen)
				for j := range data {
					c := &data[j]
					out := adx.Next(c)
					results[j] = out[0]
				}
				sinkF64s = results
			}
		})
	}
}

// --- Stochastic ---

func BenchmarkStochastic(b *testing.B) {
	for _, benchLen := range benchLengths {
		data := testCandles[:benchLen]

		b.Run("talib/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				highData := highs(data)
				lowData := lows(data)
				closeData := closes(data)
				k, d := talib.Stoch(highData, lowData, closeData, 14, 3, talib.SMA, 3, talib.SMA)
				sinkF64s = k
				sinkF64s = d
			}
		})

		b.Run("talive/"+strconv.Itoa(benchLen), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				stoch, _ := talive.NewStochastic(14, 3, 3)
				results := make([][2]float64, benchLen)
				for j := range data {
					c := &data[j]
					out := stoch.Next(c)
					results[j] = [2]float64{out[0], out[1]}
				}
				sinkF64s2 = results
			}
		})
	}
}
