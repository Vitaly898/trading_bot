// Package accounting shares sizing and cost formulas between simulation and execution.
package accounting

import "math"

func Slippage(price float64, dir int, fraction float64) float64 {
	return price * (1 + float64(dir)*fraction)
}
func Fee(price, qty, fraction float64) float64        { return price * qty * fraction }
func Gross(dir int, entry, exit, qty float64) float64 { return float64(dir) * (exit - entry) * qty }
func StopDistance(entry, stop float64) float64 {
	if stop > 0 && math.Abs(entry-stop) > 0 {
		return math.Abs(entry - stop)
	}
	return entry * 0.02
}

// Quantity returns zero when capital or sizing inputs cannot support a position.
func Quantity(equity, risk, distance float64) float64 {
	if equity <= 0 || risk <= 0 || distance <= 0 || math.IsNaN(equity+risk+distance) || math.IsInf(equity+risk+distance, 0) {
		return 0
	}
	qty := equity * risk / distance
	if math.IsInf(qty, 0) || math.IsNaN(qty) {
		return 0
	}
	return qty
}
