package store

import (
	"time"

	"bot/internal/candle"
	"bot/internal/data"
)

// LoadKlines — свечи символа/ТФ из базы, отсортированные по времени.
func (d *DB) LoadKlines(symbol, tf string) ([]candle.Candle, error) {
	rows, err := d.conn.Query(`SELECT open_time, open, high, low, close, volume
		FROM klines WHERE symbol = ? AND tf = ? ORDER BY open_time`, symbol, tf)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []candle.Candle
	for rows.Next() {
		var ts int64
		var c candle.Candle
		if err := rows.Scan(&ts, &c.O, &c.H, &c.L, &c.C, &c.V); err != nil {
			return nil, err
		}
		c.Time = time.UnixMilli(ts).UTC()
		out = append(out, c)
	}
	return out, rows.Err()
}

// LoadFunding — записи funding rate символа, отсортированные по времени.
func (d *DB) LoadFunding(symbol string) ([]data.Funding, error) {
	rows, err := d.conn.Query(`SELECT calc_time, rate
		FROM funding WHERE symbol = ? ORDER BY calc_time`, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []data.Funding
	for rows.Next() {
		var ts int64
		var f data.Funding
		if err := rows.Scan(&ts, &f.Rate); err != nil {
			return nil, err
		}
		f.CalcTime = time.UnixMilli(ts).UTC()
		out = append(out, f)
	}
	return out, rows.Err()
}
