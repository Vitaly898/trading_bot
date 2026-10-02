# Indicators Roadmap

## Already Implemented

| #  | Indicator | Category | Usage |
|----|-----------|----------|-------|
| 1  | SMA | Moving Average | 10/10 |
| 2  | EMA | Moving Average | 10/10 |
| 3  | WMA | Moving Average | 6/10 |
| 4  | SMMA (Wilder's) | Moving Average | 5/10 |
| 5  | HMA | Moving Average | 7/10 |
| 6  | VWMA | Moving Average | 7/10 |
| 7  | RSI | Oscillator | 10/10 |
| 8  | MACD | Trend/Momentum | 10/10 |
| 9  | Bollinger Bands | Volatility | 10/10 |
| 10 | Stochastic | Oscillator | 9/10 |
| 11 | Stochastic RSI | Oscillator | 8/10 |
| 12 | ATR | Volatility | 9/10 |
| 13 | ADX | Trend | 8/10 |
| 14 | DMI (+DI/-DI) | Trend | 8/10 |
| 15 | CCI | Oscillator | 7/10 |
| 16 | Williams %R | Oscillator | 7/10 |
| 17 | MFI | Volume/Oscillator | 7/10 |
| 18 | Ichimoku Cloud | Trend/Complex | 8/10 |
| 19 | Parabolic SAR | Trend | 8/10 |
| 20 | AO (Awesome Osc) | Momentum | 6/10 |
| 21 | UO (Ultimate Osc) | Momentum | 5/10 |
| 22 | Momentum | Momentum | 6/10 |
| 23 | Bull Bear Power | Momentum | 5/10 |
| 24 | OBV (On Balance Volume) | Volume | 9/10 |
| 25 | VWAP | Volume | 9/10 |
| 26 | Supertrend | Trend | 9/10 |
| 27 | Keltner Channel | Volatility | 8/10 |
| 28 | DEMA | Moving Average | 7/10 |
| 29 | TEMA | Moving Average | 7/10 |
| 30 | Donchian Channel | Volatility | 7/10 |
| 31 | Pivot Points | Support/Resistance | 8/10 |
| 32 | ZigZag | Pattern | 5/10 |
| 33 | ADR (Average Daily Range) | Volatility | 6/10 |
| 34 | ADL (Accumulation/Distribution) | Volume | 7/10 |
| 35 | KAMA (Kaufman Adaptive MA) | Moving Average | 6/10 |
| 36 | Pivot High Low | Pattern | 6/10 |
| 37 | Aroon | Trend | 7/10 |
| 38 | ROC (Rate of Change) | Momentum | 7/10 |
| 39 | CMF (Chaikin Money Flow) | Volume | 7/10 |
| 40 | PPO (Percentage Price Osc) | Momentum | 6/10 |

## To Build

Sorted by priority. Usage is approximate popularity among retail/algo traders.

| Priority | Indicator | Category | Usage | Comment |
|----------|-----------|----------|-------|---------|
| 11 | TRIX | Momentum | 6/10 | Triple-smoothed EMA rate of change. Good noise filter, used for signal line crossovers |
| 12 | EOM (Ease of Movement) | Volume | 5/10 | Relates price change to volume. Useful for detecting low-volume breakouts |
| 14 | DPO (Detrended Price Osc) | Momentum | 5/10 | Removes trend to isolate cycles. Niche but used in cycle analysis |
| 15 | Chande Momentum Osc (CMO) | Momentum | 5/10 | Like RSI but unsmoothed. Used in adaptive MA systems (Chande's VIDYA) |
| 16 | VIDYA | Moving Average | 5/10 | Volatility-adaptive MA using CMO. Adjusts speed based on market conditions |
| 18 | Coppock Curve | Momentum | 4/10 | Long-term momentum for monthly charts. Originally for S&P 500 bottom detection |
| 19 | KST (Know Sure Thing) | Momentum | 5/10 | Weighted sum of 4 ROC smoothed values. Martin Pring's trend confirmation tool |
| 20 | Mass Index | Volatility | 4/10 | Detects trend reversals via range narrowing/widening ("reversal bulge") |
| 21 | TSI (True Strength Index) | Momentum | 5/10 | Double-smoothed momentum. Cleaner than RSI for divergences |
| 22 | Connors RSI | Oscillator | 5/10 | Composite of RSI + streak RSI + ROC percentile. Popular in mean-reversion algos |
| 23 | Chaikin Oscillator | Volume | 5/10 | MACD applied to ADL. Fast/slow ADL crossover for volume momentum |
| 24 | Force Index | Volume | 5/10 | Elder's price change * volume. Measures the force behind moves |
| 25 | ATR Trailing Stop | Volatility | 7/10 | ATR-based dynamic stop-loss. Used by almost everyone who uses ATR |
| 26 | Stc (Schaff Trend Cycle) | Trend | 5/10 | MACD through stochastic filter. Faster signals than MACD alone |
| 27 | McGinley Dynamic | Moving Average | 4/10 | Self-adjusting MA that tracks price better in fast markets |
| 28 | ALMA (Arnaud Legoux MA) | Moving Average | 5/10 | Gaussian-weighted MA. Smooth, low lag. Gaining popularity on TradingView |
| 29 | Klinger Oscillator | Volume | 4/10 | Volume force oscillator. Detects long-term money flow vs short-term |
| 30 | Vortex Indicator | Trend | 5/10 | +VI/-VI based on true range. Identifies trend start and direction |
| 31 | RVI (Relative Vigor Index) | Momentum | 4/10 | Compares close-open to high-low range. Measures conviction of moves |
| 32 | Elder Ray (Bull/Bear Power) | Momentum | 4/10 | You have Bull Bear Power already; this is the full Elder Ray system with EMA |
| 33 | Chop Index (CHOP) | Volatility | 5/10 | Measures if market is choppy (ranging) or trending. Good as a filter |
| 35 | Fibonacci Retracement | Support/Resistance | 8/10 | Not a traditional indicator, but worth having as a utility. Every trader knows Fib levels |
| 36 | Linear Regression Channel | Statistical | 5/10 | Regression line + stddev bands. Used for mean-reversion and trend channels |
