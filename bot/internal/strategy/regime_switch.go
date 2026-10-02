package strategy

import (
	"time"

	"bot/internal/candle"
)

// RegimeSwitch — мета-стратегия: TREND → trend-модуль, FLAT → mean-reversion модуль.
// Смена режима при открытой позиции другого модуля = немедленное закрытие.
//
// Параметры (yaml params):
//   классификатор: adx_enter/adx_exit/er_enter/er_exit/er_period/min_bars
//   trend_*: параметры trend (trend_ma_type, trend_fast, trend_slow, trend_adx_min, ...)
//   mr_*:    параметры vwap_revert (mr_dev_mult, mr_oversold, mr_time_stop, ...)
type RegimeSwitch struct {
	clf   *RegimeClassifier
	trend *Trend
	mr    *VWAPRevert

	regime    int
	posDir    int
	posFrom   string // "trend" | "mr"
	nextOwner string
}

func NewRegimeSwitch(p Params) (*RegimeSwitch, error) {
	trend, err := NewTrend(prefixed(p, "trend_"))
	if err != nil {
		return nil, err
	}
	mr, err := NewVWAPRevert(prefixed(p, "mr_"))
	if err != nil {
		return nil, err
	}
	return &RegimeSwitch{
		clf:    NewRegimeClassifier(p),
		trend:  trend,
		mr:     mr,
		regime: RegimeFlat,
	}, nil
}

// prefixed — достаёт под-параметры модуля: trend_fast → fast.
func prefixed(p Params, prefix string) Params {
	out := Params{}
	for k, v := range p {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			out[k[len(prefix):]] = v
		}
	}
	return out
}

func (s *RegimeSwitch) OnPositionChange(dir int, _ string) {
	if dir != 0 {
		s.posDir = dir
		s.posFrom = s.nextOwner
		if s.posFrom == "mr" {
			s.mr.OnPositionChange(dir, "")
		} else {
			s.trend.OnPositionChange(dir, "")
		}
		return
	}
	if s.posFrom == "mr" {
		s.mr.OnPositionChange(0, "")
	} else {
		s.trend.OnPositionChange(0, "")
	}
	s.posDir = 0
	s.posFrom = ""
}

func (s *RegimeSwitch) OnFunding(rate float64, t time.Time) {
	s.trend.OnFunding(rate, t)
}

// Diagnostics — режим + значения индикаторов активного модуля + детекторы.
func (s *RegimeSwitch) Diagnostics() map[string]any {
	d := map[string]any{}
	if s.regime == RegimeTrend {
		d["regime"] = "TREND"
		for k, v := range s.trend.Diagnostics() {
			d[k] = v
		}
	} else {
		d["regime"] = "FLAT"
		for k, v := range s.mr.Diagnostics() {
			d[k] = v
		}
	}
	d["cls_adx"] = round2f(s.clf.lastAdx)
	d["cls_er"] = round4f(s.clf.lastER)
	return d
}

func (s *RegimeSwitch) OnCandle(c candle.Candle) (int, float64, float64) {
	s.regime = s.clf.Update(c)

	// оба модуля видят все свечи — индикаторы не протухают между режимами
	td, ts, tt := s.trend.OnCandle(c)
	md, ms, mt := s.mr.OnCandle(c)

	// смена режима при открытой позиции чужого модуля → закрытие
	if s.posDir != 0 {
		if s.posFrom == "trend" && s.regime != RegimeTrend {
			return 0, 0, 0
		}
		if s.posFrom == "mr" && s.regime != RegimeFlat {
			return 0, 0, 0
		}
	}

	s.nextOwner = ""
	if s.regime == RegimeTrend {
		if td != 0 {
			s.nextOwner = "trend"
		}
		return td, ts, tt
	}
	if md != 0 {
		s.nextOwner = "mr"
	}
	return md, ms, mt
}

