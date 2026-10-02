// Package market contains historical market records, independent of storage and transport.
package market

import "time"

// Kline — одна свеча OHLCV.
type Kline struct {
	OpenTime time.Time
	Open     float64
	High     float64
	Low      float64
	Close    float64
	Volume   float64
}

// Funding — одна запись funding rate.
type Funding struct {
	CalcTime time.Time
	Rate     float64
}
