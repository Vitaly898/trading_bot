package strategy

import "fmt"

// Params — параметры стратегии из YAML-конфига.
// Все геттеры с дефолтами: отсутствующий ключ не роняет прогон.
type Params map[string]any

func (p Params) Int(key string, def int) int {
	v, ok := p[key]
	if !ok {
		return def
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return def
}

func (p Params) Float(key string, def float64) float64 {
	v, ok := p[key]
	if !ok {
		return def
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return def
}

func (p Params) Str(key, def string) string {
	if v, ok := p[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprint(v)
	}
	return def
}

func (p Params) Bool(key string, def bool) bool {
	if v, ok := p[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}
