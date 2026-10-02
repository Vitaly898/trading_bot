package strategy

import (
	"fmt"
	"math"
	"strings"
)

var parameterTypes = buildParameterTypes()

func fields(kind, keys string, into map[string]string) {
	for _, key := range strings.Fields(keys) {
		into[key] = kind
	}
}
func buildParameterTypes() map[string]map[string]string {
	trend := map[string]string{}
	mr := map[string]string{}
	ema := map[string]string{}
	regime := map[string]string{}
	fields("int", "fast slow atr_period adx_period ema_filter st_period", trend)
	fields("float", "adx_min atr_mult tp_r stop_pct take_pct funding_long_max funding_short_min st_mult", trend)
	fields("bool", "long_short", trend)
	fields("str", "ma_type exit", trend)
	fields("int", "rsi_period atr_period adx_period time_stop", mr)
	fields("float", "oversold overbought dev_mult atr_mult tp_r stop_pct take_pct adx_max", mr)
	fields("bool", "long_short", mr)
	fields("int", "fast slow atr_period", ema)
	fields("float", "atr_mult", ema)
	fields("int", "adx_period er_period min_bars", regime)
	fields("float", "adx_enter adx_exit er_enter er_exit", regime)
	for key, kind := range trend {
		regime["trend_"+key] = kind
	}
	for key, kind := range mr {
		regime["mr_"+key] = kind
	}
	return map[string]map[string]string{"trend": trend, "vwap_revert": mr, "ema_cross": ema, "regime_switch": regime}
}

// Validate rejects ignored keys, coercions and invalid indicator parameters before construction.
func Validate(name string, p Params) error {
	schema, ok := parameterTypes[name]
	if !ok {
		return fmt.Errorf("unknown strategy %q", name)
	}
	for key, value := range p {
		kind, ok := schema[key]
		if !ok {
			return fmt.Errorf("strategy %s: unknown parameter %q", name, key)
		}
		short := strings.TrimPrefix(strings.TrimPrefix(key, "trend_"), "mr_")
		switch kind {
		case "bool":
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("%s must be boolean", key)
			}
		case "str":
			s, ok := value.(string)
			if !ok {
				return fmt.Errorf("%s must be string", key)
			}
			if short == "ma_type" && !validMA(s) {
				return fmt.Errorf("invalid %s: %s", key, s)
			}
			if short == "exit" && s != "signal" && s != "supertrend" {
				return fmt.Errorf("invalid exit %q", s)
			}
		default:
			var n float64
			switch v := value.(type) {
			case int:
				n = float64(v)
			case int64:
				n = float64(v)
			case float64:
				n = v
			default:
				return fmt.Errorf("%s must be numeric", key)
			}
			if math.IsNaN(n) || math.IsInf(n, 0) {
				return fmt.Errorf("%s must be finite", key)
			}
			if kind == "int" && (n != math.Trunc(n) || n > float64(int(^uint(0)>>1))-1024) {
				return fmt.Errorf("%s must be an integer in range", key)
			}
			if kind == "int" {
				minimum := 2.0
				if short == "ema_filter" || short == "time_stop" || short == "min_bars" {
					minimum = 0
				}
				if short == "er_period" {
					minimum = 1
				}
				if n < minimum {
					return fmt.Errorf("%s must be >= %g", key, minimum)
				}
			}
			if short == "ema_filter" && n == 1 {
				return fmt.Errorf("ema_filter must be 0 or >= 2")
			}
			if short == "st_mult" && n <= 0 {
				return fmt.Errorf("st_mult must be positive")
			}
			if kind == "float" && short != "funding_long_max" && short != "funding_short_min" && n < 0 {
				return fmt.Errorf("%s must be nonnegative", key)
			}
			if (short == "stop_pct" || short == "take_pct") && n >= 1 {
				return fmt.Errorf("%s must be less than 1", key)
			}
			if (strings.HasPrefix(short, "er_") && kind == "float") && n > 1 {
				return fmt.Errorf("%s must be <= 1", key)
			}
			if (strings.HasPrefix(short, "adx_") && kind == "float" || short == "oversold" || short == "overbought") && n > 100 {
				return fmt.Errorf("%s must be <= 100", key)
			}
		}
	}
	if name == "regime_switch" {
		if err := Validate("trend", prefixed(p, "trend_")); err != nil {
			return err
		}
		if err := Validate("vwap_revert", prefixed(p, "mr_")); err != nil {
			return err
		}
		if p.Float("adx_exit", 20) > p.Float("adx_enter", 25) || p.Float("er_exit", .18) > p.Float("er_enter", .25) {
			return fmt.Errorf("classifier exit thresholds must not exceed entry thresholds")
		}
	}
	if name == "trend" || name == "ema_cross" {
		if p.Int("fast", 20) >= p.Int("slow", 50) {
			return fmt.Errorf("fast must be less than slow")
		}
	}
	if name == "vwap_revert" && p.Float("oversold", 30) >= p.Float("overbought", 70) {
		return fmt.Errorf("oversold must be less than overbought")
	}
	return nil
}

func validMA(s string) bool {
	switch s {
	case "ema", "sma", "smma", "wma", "hma", "vwma", "kama":
		return true
	}
	return false
}
