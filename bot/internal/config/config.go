// Package config — YAML-конфиг прогона бэктеста.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config — полное описание одного прогона бэктеста.
type Config struct {
	Symbol  string  `yaml:"symbol"`
	Symbols []string `yaml:"symbols"` // портфельный режим (cmd/portfolio)
	MaxPositions int `yaml:"max_positions"` // лимит одновременных позиций (0 = без лимита)
	MaxTotalRisk float64 `yaml:"max_total_risk"` // бюджет риска портфеля, доля equity (0 = без лимита)
	Mode string `yaml:"mode"` // shared (общий капитал+лимиты, дефолт) | split (равные суб-портфели)
	Leverage int `yaml:"leverage"` // плечо на фьючерсах (0/1 = 1x; только live-режим)
	TF      string  `yaml:"tf"`
	DB      string  `yaml:"db"`
	Equity  float64 `yaml:"equity"`
	Risk    float64 `yaml:"risk"`
	Fee     float64 `yaml:"fee"`
	Slip    float64 `yaml:"slip"`
	Funding *bool   `yaml:"funding"` // по умолчанию true

	Strategy struct {
		Name   string         `yaml:"name"`
		Params map[string]any `yaml:"params"`
	} `yaml:"strategy"`
}

func defaults() Config {
	return Config{
		Symbol: "BTCUSDT",
		TF:     "1h",
		DB:     "history.db",
		Equity: 10000,
		Risk:   0.01,
		Fee:    0.0005,
		Slip:   0.0002,
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := defaults()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("парсинг %s: %w", path, err)
	}
	if cfg.Strategy.Name == "" {
		return nil, fmt.Errorf("%s: не задан strategy.name", path)
	}
	return &cfg, nil
}

func (c *Config) UseFunding() bool {
	return c.Funding == nil || *c.Funding
}
