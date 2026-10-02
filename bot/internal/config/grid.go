package config

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
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
	if err := yaml.Unmarshal(raw, &sf); err != nil {
		return nil, fmt.Errorf("парсинг %s: %w", path, err)
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
