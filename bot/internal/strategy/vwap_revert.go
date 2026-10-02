package strategy

import (
	"bot/internal/candle"

	"github.com/MoroZvlg/talive"
)

// VWAPRevert — mean-reversion к VWAP (Рецепт 2, «сделка на стороне ММ»).
//
// Вход long:  close отклонился ниже VWAP больше чем на dev_mult×ATR
//             + RSI развернулся вверх из зоны перепроданности + ADX < adx_max (флэт).
// Вход short: симметрично.
// Выход:      достижение VWAP / временной стоп (time_stop свечей) / RSI в противоположной зоне.
// Стоп:       atr_mult × ATR от закрытия сигнальной свечи.
//
// Параметры (yaml):
//   rsi_period: int                            (def 14)
//   oversold, overbought: float                (def 30 / 70)
//   dev_mult: float                            (def 1.5)
//   adx_period: int, adx_max: float            (def 14 / 0=выкл)
//   atr_period, atr_mult: стоп                 (def 14 / 1.0)
//   time_stop: int (свечей, 0=выкл)            (def 12)
//   tp_r: float                                (def 0=биржевого тейка нет; при >0
//     жёсткий тейк = tp_r × stopDist, вдобавок к выходу у VWAP)
//   stop_pct, take_pct: float                  (def 0=выкл; фикс. уровни от цены)
//   long_short: bool                           (def true)
type VWAPRevert struct {
	vwap talive.Indicator
	rsi  talive.Indicator
	adx  talive.Indicator
	atr  talive.Indicator

	oversold   float64
	overbought float64
	devMult    float64
	atrMult    float64
	adxMax     float64
	tpR        float64
	stopPct    float64
	takePct    float64
	timeStop   int
	longOnly   bool

	prevRSI float64
	hasPrev bool
	posDir  int
	barsIn  int
	lastCandle candle.Candle
}

// Diagnostics — значения индикаторов флэт-модуля.
func (s *VWAPRevert) Diagnostics() map[string]any {
	d := map[string]any{}
	if !s.vwap.IsIdle() {
		d["vwap"] = round4f(s.vwap.Current(s.lastCandle)[0])
	}
	if !s.rsi.IsIdle() {
		d["rsi"] = round2f(s.rsi.Current(s.lastCandle)[0])
	}
	if !s.atr.IsIdle() {
		d["atr"] = round2f(s.atr.Current(s.lastCandle)[0])
	}
	return d
}

func NewVWAPRevert(p Params) (*VWAPRevert, error) {
	vwap, err := talive.NewVWAP()
	if err != nil {
		return nil, err
	}
	vwap.WithAnchor(talive.AnchorDaily)
	s := &VWAPRevert{
		vwap:       vwap,
		rsi:        mustIndicator(talive.NewRSI(p.Int("rsi_period", 14))),
		atr:        mustIndicator(talive.NewATR(p.Int("atr_period", 14))),
		oversold:   p.Float("oversold", 30),
		overbought: p.Float("overbought", 70),
		devMult:    p.Float("dev_mult", 1.5),
		atrMult:    p.Float("atr_mult", 1.0),
		tpR:        p.Float("tp_r", 0),
		stopPct:    p.Float("stop_pct", 0),
		takePct:    p.Float("take_pct", 0),
		timeStop:   p.Int("time_stop", 12),
		longOnly:   !p.Bool("long_short", true),
	}
	s.adxMax = p.Float("adx_max", 0)
	if s.adxMax > 0 {
		s.adx = mustIndicator(talive.NewADX(p.Int("adx_period", 14)))
	}
	return s, nil
}

func (s *VWAPRevert) OnPositionChange(dir int, _ string) {
	s.posDir = dir
	s.barsIn = 0
}

func (s *VWAPRevert) OnCandle(c candle.Candle) (int, float64, float64) {
	s.lastCandle = c
	v := s.vwap.Next(c)[0]
	r := s.rsi.Next(c)[0]
	a := s.atr.Next(c)[0]

	adxOK := true
	if s.adx != nil {
		adxOK = s.adx.Next(c)[0] < s.adxMax
	}

	rsiRising := s.hasPrev && r > s.prevRSI
	rsiFalling := s.hasPrev && r < s.prevRSI
	s.prevRSI = r
	s.hasPrev = true

	if s.vwap.IsIdle() || s.rsi.IsIdle() || s.atr.IsIdle() {
		return 0, 0, 0
	}

	// управление открытой позицией
	if s.posDir != 0 {
		s.barsIn++
		timeUp := s.timeStop > 0 && s.barsIn >= s.timeStop
		if s.posDir > 0 && (c.C >= v || r >= s.overbought || timeUp) {
			return 0, 0, 0
		}
		if s.posDir < 0 && (c.C <= v || r <= s.oversold || timeUp) {
			return 0, 0, 0
		}
		return s.posDir, 0, 0 // держим
	}

	// уровни: stopDist = atr_mult×ATR или фикс stop_pct; take = tp_r×stopDist
	// или фикс take_pct (0 = биржевого тейка нет, выход у VWAP)
	stopDist := s.atrMult * a
	if s.stopPct > 0 {
		stopDist = c.C * s.stopPct
	}
	takeDist := s.tpR * stopDist
	if s.takePct > 0 {
		takeDist = c.C * s.takePct
	}
	var takeL float64
	if takeDist > 0 {
		takeL = take(c.C, takeDist, 1)
	}

	// входы
	dev := s.devMult * a
	switch {
	case c.C < v-dev && r < s.oversold && rsiRising && adxOK:
		return 1, c.C - stopDist, takeL
	case c.C > v+dev && r > s.overbought && rsiFalling && adxOK && !s.longOnly:
		return -1, c.C + stopDist, take(c.C, takeDist, -1)
	}
	return 0, 0, 0
}
