package live

import (
	"context"
	"time"
)

// Position — открытая позиция (единый вид для local и testnet).
type Position struct {
	Symbol        string    `json:"symbol"`
	Dir           int       `json:"dir"` // +1 long, -1 short
	Qty           float64   `json:"qty"`
	EntryPrice    float64   `json:"entry_price"`
	Stop          float64   `json:"stop"`
	Take          float64   `json:"take"`
	EntryTime     time.Time `json:"entry_time"`
	EntryOrderID  int64     `json:"entry_order_id,omitempty"`
	ClientOrderID string    `json:"client_order_id,omitempty"`
	PendingClose  string    `json:"pending_close,omitempty"`
	StopOrderID   int64     `json:"stop_order_id,omitempty"`
	TakeOrderID   int64     `json:"take_order_id,omitempty"`
	Unconfirmed   bool      `json:"unconfirmed,omitempty"`
}

// Executor — исполнение ордеров.
type Executor interface {
	// Open — вход маркетом + стоп/тейк. Возвращает фактическую цену входа.
	Open(ctx context.Context, sym string, dir int, qty, stop, take, refPrice float64) (float64, error)
	// Close — закрытие позиции по символу маркетом (+ отмена SL/TP ордеров).
	Close(ctx context.Context, sym string, refPrice float64) (float64, error)
	// Equity — текущий капитал в USDT.
	Equity(ctx context.Context) (float64, error)
	// Positions — локальный кэш открытых позиций.
	Positions() map[string]*Position
	// Name — режим.
	Name() string
}
