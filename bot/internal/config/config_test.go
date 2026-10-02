package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bot/internal/strategy"
)

func fixture(t *testing.T, s string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte(s), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}
func TestStrictConfig(t *testing.T) {
	good := "strategy:\n  name: trend\n"
	for _, s := range []string{good + "typo: 1\n", good + "risk: 0\n", good + "risk: null\n", good + "risk: 0.01\nrisk: 0.02\n", "strategy: {name: trend, prams: {}}\n", good + "equity: .nan\n", good + "mode: wrong\n", good + "tf: 0h\n", good + "symbols: [ETHUSDT, ethusdt]\n", good + "---\nstrategy: {name: trend}\n", "strategy: {name: trend, params: {fast: 1}}\n", "strategy: {name: trend, params: {fast: 2.5}}\n", "strategy: {name: trend, params: {long_short: 'true'}}\n", "strategy: {name: trend, params: {frist: 10}}\n", "strategy: {name: trend, params: {st_mult: 0}}\n"} {
		if _, err := Load(fixture(t, s)); err == nil {
			t.Errorf("accepted invalid config: %s", s)
		}
	}
	c, err := Load(fixture(t, good+"symbols: [ethusdt, solusdt]\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Mode != "shared" || c.Symbols[0] != "ETHUSDT" {
		t.Fatal(c)
	}
	trial, err := c.WithParams(map[string]any{"fast": 12})
	if err != nil {
		t.Fatal(err)
	}
	trial.Symbols[0] = "OTHER"
	if c.Strategy.Params["fast"] != nil || c.Symbols[0] != "ETHUSDT" {
		t.Fatal("trial mutated base")
	}
}
func TestRepositoryConfigurations(t *testing.T) {
	// Config paths are intentionally relative to bot/, as in CLI invocations.
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(root, "configs", "*.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"sweeps", "walkforward"} {
		more, _ := filepath.Glob(filepath.Join(root, "configs", dir, "*.yaml"))
		files = append(files, more...)
	}
	if len(files) == 0 {
		t.Fatal("missing repository fixtures")
	}
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), "strategy:") {
				c, err := Load(path)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := strategy.New(c.Strategy.Name, strategy.Params(c.Strategy.Params)); err != nil {
					t.Fatal(err)
				}
				return
			}
			if strings.Contains(string(raw), "train_months:") {
				w, err := LoadWalkForward(path)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := Load(filepath.Join(root, w.Base)); err != nil {
					t.Fatal(err)
				}
				return
			}
			sf, err := LoadSweep(path)
			if err != nil {
				t.Fatal(err)
			}
			base, err := Load(filepath.Join(root, sf.Base))
			if err != nil {
				t.Fatal(err)
			}
			for _, params := range Cartesian(sf.Grid) {
				if _, err := base.WithParams(params); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
func TestExperimentFilesRejectInvalidWindowsAndGrid(t *testing.T) {
	for _, s := range []string{"base: x\ngrid: {fast: []}\n", "base: x\ngrid: {fast: [10]}\ntypo: true\n", "base: x\ngrid: {fast: [10]}\ntop: -1\n"} {
		if _, err := LoadSweep(fixture(t, s)); err == nil {
			t.Fatal("invalid grid accepted")
		}
	}
	for _, s := range []string{"mode: fixed\nbase: x\ntrain_months: 12\ntest_months: 0\n", "mode: fixed\nbase: x\ntrain_months: 12\ntest_months: 3\nstep_months: 1\n", "mode: optimize\nbase: x\ntrain_months: 12\ntest_months: 3\n", "mode: fixed\nbase: x\ntrain_months: 12\ntest_months: 3\nmetric: invalid\n"} {
		if _, err := LoadWalkForward(fixture(t, s)); err == nil {
			t.Fatal("invalid windows accepted")
		}
	}
}
