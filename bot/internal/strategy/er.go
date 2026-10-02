package strategy

import (
	"github.com/MoroZvlg/talive"
)

// ER — Efficiency Ratio Кауфмана: |чистое движение| / сумма шагов.
// ER → 1 в чистом тренде, ER → 0 в идеальном шуме/пиле.
// Главный измеритель боковика для классификатора режима.
// В talive не экспонируется (живёт внутри KAMA) — реализуем сами.
type ER struct {
	period int
	closes []float64 // последние period+1 закрытий
	out    []float64
}

func NewER(period int) *ER {
	return &ER{period: period, out: []float64{0}}
}

func (e *ER) Next(c talive.OHLCV) []float64 {
	e.closes = append(e.closes, c.Close())
	if len(e.closes) > e.period+1 {
		e.closes = e.closes[1:]
	}
	if e.IsIdle() {
		return e.out
	}
	change := absF(e.closes[len(e.closes)-1] - e.closes[0])
	vol := 0.0
	for i := 1; i < len(e.closes); i++ {
		vol += absF(e.closes[i] - e.closes[i-1])
	}
	if vol > 0 {
		e.out[0] = change / vol
	}
	return e.out
}

func (e *ER) Current(c talive.OHLCV) []float64 {
	snapshot := append([]float64{}, e.closes...)
	out := e.Next(c)
	e.closes = snapshot
	return out
}

func (e *ER) IsIdle() bool        { return len(e.closes) < e.period+1 }
func (e *ER) IdlePeriod() int     { return e.period + 1 }
func (e *ER) IsWarmedUp() bool    { return !e.IsIdle() }
func (e *ER) WarmUpPeriod() int   { return e.period + 1 }

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
