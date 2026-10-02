package binance

import "time"

type Kline struct {
	O            float64
	H            float64
	L            float64
	C            float64
	V            float64
	IsClosed     bool
	Symbol       string
	TimeStart    time.Time
	TimeReceived time.Time
}

func (kline *Kline) Open() float64 {
	return kline.O
}

func (kline *Kline) Close() float64 {
	return kline.C
}

func (kline *Kline) High() float64 {
	return kline.H
}

func (kline *Kline) Low() float64 {
	return kline.L
}

func (kline *Kline) Volume() float64 {
	return kline.V
}

func (kline *Kline) Timestamp() time.Time {
	return kline.TimeStart
}
