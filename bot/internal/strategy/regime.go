package strategy

import (
	"bot/internal/candle"

	"github.com/MoroZvlg/talive"
)

// Режимы рынка.
const (
	RegimeFlat  = 0
	RegimeTrend = 1
)

// RegimeClassifier — автомат определения режима: ADX + ER + гистерезис + мин. длительность.
//
// Флэт → Тренд:  ADX >= adx_enter И ER >= er_enter
// Тренд → Флэт:  ADX < adx_exit ИЛИ ER < er_exit
// Переход не чаще, чем раз в min_bars свечей (анти-дребезг).
//
// Параметры (yaml, префикса нет — живут в params regime_switch):
//
//	adx_period: 14, er_period: 20
//	adx_enter: 25, adx_exit: 20
//	er_enter: 0.25, er_exit: 0.18
//	min_bars: 3
type RegimeClassifier struct {
	adx talive.Indicator
	er  *ER

	adxEnter, adxExit float64
	erEnter, erExit   float64
	minBars           int

	state       int
	barsInState int
	lastAdx     float64 // последние значения детекторов (для диагностики)
	lastER      float64
}

func NewRegimeClassifier(p Params) *RegimeClassifier {
	return &RegimeClassifier{
		adx:      mustIndicator(talive.NewADX(p.Int("adx_period", 14))),
		er:       NewER(p.Int("er_period", 20)),
		adxEnter: p.Float("adx_enter", 25),
		adxExit:  p.Float("adx_exit", 20),
		erEnter:  p.Float("er_enter", 0.25),
		erExit:   p.Float("er_exit", 0.18),
		minBars:  p.Int("min_bars", 3),
		state:    RegimeFlat,
	}
}

// Update — скармливает свечу детекторам, возвращает текущий режим.
func (rc *RegimeClassifier) Update(c candle.Candle) int {
	a := rc.adx.Next(c)[0]
	e := rc.er.Next(c)[0]
	rc.lastAdx, rc.lastER = a, e
	rc.barsInState++

	if rc.adx.IsIdle() || rc.er.IsIdle() || rc.barsInState < rc.minBars {
		return rc.state
	}

	switch rc.state {
	case RegimeFlat:
		if a >= rc.adxEnter && e >= rc.erEnter {
			rc.state = RegimeTrend
			rc.barsInState = 0
		}
	case RegimeTrend:
		if a < rc.adxExit || e < rc.erExit {
			rc.state = RegimeFlat
			rc.barsInState = 0
		}
	}
	return rc.state
}

// State — текущий режим без продвижения.
func (rc *RegimeClassifier) State() int { return rc.state }
