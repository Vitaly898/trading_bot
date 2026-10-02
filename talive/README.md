# talive

[![CI](https://github.com/MoroZvlg/talive/actions/workflows/ci.yml/badge.svg)](https://github.com/MoroZvlg/talive/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/MoroZvlg/talive)](https://goreportcard.com/report/github.com/MoroZvlg/talive)
[![codecov](https://codecov.io/gh/MoroZvlg/talive/branch/main/graph/badge.svg)](https://codecov.io/gh/MoroZvlg/talive)
[![Go Reference](https://pkg.go.dev/badge/github.com/MoroZvlg/talive.svg)](https://pkg.go.dev/github.com/MoroZvlg/talive)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Streaming technical analysis indicators for Go — zero-allocation, bar-by-bar.

Unlike batch TA libraries (ta-lib, go-talib) that recalculate over a full history slice on every update, `talive` maintains running state so each new candle costs **O(1)** regardless of history length. Built for live market feeds, algo trading bots, and any latency-sensitive system.

## Install

```bash
go get github.com/MoroZvlg/talive
```

## Quick Start

```go
import "github.com/MoroZvlg/talive"

// Create indicator
rsi, _ := talive.NewRSI(14)

// Feed candles one by one
for _, candle := range candles {
    values := rsi.Next(candle) // updates state, returns []float64
}

// Peek without updating state
values := rsi.Current(candle)

// Check if indicator has enough data
if rsi.IsIdle() {
    // still warming up
}
```

Candles must implement the `OHLCV` interface:

```go
type OHLCV interface {
    Open() float64
    High() float64
    Low() float64
    Close() float64
    Volume() float64
    Timestamp() time.Time // used by Anchored indicators (e.g. VWAP) for boundary detection
}
```

## Indicators

### Trend

| Indicator | Constructor | Output |
|-----------|-------------|--------|
| EMA | `NewEMA(period)` | `[ema]` |
| SMA | `NewSMA(period)` | `[sma]` |
| SMMA | `NewSMMA(period)` | `[smma]` |
| WMA | `NewWMA(period)` | `[wma]` |
| HMA | `NewHMA(period)` | `[hma]` |
| VWMA | `NewVWMA(period)` | `[vwma]` |
| KAMA | `NewKAMA(period, fastPeriod, slowPeriod)` | `[kama]` |
| MACD | `NewMACD(fast, slow, signal)` | `[macd, signal, hist]` |
| Bollinger Bands | `NewBBands(period, upMult, downMult, maType)` | `[upper, mid, lower]` |
| Parabolic SAR | `NewSAR(start, increment, maxAF)` | `[sar]` |
| Ichimoku Cloud | `NewIchimoku(conv, base, spanB, shift)` | `[tenkan, kijun, spanA, spanB]` |
| ADX | `NewADX(period)` | `[adx]` |
| Aroon | `NewAroon(period)` | `[aroonUp, aroonDown]` |
| Supertrend | `NewSupertrend(period, multiplier)` | `[supertrend, direction]` |
| ZigZag | `NewZigZag(deviation, minTrendLength)` | `[lastHigh, lastLow, event]` |
| Pivot High Low | `NewPivotHL(leftBars, rightBars)` | `[lastPivotHigh, lastPivotLow, event]` |

### Momentum

| Indicator | Constructor | Output |
|-----------|-------------|--------|
| RSI | `NewRSI(period)` | `[rsi]` |
| Stochastic | `NewStochastic(kLen, kSmooth, dSmooth)` | `[k, d]` |
| Stochastic RSI | `NewStochasticRSI(rsiPeriod, stochLen, kSmooth, dSmooth)` | `[k, d]` |
| CCI | `NewCCI(period)` | `[cci]` |
| Williams %R | `NewWilliams(period)` | `[williams]` |
| Ultimate Oscillator | `NewUO(periodMin, periodMid, periodMax)` | `[uo]` |
| Awesome Oscillator | `NewAO()` | `[ao]` |
| Momentum | `NewMomentum(period)` | `[momentum]` |
| ROC | `NewROC(period)` | `[roc]` |
| PPO | `NewPPO(fast, slow, signal)` | `[ppo, signal, hist]` |
| Bull Bear Power | `NewBullBearPower(period)` | `[bbp]` |

### Volatility

| Indicator | Constructor | Output |
|-----------|-------------|--------|
| ATR | `NewATR(period)` | `[atr]` |
| ADR | `NewADR(period)` | `[adr]` |
| StdDev | `NewStdDev(period, ddof)` | `[stddev]` |
| Variance | `NewVariance(period)` | `[variance]` |
| Keltner Channels | `NewKeltnerChannels(period, atrPeriod, multiplier)` | `[upper, basis, lower]` |
| Donchian Channel | `NewDonchianChannels(period)` | `[upper, mid, lower]` |
| Pivot Points | `NewPivotPoints()` | `[pp, r1, r2, r3, s1, s2, s3]` |

### Volume

| Indicator | Constructor | Output |
|-----------|-------------|--------|
| Accumulation/Distribution Line | `NewADL()` | `[adl]` |
| CMF | `NewCMF(period)` | `[cmf]` |
| OBV | `NewOBV()` | `[obv]` |
| VWAP | `NewVWAP()` | `[vwap, upper1, lower1, ...]` |
| MFI | `NewMFI(period)` | `[mfi]` |

### Anchored indicators

Some indicators accumulate state from an anchor point or publish values from completed
anchor periods, and need to be reset at session/period boundaries. They implement the
`Anchored` interface:

```go
type Anchored interface {
    Indicator
    Reset()  // clear accumulated state
}
```

Two ways to use them:

```go
// 1) Auto-reset on time-based boundaries (Daily/Weekly/Monthly/Quarterly/Yearly).
//    Boundary detection uses candle.Timestamp() in its embedded location.
vwap, _ := talive.NewVWAP()
vwap.WithAnchor(talive.AnchorDaily)

// 2) Manual reset for custom anchors (sessions, earnings, splits).
vwap, _ := talive.NewVWAP()
for _, c := range candles {
    if myCustomBoundary(c) {
        vwap.Reset()
    }
    vwap.Next(c)
}
```

## Benchmarks

Measured on Apple M3 Pro:

```
EMA              ~2.8 ns/op    0 B/op    0 allocs/op
SMMA             ~2.8 ns/op    0 B/op    0 allocs/op
SMA              ~4.2 ns/op    0 B/op    0 allocs/op
WMA              ~4.5 ns/op    0 B/op    0 allocs/op
Momentum         ~4.3 ns/op    0 B/op    0 allocs/op
BullBearPower    ~4.8 ns/op    0 B/op    0 allocs/op
ADR              ~5.1 ns/op    0 B/op    0 allocs/op
VWMA             ~5.7 ns/op    0 B/op    0 allocs/op
ZigZag           ~6.6 ns/op    0 B/op    0 allocs/op
ATR              ~6.9 ns/op    0 B/op    0 allocs/op
MACD             ~7.0 ns/op    0 B/op    0 allocs/op
KAMA             ~7.1 ns/op    0 B/op    0 allocs/op
VWAP             ~8.3 ns/op    0 B/op    0 allocs/op
AO               ~8.6 ns/op    0 B/op    0 allocs/op
BBands          ~10.4 ns/op    0 B/op    0 allocs/op
RSI             ~10.5 ns/op    0 B/op    0 allocs/op
ADL             ~10.7 ns/op    0 B/op    0 allocs/op
OBV             ~11.6 ns/op    0 B/op    0 allocs/op
MFI             ~12.5 ns/op    0 B/op    0 allocs/op
HMA             ~12.7 ns/op    0 B/op    0 allocs/op
KeltnerChannels ~12.8 ns/op    0 B/op    0 allocs/op
Williams        ~15.1 ns/op    0 B/op    0 allocs/op
SAR             ~15.8 ns/op    0 B/op    0 allocs/op
UO              ~16.0 ns/op    0 B/op    0 allocs/op
Supertrend      ~17.2 ns/op    0 B/op    0 allocs/op
ADX             ~21.3 ns/op    0 B/op    0 allocs/op
PivotPoints     ~22.8 ns/op    0 B/op    0 allocs/op
PivotHL         ~26.8 ns/op    0 B/op    0 allocs/op
Ichimoku        ~50.1 ns/op    0 B/op    0 allocs/op
DonchianChannels ~54.5 ns/op   0 B/op    0 allocs/op
StochasticRSI   ~68.5 ns/op    0 B/op    0 allocs/op
```

Zero allocations in the hot path — output slices are pre-allocated in constructors and reused on every call.

### vs go-talib (batch mode)

In batch mode (full history from scratch) you can expect **2-5x less memory** but **1.5-3x slower** wall time — some indicators are actually might be faster. Full results in [examples/bench](examples/bench).

## Why talive

- **Streaming-first** — designed for live feeds, not batch recalculation
- **Zero-allocation hot path** — output slices pre-allocated in constructors, reused on every call
- **Simple interface** — `Next`, `Current`, `IsIdle` — that's it
- **No dependencies** — pure Go, nothing to break

## License

MIT
