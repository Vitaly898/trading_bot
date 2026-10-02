// Package candle — свеча, реализующая интерфейс talive.OHLCV.
package candle

import "time"

type Candle struct {
	Time   time.Time
	O, H, L, C, V float64
}

func (c Candle) Open() float64           { return c.O }
func (c Candle) High() float64           { return c.H }
func (c Candle) Low() float64            { return c.L }
func (c Candle) Close() float64          { return c.C }
func (c Candle) Volume() float64         { return c.V }
func (c Candle) Timestamp() time.Time    { return c.Time }
