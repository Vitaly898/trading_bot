// Package trading defines execution-independent strategy contracts.
package trading

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

// PositionState contains trading state only; indicators are warmed separately.
type PositionState struct {
	Dir    int    `json:"dir"`
	Owner  string `json:"owner,omitempty"`
	BarsIn int    `json:"bars_in,omitempty"`
}

// PositionRestorer allows live restarts without resetting ownership or time stops.
type PositionRestorer interface {
	PositionState() PositionState
	EntryState(dir int) PositionState
	RestorePosition(PositionState) error
}
