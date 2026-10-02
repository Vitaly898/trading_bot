package config

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// SweepFile — файл сетки параметров для sweep/walk-forward.
type SweepFile struct {
	Base string           `yaml:"base"`
	Top  int              `yaml:"top"`
	Grid map[string][]any `yaml:"grid"`
}

func LoadSweep(path string) (*SweepFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sf SweepFile
	if err := decode(raw, &sf); err != nil {
		return nil, fmt.Errorf("парсинг %s: %w", path, err)
	}
	if strings.TrimSpace(sf.Base) == "" {
		return nil, fmt.Errorf("%s: base is required", path)
	}
	if sf.Top < 0 {
		return nil, fmt.Errorf("%s: top must be nonnegative", path)
	}
	for key, values := range sf.Grid {
		if key == "" || len(values) == 0 {
			return nil, fmt.Errorf("%s: empty grid axis %q", path, key)
		}
	}
	if len(sf.Grid) == 0 {
		return nil, fmt.Errorf("%s: пустая grid", path)
	}
	return &sf, nil
}

// Cartesian — декартово произведение значений grid. Ключи сортируются для детерминизма.
func Cartesian(grid map[string][]any) []map[string]any {
	keys := make([]string, 0, len(grid))
	for k := range grid {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := []map[string]any{{}}
	for _, k := range keys {
		var next []map[string]any
		for _, c := range out {
			for _, v := range grid[k] {
				nc := make(map[string]any, len(c)+1)
				for ck, cv := range c {
					nc[ck] = cv
				}
				nc[k] = v
				next = append(next, nc)
			}
		}
		out = next
	}
	return out
}

// FmtParams — "k=v k=v" в отсортированном порядке.
func FmtParams(p map[string]any) string {
	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, p[k]))
	}
	return strings.Join(parts, " ")
}

// ValidateGrid checks all combinations against the active base strategy.
func ValidateGrid(base *Config, combinations []map[string]any) error {
	if len(combinations) == 0 {
		return fmt.Errorf("empty parameter grid")
	}
	for _, params := range combinations {
		if _, err := base.WithParams(params); err != nil {
			return fmt.Errorf("grid %s: %w", FmtParams(params), err)
		}
	}
	return nil
}
