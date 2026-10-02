package strategy

import (
	"math"
	"testing"

	"bot/internal/candle"
)

func TestERMatchesReference(t *testing.T) {
	for _, prices := range [][]float64{{1, 2, 3, 4, 5, 5, 5, 5}, {1, 2, 1, 2, 1, 2}, {4, 4, 4, 4, 4}} {
		e := NewER(3)
		for i, p := range prices {
			got := e.Next(candle.Candle{C: p})[0]
			want := 0.0
			if i >= 3 {
				vol := 0.0
				for j := i - 2; j <= i; j++ {
					vol += math.Abs(prices[j] - prices[j-1])
				}
				if vol > 0 {
					want = math.Abs(p-prices[i-3]) / vol
				}
			}
			if math.Abs(got-want) > 1e-12 {
				t.Fatalf("prices=%v index=%d: got %g want %g", prices, i, got, want)
			}
		}
	}
}

func TestERCurrentDoesNotAdvanceState(t *testing.T) {
	e, control := NewER(3), NewER(3)
	for _, p := range []float64{1, 2, 3, 4} {
		e.Next(candle.Candle{C: p})
		control.Next(candle.Candle{C: p})
	}
	before := e.out[0]
	got := e.Current(candle.Candle{C: 1})[0]
	if math.Abs(got-1.0/5) > 1e-12 {
		t.Fatalf("preview: %g", got)
	}
	if e.out[0] != before {
		t.Fatalf("Current changed committed output: %g -> %g", before, e.out[0])
	}
	for _, p := range []float64{4, 4, 4, 5, 1} {
		got, want := e.Next(candle.Candle{C: p})[0], control.Next(candle.Candle{C: p})[0]
		if got != want {
			t.Fatalf("Next after preview: %g, want %g", got, want)
		}
	}
}
