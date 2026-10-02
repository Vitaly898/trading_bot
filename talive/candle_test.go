package talive_test

import "time"

type testCandle struct {
	open, high, low, close, volume float64
	timestamp                      time.Time
}

func (c *testCandle) Open() float64        { return c.open }
func (c *testCandle) High() float64        { return c.high }
func (c *testCandle) Low() float64         { return c.low }
func (c *testCandle) Close() float64       { return c.close }
func (c *testCandle) Volume() float64      { return c.volume }
func (c *testCandle) Timestamp() time.Time { return c.timestamp }
