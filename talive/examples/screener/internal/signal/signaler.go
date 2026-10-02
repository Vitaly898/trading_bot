package signal

import (
	"screener/internal/binance"
)

type Signaler interface {
	// Next advances indicator state and returns a signal.
	Next(kline *binance.Kline) int // buy = 1, sell = -1, hold = 0
	// Current calculates a signal without advancing state.
	Current(kline *binance.Kline) int // buy = 1, sell = -1, hold = 0
	MaxWarmUp() int
}
