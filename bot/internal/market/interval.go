package market

import (
	"fmt"
	"time"
)

// Interval supports fixed-length Binance candle intervals used by this application.
// Calendar months are deliberately excluded: they do not have a fixed duration.
func Interval(tf string) (time.Duration, error) {
	intervals := map[string]time.Duration{"1m": time.Minute, "3m": 3 * time.Minute, "5m": 5 * time.Minute, "15m": 15 * time.Minute, "30m": 30 * time.Minute, "1h": time.Hour, "2h": 2 * time.Hour, "4h": 4 * time.Hour, "6h": 6 * time.Hour, "8h": 8 * time.Hour, "12h": 12 * time.Hour, "1d": 24 * time.Hour, "3d": 72 * time.Hour, "1w": 7 * 24 * time.Hour}
	d, ok := intervals[tf]
	if !ok {
		return 0, fmt.Errorf("unsupported candle interval %q", tf)
	}
	return d, nil
}
