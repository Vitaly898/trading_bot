package strategy

import (
	"math"
	"time"

	"bot/internal/candle"

	"github.com/MoroZvlg/talive"
)

// Trend — конфигурируемая трендовая стратегия (Рецепт 1).
//
// Вход:  MA(fast) над/под MA(slow) + фильтры.
// Выход: по сигналу MA (exit=signal) или по Supertrend (exit=supertrend).
// Стоп:  atr_mult × ATR от закрытия сигнальной свечи.
//
// Параметры (yaml):
//
//	ma_type: ema|sma|smma|wma|hma|vwma|kama   (def ema)
//	fast, slow: int                            (def 20/50)
//	adx_period: int, adx_min: float            (def 14 / 0=выкл)
//	ema_filter: int                            (def 0=выкл; long только выше EMA(filter))
//	exit: signal|supertrend                    (def signal)
//	st_period, st_mult: для supertrend         (def 10 / 3.0)
//	atr_period, atr_mult: стоп                 (def 14 / 2.0)
//	stop_pct, take_pct: float                  (def 0=выкл; фиксированные уровни
//	  в долях от цены: 0.03 = 3%. При stop_pct>0 атр-расчёт игнорируется)
//	long_short: bool                           (def true; false = только лонг)
//	funding_long_max: float                    (def 0=выкл; вето на лонг, если funding > порога)
//	funding_short_min: float                   (def 0=выкл; вето на шорт, если funding < порога)
//	  Пороги в долях: 0.0003 = 0.03% за 8ч. Логика MM-фильтра: экстремальный
//	  положительный funding = толпа в лонгах → не встаём на её стороне.
//	tp_r: float                                (def 0=без тейка; тейк = tp_r × stopDist,
//	  где stopDist = atr_mult × ATR. Т.е. тейк кратен риску: tp_r=2 → 2R)
type Trend struct {
	fast, slow talive.Indicator
	adx        talive.Indicator
	emaFilter  talive.Indicator
	atr        talive.Indicator
	st         talive.Indicator

	adxMin   float64
	exitMode string
	atrMult  float64
	tpR      float64
	stopPct  float64
	takePct  float64
	longOnly bool

	fundLongMax  float64
	fundShortMin float64
	lastFunding  float64
	hasFunding   bool
	lastCandle   candle.Candle

	posDir int // фактическое состояние от движка
}

func NewTrend(p Params) (*Trend, error) {
	fast, err := makeMA(p.Str("ma_type", "ema"), p.Int("fast", 20))
	if err != nil {
		return nil, err
	}
	slow, err := makeMA(p.Str("ma_type", "ema"), p.Int("slow", 50))
	if err != nil {
		return nil, err
	}
	t := &Trend{
		fast:     fast,
		slow:     slow,
		atr:      mustIndicator(talive.NewATR(p.Int("atr_period", 14))),
		adxMin:   p.Float("adx_min", 0),
		exitMode: p.Str("exit", "signal"),
		atrMult:  p.Float("atr_mult", 2.0),
		tpR:      p.Float("tp_r", 0),
		stopPct:  p.Float("stop_pct", 0),
		takePct:  p.Float("take_pct", 0),
		longOnly: !p.Bool("long_short", true),
		// 0 = фильтр выключен (funding rate не бывает ровно 0 смысленно)
		fundLongMax:  p.Float("funding_long_max", 0),
		fundShortMin: p.Float("funding_short_min", 0),
	}
	if t.adxMin > 0 {
		t.adx = mustIndicator(talive.NewADX(p.Int("adx_period", 14)))
	}
	if ef := p.Int("ema_filter", 0); ef > 0 {
		t.emaFilter = mustIndicator(talive.NewEMA(ef))
	}
	if t.exitMode == "supertrend" {
		t.st = mustIndicator(talive.NewSupertrend(p.Int("st_period", 10), p.Float("st_mult", 3.0)))
	}
	return t, nil
}

func (t *Trend) OnPositionChange(dir int, _ string) { t.posDir = dir }

func (t *Trend) OnFunding(rate float64, _ time.Time) {
	t.lastFunding = rate
	t.hasFunding = true
}

// fundingVeto — MM-фильтр: не входить на стороне перекошенной толпы.
func (t *Trend) fundingVeto(dir int) bool {
	if !t.hasFunding {
		return false
	}
	if dir > 0 && t.fundLongMax != 0 && t.lastFunding > t.fundLongMax {
		return true // толпа в лонгах и платит — не лонгуем
	}
	if dir < 0 && t.fundShortMin != 0 && t.lastFunding < t.fundShortMin {
		return true // толпа в шортах и платит — не шортим
	}
	return false
}

func (t *Trend) OnCandle(c candle.Candle) (int, float64, float64) {
	t.lastCandle = c
	f := t.fast.Next(c)[0]
	s := t.slow.Next(c)[0]
	a := t.atr.Next(c)[0]

	adxOK := true
	if t.adx != nil {
		adxOK = t.adx.Next(c)[0] >= t.adxMin
	}
	filterOKLong, filterOKShort := true, true
	if t.emaFilter != nil {
		ef := t.emaFilter.Next(c)[0]
		filterOKLong = c.C > ef
		filterOKShort = c.C < ef
	}
	var stVals []float64
	if t.st != nil {
		stVals = t.st.Next(c)
	}

	if t.fast.IsIdle() || t.slow.IsIdle() || t.atr.IsIdle() {
		return 0, 0, 0
	}

	// выход по Supertrend: держим позицию, пока цена не пересекла ST против нас
	if t.exitMode == "supertrend" && t.posDir != 0 && stVals != nil {
		if t.posDir > 0 && c.C < stVals[0] {
			return 0, 0, 0
		}
		if t.posDir < 0 && c.C > stVals[0] {
			return 0, 0, 0
		}
		if adxOK { // условия входа всё ещё валидны — держим
			return t.posDir, 0, 0
		}
		return 0, 0, 0
	}

	// уровни стопа и тейка: stopDist = atr_mult×ATR или фикс stop_pct от цены;
	// takeDist = tp_r×stopDist или фикс take_pct от цены
	stopDist := t.atrMult * a
	if t.stopPct > 0 {
		stopDist = c.C * t.stopPct
	}
	takeDist := t.tpR * stopDist
	if t.takePct > 0 {
		takeDist = c.C * t.takePct
	}

	// сигнальный режим (и вход для supertrend-режима)
	switch {
	case f > s && adxOK && filterOKLong && !t.fundingVeto(1):
		return 1, c.C - stopDist, take(c.C, takeDist, 1)
	case f < s && adxOK && filterOKShort && !t.longOnly && !t.fundingVeto(-1):
		return -1, c.C + stopDist, take(c.C, takeDist, -1)
	}
	return 0, 0, 0
}

// Diagnostics — значения индикаторов тренд-модуля.
func (t *Trend) Diagnostics() map[string]any {
	d := map[string]any{}
	if !t.fast.IsIdle() {
		d["kama_fast"] = round4f(t.fast.Current(t.lastCandle)[0])
	}
	if !t.slow.IsIdle() {
		d["kama_slow"] = round4f(t.slow.Current(t.lastCandle)[0])
	}
	if t.adx != nil && !t.adx.IsIdle() {
		d["adx"] = round2f(t.adx.Current(t.lastCandle)[0])
	}
	if !t.atr.IsIdle() {
		d["atr"] = round2f(t.atr.Current(t.lastCandle)[0])
	}
	d["funding"] = round6f(t.lastFunding)
	return d
}

func round4f(x float64) float64 { return math.Round(x*10000) / 10000 }
func round2f(x float64) float64 { return math.Round(x*100) / 100 }
func round6f(x float64) float64 { return math.Round(x*1000000) / 1000000 }

// take — цена тейка; 0, если тейк выключен (tp_r=0).
func take(close, dist float64, dir int) float64 {
	if dist <= 0 {
		return 0
	}
	return close + float64(dir)*dist
}
