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
and the Efficiency Ratio indicator. The indicator library also has tests.
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
