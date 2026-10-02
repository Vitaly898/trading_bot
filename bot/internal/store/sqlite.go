// Package store — локальное хранилище исторических данных в SQLite.
package store

import (
	"database/sql"
	"fmt"
	"time"

	"bot/internal/data"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, err
	}
	d := &DB{conn: conn}
	if err := d.migrate(); err != nil {
		conn.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) Close() error { return d.conn.Close() }

func (d *DB) migrate() error {
	_, err := d.conn.Exec(`
CREATE TABLE IF NOT EXISTS klines (
    symbol    TEXT NOT NULL,
    tf        TEXT NOT NULL,
    open_time INTEGER NOT NULL,  -- unix ms UTC
    open  REAL NOT NULL,
    high  REAL NOT NULL,
    low   REAL NOT NULL,
    close REAL NOT NULL,
    volume REAL NOT NULL,
    PRIMARY KEY (symbol, tf, open_time)
) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS funding (
    symbol    TEXT NOT NULL,
    calc_time INTEGER NOT NULL,  -- unix ms UTC
    rate REAL NOT NULL,
    PRIMARY KEY (symbol, calc_time)
) WITHOUT ROWID;
`)
	return err
}

func (d *DB) InsertKlines(symbol, tf string, ks []data.Kline) (int, error) {
	tx, err := d.conn.Begin()
	if err != nil {
		return 0, err
	}
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO klines
		(symbol, tf, open_time, open, high, low, close, volume)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	n := 0
	for _, k := range ks {
		res, err := stmt.Exec(symbol, tf, k.OpenTime.UnixMilli(),
			k.Open, k.High, k.Low, k.Close, k.Volume)
		if err != nil {
			stmt.Close()
			tx.Rollback()
			return n, err
		}
		if aff, _ := res.RowsAffected(); aff > 0 {
			n++
		}
	}
	stmt.Close()
	return n, tx.Commit()
}

func (d *DB) InsertFunding(symbol string, fs []data.Funding) (int, error) {
	tx, err := d.conn.Begin()
	if err != nil {
		return 0, err
	}
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO funding (symbol, calc_time, rate)
		VALUES (?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	n := 0
	for _, f := range fs {
		res, err := stmt.Exec(symbol, f.CalcTime.UnixMilli(), f.Rate)
		if err != nil {
			stmt.Close()
			tx.Rollback()
			return n, err
		}
		if aff, _ := res.RowsAffected(); aff > 0 {
			n++
		}
	}
	stmt.Close()
	return n, tx.Commit()
}

// KlineStats — сводка по таблице свечей.
type KlineStats struct {
	Symbol   string
	TF       string
	Count    int
	First    time.Time
	Last     time.Time
	Gaps     int // количество разрывов (пропущенных свечей)
	Interval time.Duration
}

func (d *DB) KlineStats(symbol, tf string) (*KlineStats, error) {
	st := &KlineStats{Symbol: symbol, TF: tf}
	row := d.conn.QueryRow(`SELECT COUNT(*), MIN(open_time), MAX(open_time)
		FROM klines WHERE symbol = ? AND tf = ?`, symbol, tf)
	var firstMs, lastMs sql.NullInt64
	if err := row.Scan(&st.Count, &firstMs, &lastMs); err != nil {
		return nil, err
	}
	if st.Count == 0 {
		return st, nil
	}
	st.First = time.UnixMilli(firstMs.Int64).UTC()
	st.Last = time.UnixMilli(lastMs.Int64).UTC()

	iv, err := tfDuration(tf)
	if err != nil {
		return nil, err
	}
	st.Interval = iv

	rows, err := d.conn.Query(`SELECT open_time FROM klines
		WHERE symbol = ? AND tf = ? ORDER BY open_time`, symbol, tf)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var prev int64
	firstRow := true
	expectedMs := iv.Milliseconds()
	for rows.Next() {
		var ts int64
		if err := rows.Scan(&ts); err != nil {
			return nil, err
		}
		if !firstRow && ts-prev != expectedMs {
			st.Gaps += int((ts - prev) / expectedMs) // грубая оценка пропусков
		}
		prev = ts
		firstRow = false
	}
	return st, rows.Err()
}

// FundingStats — сводка по funding rate.
type FundingStats struct {
	Symbol  string
	Count   int
	First   time.Time
	Last    time.Time
	AvgRate float64
	MinRate float64
	MaxRate float64
}

func (d *DB) FundingStats(symbol string) (*FundingStats, error) {
	st := &FundingStats{Symbol: symbol}
	row := d.conn.QueryRow(`SELECT COUNT(*), MIN(calc_time), MAX(calc_time),
		AVG(rate), MIN(rate), MAX(rate) FROM funding WHERE symbol = ?`, symbol)
	var firstMs, lastMs sql.NullInt64
	var avg, mn, mx sql.NullFloat64
	if err := row.Scan(&st.Count, &firstMs, &lastMs, &avg, &mn, &mx); err != nil {
		return nil, err
	}
	if st.Count > 0 {
		st.First = time.UnixMilli(firstMs.Int64).UTC()
		st.Last = time.UnixMilli(lastMs.Int64).UTC()
		st.AvgRate, st.MinRate, st.MaxRate = avg.Float64, mn.Float64, mx.Float64
	}
	return st, nil
}

func tfDuration(tf string) (time.Duration, error) {
	if len(tf) < 2 {
		return 0, fmt.Errorf("bad tf %q", tf)
	}
	n := 0
	if _, err := fmt.Sscanf(tf, "%d", &n); err != nil {
		return 0, err
	}
	unit := tf[len(tf)-1]
	switch unit {
	case 'm':
		return time.Duration(n) * time.Minute, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	}
	return 0, fmt.Errorf("unknown tf unit %q", tf)
}
