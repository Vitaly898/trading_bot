# Обертка над go с фиксированным окружением проекта:
# локальный toolchain (без авто-скачивания) + публичный proxy (не корпоративный).
export PATH := /opt/homebrew/bin:$(PATH)
export GOTOOLCHAIN := local
export GOPROXY := https://proxy.golang.org,direct

SYMBOL ?= BTCUSDT
TFS    ?= 15m,1h,4h
FROM   ?= 2024-01
TO     ?= 2025-08
DB     ?= history.db

.PHONY: build test loader stats clean-cache

build:
	go build ./...

test:
	go test ./...

# Загрузка истории: make loader SYMBOL=ETHUSDT FROM=2024-01 TO=2025-08
loader:
	go run ./cmd/loader -symbol $(SYMBOL) -tfs $(TFS) -from $(FROM) -to $(TO) -db $(DB)

# Проверка базы: make stats
stats:
	go run ./cmd/dbstats -symbol $(SYMBOL) -tfs $(TFS) -db $(DB)

# Бэктест по конфигу: make bt CONFIG=configs/trend_4h.yaml
# Быстрый режим без конфига: make btq TF=4h FAST=20 SLOW=50
CONFIG ?= configs/trend_4h_baseline.yaml
TF ?= 1h
FAST ?= 20
SLOW ?= 50
bt:
	go run ./cmd/backtest -config $(CONFIG)

btq:
	go run ./cmd/backtest -symbol $(SYMBOL) -tf $(TF) -strategy ema_cross -fast $(FAST) -slow $(SLOW) -db $(DB)

# Тюнинг-сессия: make sweep SWEEP=configs/sweeps/trend_4h.yaml
SWEEP ?= configs/sweeps/trend_4h.yaml
sweep:
	go run ./cmd/sweep -sweep $(SWEEP)

# Walk-forward: make wf WF=configs/walkforward/trend_fixed.yaml
WF ?= configs/walkforward/trend_fixed.yaml
wf:
	go run ./cmd/walkforward -wf $(WF)

# Портфель: make pf CONFIG=configs/portfolio_4h.yaml
pf:
	go run ./cmd/portfolio -config $(CONFIG)

# Paper trading: make paper (local) / make papert (testnet, нужны env-ключи)
PCONFIG ?= configs/eth_sol_regime.yaml
paper:
	go run ./cmd/paper -config $(PCONFIG) -exec local

papert:
	@test -n "$$BINANCE_TESTNET_KEY" || (echo "нужны env BINANCE_TESTNET_KEY/BINANCE_TESTNET_SECRET" && exit 1)
	go run ./cmd/paper -config $(PCONFIG) -exec testnet

# Лечение "version goX does not match go tool version goY"
clean-cache:
	go clean -cache
