// Package backtest — движок бэктеста: прогон стратегии по истории
// с учётом комиссий, проскальзывания и funding rate.
package backtest

import (
	"time"

	"bot/internal/candle"
)

// Strategy — интерфейс стратегии. Вызывается на каждой закрытой свече.
type Strategy interface {
	// OnCandle возвращает целевое направление: +1 long, -1 short, 0 — вне рынка.
	// stopPrice — цена стопа для НОВОЙ позиции (0 — без стопа).
	// takePrice — цена тейка для НОВОЙ позиции (0 — без тейка).
	// Если направление совпадает с текущим, stopPrice/takePrice игнорируются
	// (уровни не двигаются; трейлинг — TODO).
	OnCandle(c candle.Candle) (dir int, stopPrice, takePrice float64)
}

// PositionAware — опциональный интерфейс стратегии: движок сообщает
// фактическое состояние позиции после каждого открытия/закрытия.
// dir: +1, -1 или 0. exitReason: причина закрытия ("stop","take","signal","end")
// — пустая при открытии. Нужен для выходов «в позиции» и cooldown после стопа.
type PositionAware interface {
	OnPositionChange(dir int, exitReason string)
}

// FundingAware — опциональный интерфейс стратегии: движок передаёт
// каждое funding-событие по мере его наступления (без заглядывания вперёд).
// Стратегия может строить MM-фильтры (вето по экстремальному funding и т.п.)
type FundingAware interface {
	OnFunding(rate float64, t time.Time)
}

// DiagnosticsProvider — опциональный интерфейс: стратегия отдаёт значения
// своих индикаторов (для логирования контекста открытия сделки).
type DiagnosticsProvider interface {
	Diagnostics() map[string]any
}

// Config — параметры прогона.
type Config struct {
	Symbol string
	TF     string

	StartEquity float64 // стартовый капитал, USDT
	RiskPct     float64 // доля капитала в риске на сделку (0.01 = 1%)

	TakerFeePct float64 // комиссия тейкера, доля (0.0005 = 0.05%)
	SlippagePct float64 // проскальзывание на вход/выход, доля

	UseFunding bool // применять funding rate к открытой позиции

	// StartTrading — дата начала торговли (walk-forward). Свечи до неё идут
	// только на прогрев индикаторов и funding-состояния. Нулевое = с начала.
	StartTrading time.Time
}

// Trade — завершённая сделка.
type Trade struct {
	Symbol     string
	Dir        int
	EntryTime  time.Time
	ExitTime   time.Time
	EntryPrice float64
	ExitPrice  float64
	Qty        float64
	PnL        float64 // чистый P&L сделки (после комиссий и funding)
	Fees       float64
	Funding    float64 // суммарный funding по сделке (отрицательный = платили)
	ExitReason string  // signal | stop | take | end
}

// Report — итог прогона.
type Report struct {
	Config

	StartTime time.Time
	EndTime   time.Time

	FinalEquity float64
	TotalReturn float64 // доля
	MaxDrawdown float64 // доля (положительное число)

	Trades      []Trade
	Wins        int
	Losses      int
	WinRate     float64
	ProfitFactor float64
	Expectancy  float64 // средний PnL на сделку, USDT
	AvgWin      float64
	AvgLoss     float64 // отрицательное

	TotalFees    float64
	TotalFunding float64

	ExposurePct float64 // доля времени в рынке
	Sharpe      float64 // по доходностям свечей, годовая

	ExitReasons map[string]int // статистика причин выхода

	// Equity — кривая капитала (mark-to-market) и её метки времени.
	Equity      []float64
	EquityTimes []time.Time
}
