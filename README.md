# Trading bot

Go tools for Binance Futures historical data, backtesting, portfolio experiments,
parameter sweeps, walk-forward evaluation, and paper trading.

## Layout

- `bot/`: application module, CLI commands, strategies, and YAML configurations.
- `talive/`: local snapshot of [MoroZvlg/talive](https://github.com/MoroZvlg/talive),
  including its MIT license and tests. Snapshot base:
  `a3da2e1614bb1fbb09f9c3b60b6f47af2b189f17`; local screener module adjustments
  are included. `bot/go.mod` uses this directory through a local `replace`.
- `bot_plan.md`, `realized.md`, `features.md`: project notes and backlog.

## Build and test

Requires Go 1.25 or newer. Run from the corresponding module directory:

```sh
cd bot
go build ./...
go test ./...
cd ../talive
go test ./...
```

The application includes regression tests for backtest accounting, portfolio equity,
live execution and restart recovery, and the Efficiency Ratio indicator. The indicator library also has tests.
Run `go test -race ./...` from `bot/` to include the race detector.

## Historical data and experiments

Run from `bot/`. Historical data is stored locally and is not included in Git.

```sh
make loader SYMBOL=ETHUSDT TFS=4h FROM=2024-01 TO=2025-08
make stats SYMBOL=ETHUSDT TFS=4h
make bt CONFIG=configs/eth_sol_regime.yaml
make pf CONFIG=configs/eth_sol_regime.yaml
make sweep SWEEP=configs/sweeps/regime_classifier.yaml
make wf WF=configs/walkforward/eth_sol_regime.yaml
```

Check each configuration's symbol list and load the required data before running it.
The Makefile sets a Homebrew toolchain path and public Go proxy for the original
development environment; equivalent `go run ./cmd/<command>` commands can be used.

## Paper trading

```sh
cd bot
make paper PCONFIG=configs/eth_sol_regime.yaml
```

Local mode simulates execution using live market data. Testnet mode submits orders
to Binance Futures Testnet and requires `BINANCE_TESTNET_KEY` and
`BINANCE_TESTNET_SECRET` environment variables:

```sh
make papert PCONFIG=configs/eth_sol_regime.yaml
```

Runtime logs, database files, exchange metadata, and position snapshots are excluded
from Git. See `features.md` for pending reliability and engineering work.

## Live state and recovery

The paper runner stores a versioned snapshot in `paper_state.json`. It includes
local equity, positions, protective order IDs, pending client order IDs, processed
candle timestamps, and strategy position state (including RegimeSwitch ownership
and mean-reversion time stops). Snapshots are written atomically with mode `0600`.
Use `-state <path>` with `go run ./cmd/paper` to separate runs or execution modes.

On startup, indicators warm from closed candles and saved position state is
restored. Closed candles missed during downtime count towards time stops, within
the fetched warmup history; trading decisions resume on the next closed candle.
Historical warmup does not submit orders. The strategy, timeframe, symbols, and
execution mode must match the snapshot.

Testnet positions and conditional orders reconcile every 30 seconds in the same
loop as candle execution. Missing known SL/TP orders are restored; confirmed
exchange closures notify the strategy with reason `exchange`. Stop vs. take is
not inferred from candle prices. New entries pause while reconciliation fails,
execution remains uncertain, or a snapshot cannot be saved. Existing protections
remain in place until a market close is confirmed. Position handling supports
one-way mode (`BOTH`); hedge-mode responses are rejected.

Market execution uses confirmed `executedQty` and `avgPrice`. Client order IDs and
entry/close intent are saved before order submission. Uncertain requests are
queried and reconciled rather than blindly resubmitted. Bot conditional orders
use the `tb-` client ID prefix; cleanup preserves unrelated orders.

Legacy snapshots lack local equity and strategy ownership and cannot be safely
converted. They are rejected without overwriting the file. For an explicit fresh
local simulation, use a new path, for example:

```sh
cd bot
go run ./cmd/paper -config configs/eth_sol_regime.yaml -exec local -state paper_local_v1_state.json
```

An existing testnet position without matching saved strategy ownership requires
operator recovery; the runner does not guess the owning RegimeSwitch module.
Execution/recovery tests use in-memory HTTP transports; no exchange credentials
or actual orders are needed. Endpoint fields follow the
[Binance USD-M trade API](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/trade).
