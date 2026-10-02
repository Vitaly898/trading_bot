// Package experiment is the common entry point for historical strategy experiments.
package experiment

import (
	"fmt"
	"time"

	"bot/internal/backtest"
	"bot/internal/candle"
	"bot/internal/config"
	"bot/internal/market"
	"bot/internal/store"
	"bot/internal/strategy"
	"bot/internal/trading"
)

// HistoryReader permits storage-independent experiments and synthetic tests.
type HistoryReader interface {
	LoadKlines(string, string) ([]candle.Candle, error)
	LoadFunding(string) ([]market.Funding, error)
}

func Load(reader HistoryReader, c *config.Config) ([]backtest.SymbolData, error) {
	syms := c.Symbols
	if len(syms) == 0 {
		syms = []string{c.Symbol}
	}
	var ds []backtest.SymbolData
	for _, sym := range syms {
		cs, err := reader.LoadKlines(sym, c.TF)
		if err != nil {
			return nil, fmt.Errorf("%s candles: %w", sym, err)
		}
		if len(cs) == 0 {
			return nil, fmt.Errorf("no candles for %s %s", sym, c.TF)
		}
		var fs []market.Funding
		// Funding also supplies strategy filters even when accounting is disabled.
		fs, err = reader.LoadFunding(sym)
		if err != nil {
			return nil, fmt.Errorf("%s funding: %w", sym, err)
		}
		ds = append(ds, backtest.SymbolData{Symbol: sym, Candles: cs, Funding: fs})
	}
	return ds, nil
}
func Open(c *config.Config) ([]backtest.SymbolData, error) {
	db, err := store.OpenReadOnly(c.DB)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return Load(db, c)
}
func EngineConfig(c *config.Config, start time.Time) backtest.Config {
	symbol := c.Symbol
	if len(c.Symbols) > 0 {
		symbol = "PORTFOLIO"
	}
	return backtest.Config{Symbol: symbol, TF: c.TF, StartEquity: c.Equity, RiskPct: c.Risk, TakerFeePct: c.Fee, SlippagePct: c.Slip, UseFunding: c.UseFunding(), StartTrading: start}
}

// Run constructs a fresh strategy per symbol, honors split/shared and returns configuration errors.
func Run(c *config.Config, ds []backtest.SymbolData, start time.Time) (*backtest.Report, error) {
	validated, err := c.WithParams(nil)
	if err != nil {
		return nil, err
	}
	c = validated
	if len(ds) == 0 {
		return nil, fmt.Errorf("empty experiment dataset")
	}
	expected := c.Symbols
	if len(expected) == 0 {
		expected = []string{c.Symbol}
	}
	if len(ds) != len(expected) {
		return nil, fmt.Errorf("dataset does not cover configured symbols")
	}
	allowed := map[string]bool{}
	for _, sym := range expected {
		allowed[sym] = true
	}
	strategies := map[string]trading.Strategy{}
	for _, sd := range ds {
		if !allowed[sd.Symbol] || strategies[sd.Symbol] != nil {
			return nil, fmt.Errorf("unexpected or duplicate dataset symbol %q", sd.Symbol)
		}
		st, err := strategy.New(c.Strategy.Name, strategy.Params(c.Strategy.Params))
		if err != nil {
			return nil, err
		}
		strategies[sd.Symbol] = st
	}
	cfg := EngineConfig(c, start)
	if len(c.Symbols) == 0 {
		if len(ds) != 1 {
			return nil, fmt.Errorf("single-symbol experiment requires one dataset")
		}
		cfg.Symbol = ds[0].Symbol
		return backtest.Run(cfg, strategies[ds[0].Symbol], ds[0].Candles, ds[0].Funding), nil
	}
	mk := func(sym string) backtest.Strategy { return strategies[sym] }
	if c.Mode == "split" {
		return backtest.RunSplit(cfg, mk, ds), nil
	}
	return backtest.RunPortfolio(backtest.PortfolioConfig{Config: cfg, MaxPositions: c.MaxPositions, MaxTotalRisk: c.MaxTotalRisk}, mk, ds), nil
}
