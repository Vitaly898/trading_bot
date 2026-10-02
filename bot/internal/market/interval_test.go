package market

import (
	"testing"
	"time"
)

func TestIntervals(t *testing.T) {
	for tf, want := range map[string]time.Duration{"15m": 15 * time.Minute, "4h": 4 * time.Hour, "1w": 7 * 24 * time.Hour} {
		got, err := Interval(tf)
		if err != nil || got != want {
			t.Fatalf("%s: %v %v", tf, got, err)
		}
	}
	for _, tf := range []string{"", "0h", "-1h", "1garbageh", "1M"} {
		if _, err := Interval(tf); err == nil {
			t.Fatal(tf)
		}
	}
}
