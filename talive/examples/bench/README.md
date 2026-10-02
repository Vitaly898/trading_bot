# Benchmark: talive vs go-talib

Compares **batch computation** performance between
[talive](https://github.com/MoroZvlg/talive) (streaming, one candle at a time)
and [go-talib](https://github.com/markcheno/go-talib) (classic slice-in / slice-out).

> **Important context:** talive is designed for **streaming** — process one candle,
> get a result immediately, zero allocations in steady state. These benchmarks
> intentionally test the *worst case* for talive: computing an entire history from
> scratch (creating the indicator + calling Next N times + collecting into a slice).
> In real streaming usage, talive has no per-candle allocations at all.

Both sides pay the cost of extracting price data from candles (closes/highs/lows)
inside the benchmark loop to ensure a fair comparison.

## Running

```bash
go test -bench . -benchmem -count=3
```

## Results (Apple M3 Pro, arm64)

Ratio = talive / talib. Values **below 1.0x** mean talive is faster (or uses less memory).

### EMA(14)

| Size | talib ns/op | talive ns/op | Ratio    | talib B/op | talive B/op | Mem Ratio |
|------|-------------|--------------|----------|------------|-------------|-----------|
| 100  | 324         | 207          | **0.6x** | 1792 (2)   | 968 (3)     | **0.5x**  |
| 200  | 657         | 386          | **0.6x** | 3584 (2)   | 1864 (3)    | **0.5x**  |
| 500  | 1604        | 914          | **0.6x** | 8192 (2)   | 4168 (3)    | **0.5x**  |
| 1000 | 3185        | 1821         | **0.6x** | 16384 (2)  | 8264 (3)    | **0.5x**  |
| 2000 | 6290        | 3532         | **0.6x** | 32768 (2)  | 16456 (3)   | **0.5x**  |

### SMA(14)

| Size | talib ns/op | talive ns/op | Ratio | talib B/op | talive B/op | Mem Ratio |
|------|-------------|--------------|-------|------------|-------------|-----------|
| 100  | 342         | 479          | 1.4x  | 1792 (2)   | 1008 (2)    | **0.6x**  |
| 200  | 711         | 960          | 1.4x  | 3584 (2)   | 1904 (2)    | **0.5x**  |
| 500  | 1729        | 2313         | 1.3x  | 8192 (2)   | 4208 (2)    | **0.5x**  |
| 1000 | 3406        | 4667         | 1.4x  | 16384 (2)  | 8304 (2)    | **0.5x**  |
| 2000 | 6595        | 9139         | 1.4x  | 32768 (2)  | 16496 (2)   | **0.5x**  |

### RSI(14)

| Size | talib ns/op | talive ns/op | Ratio | talib B/op | talive B/op | Mem Ratio |
|------|-------------|--------------|-------|------------|-------------|-----------|
| 100  | 542         | 676          | 1.2x  | 1792 (2)   | 1160 (7)    | **0.6x**  |
| 200  | 1148        | 1526         | 1.3x  | 3584 (2)   | 2056 (7)    | **0.6x**  |
| 500  | 2869        | 4250         | 1.5x  | 8192 (2)   | 4360 (7)    | **0.5x**  |
| 1000 | 5709        | 8649         | 1.5x  | 16384 (2)  | 8456 (7)    | **0.5x**  |
| 2000 | 11278       | 17634        | 1.6x  | 32768 (2)  | 16648 (7)   | **0.5x**  |

### MACD(12, 26, 9)

| Size | talib ns/op | talive ns/op | Ratio    | talib B/op | talive B/op | Mem Ratio |
|------|-------------|--------------|----------|------------|-------------|-----------|
| 100  | 1023        | 1118         | 1.1x     | 5376 (6)   | 3040 (9)    | **0.6x**  |
| 200  | 2196        | 2033         | **0.9x** | 10752 (6)  | 5216 (9)    | **0.5x**  |
| 500  | 5048        | 4880         | **1.0x** | 24576 (6)  | 12640 (9)   | **0.5x**  |
| 1000 | 10276       | 9542         | **0.9x** | 49152 (6)  | 24928 (9)   | **0.5x**  |
| 2000 | 19989       | 18646        | **0.9x** | 98304 (6)  | 49504 (9)   | **0.5x**  |

### Bollinger Bands(20, 2.0)

| Size | talib ns/op | talive ns/op | Ratio | talib B/op | talive B/op | Mem Ratio |
|------|-------------|--------------|-------|------------|-------------|-----------|
| 100  | 926         | 1580         | 1.7x  | 5376 (6)   | 3632 (15)   | 0.7x      |
| 200  | 1788        | 2772         | 1.6x  | 10752 (6)  | 5808 (15)   | **0.5x**  |
| 500  | 4371        | 6625         | 1.5x  | 24576 (6)  | 13232 (15)  | **0.5x**  |
| 1000 | 8932        | 13005        | 1.5x  | 49152 (6)  | 25520 (15)  | **0.5x**  |
| 2000 | 17662       | 25638        | 1.5x  | 98304 (6)  | 50096 (15)  | **0.5x**  |

### CCI(20)

| Size | talib ns/op | talive ns/op | Ratio    | talib B/op | talive B/op | Mem Ratio |
|------|-------------|--------------|----------|------------|-------------|-----------|
| 100  | 1990        | 1288         | **0.6x** | 3744 (5)   | 1056 (2)    | **0.3x**  |
| 200  | 4278        | 2685         | **0.6x** | 7328 (5)   | 1952 (2)    | **0.3x**  |
| 500  | 11061       | 6860         | **0.6x** | 16544 (5)  | 4256 (2)    | **0.3x**  |
| 1000 | 21936       | 13872        | **0.6x** | 32928 (5)  | 8352 (2)    | **0.3x**  |
| 2000 | 43795       | 27253        | **0.6x** | 65696 (5)  | 16544 (2)   | **0.3x**  |

### ADX(14)

| Size | talib ns/op | talive ns/op | Ratio | talib B/op | talive B/op | Mem Ratio |
|------|-------------|--------------|-------|------------|-------------|-----------|
| 100  | 858         | 1954         | 2.3x  | 3584 (4)   | 1320 (11)   | **0.4x**  |
| 200  | 1780        | 4003         | 2.2x  | 7168 (4)   | 2216 (11)   | **0.3x**  |
| 500  | 4250        | 10300        | 2.4x  | 16384 (4)  | 4520 (11)   | **0.3x**  |
| 1000 | 8329        | 21471        | 2.6x  | 32768 (4)  | 8616 (11)   | **0.3x**  |
| 2000 | 16136       | 42227        | 2.6x  | 65536 (4)  | 16808 (11)  | **0.3x**  |

### Stochastic(14, 3, 3)

| Size | talib ns/op | talive ns/op | Ratio | talib B/op  | talive B/op | Mem Ratio |
|------|-------------|--------------|-------|-------------|-------------|-----------|
| 100  | 1666        | 3685         | 2.2x  | 8000 (10)   | 2528 (15)   | **0.3x**  |
| 200  | 3600        | 8775         | 2.4x  | 16640 (10)  | 3936 (15)   | **0.2x**  |
| 500  | 8582        | 23793        | 2.8x  | 40960 (10)  | 8928 (15)   | **0.2x**  |
| 1000 | 17337       | 54211        | 3.1x  | 81920 (10)  | 17120 (15)  | **0.2x**  |
| 2000 | 33669       | 109170       | 3.2x  | 163841 (10) | 33504 (15)  | **0.2x**  |
