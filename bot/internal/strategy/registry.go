package strategy

import (
	"fmt"
	"sort"

	"bot/internal/backtest"
)

// Builder — конструктор стратегии из параметров конфига.
type Builder func(p Params) (backtest.Strategy, error)

var builders = map[string]Builder{}

func register(name string, b Builder) {
	builders[name] = b
}

func init() {
	register("ema_cross", func(p Params) (backtest.Strategy, error) {
		return NewEMACrossATR(
			p.Int("fast", 20), p.Int("slow", 50),
			p.Int("atr_period", 14), p.Float("atr_mult", 2.0),
		), nil
	})
	register("trend", func(p Params) (backtest.Strategy, error) {
		return NewTrend(p)
	})
	register("vwap_revert", func(p Params) (backtest.Strategy, error) {
		return NewVWAPRevert(p)
	})
	register("regime_switch", func(p Params) (backtest.Strategy, error) {
		return NewRegimeSwitch(p)
	})
}

// New — стратегия по имени из конфига.
func New(name string, p Params) (backtest.Strategy, error) {
	b, ok := builders[name]
	if !ok {
		return nil, fmt.Errorf("неизвестная стратегия %q (доступны: %v)", name, Names())
	}
	return b(p)
}

// Names — список зарегистрированных стратегий.
func Names() []string {
	out := make([]string, 0, len(builders))
	for n := range builders {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
