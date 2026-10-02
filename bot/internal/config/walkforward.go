package config

import (
	"fmt"
	"os"
)

type WalkForward struct {
	Mode        string `yaml:"mode"`
	Base        string `yaml:"base"`
	Grid        string `yaml:"grid"`
	TrainMonths int    `yaml:"train_months"`
	TestMonths  int    `yaml:"test_months"`
	StepMonths  int    `yaml:"step_months"`
	Metric      string `yaml:"metric"`
}

func LoadWalkForward(path string) (*WalkForward, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var w WalkForward
	if err := decode(raw, &w); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if w.Metric == "" {
		w.Metric = "sharpe"
	}
	if w.StepMonths == 0 {
		w.StepMonths = w.TestMonths
	}
	if (w.Mode != "fixed" && w.Mode != "optimize") || w.Base == "" || w.TrainMonths <= 0 || w.TestMonths <= 0 || w.StepMonths <= 0 {
		return nil, fmt.Errorf("%s: invalid mode, base or window sizes", path)
	}
	if w.StepMonths < w.TestMonths {
		return nil, fmt.Errorf("%s: overlapping OOS windows cannot be compounded", path)
	}
	if w.Mode == "optimize" && w.Grid == "" {
		return nil, fmt.Errorf("%s: optimize requires grid", path)
	}
	if w.Metric != "sharpe" && w.Metric != "return" && w.Metric != "pf" {
		return nil, fmt.Errorf("%s: unknown metric %q", path, w.Metric)
	}
	return &w, nil
}
