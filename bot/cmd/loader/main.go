// loader — скачивает историю (klines + funding) с data.binance.vision в SQLite.
// Пример:
//   go run ./cmd/loader -symbol BTCUSDT -tfs 15m,1h,4h -from 2024-01 -to 2025-09 -db history.db
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"bot/internal/data"
	"bot/internal/store"
)

func parseMonth(s string) (data.Month, error) {
	t, err := time.Parse("2006-01", s)
	if err != nil {
		return data.Month{}, fmt.Errorf("месяц в формате YYYY-MM: %w", err)
	}
	return data.Month{t.Year(), t.Month()}, nil
}

func main() {
	symbol := flag.String("symbol", "BTCUSDT", "торговая пара")
	tfs := flag.String("tfs", "1h", "таймфреймы через запятую: 15m,1h,4h,1d")
	from := flag.String("from", "2024-01", "начальный месяц YYYY-MM")
	to := flag.String("to", "", "конечный месяц YYYY-MM (по умолчанию — предыдущий)")
	dbPath := flag.String("db", "history.db", "путь к SQLite базе")
	withFunding := flag.Bool("funding", true, "грузить funding rate")
	flag.Parse()

	fromM, err := parseMonth(*from)
	if err != nil {
		log.Fatal(err)
	}
	var toM data.Month
	if *to == "" {
		prev := time.Now().UTC().AddDate(0, -1, 0) // текущий месяц может быть неполным
		toM = data.Month{prev.Year(), prev.Month()}
	} else {
		toM, err = parseMonth(*to)
		if err != nil {
			log.Fatal(err)
		}
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	startMs := time.Date(fromM.Year, fromM.Month, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
	endMs := time.Date(toM.Year, toM.Month, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0).UnixMilli()
	log.Printf("Загрузка %s: %s .. %s → %s (REST fapi)",
		*symbol,
		time.UnixMilli(startMs).UTC().Format("2006-01-02"),
		time.UnixMilli(endMs).UTC().Format("2006-01-02"), *dbPath)

	sym := strings.ToUpper(*symbol)
	for _, tf := range strings.Split(*tfs, ",") {
		tf = strings.TrimSpace(tf)
		ks, err := data.FetchKlinesREST(sym, tf, startMs, endMs)
		if err != nil {
			log.Printf("  [%s] klines: ошибка: %v", tf, err)
			continue
		}
		n, err := db.InsertKlines(sym, tf, ks)
		if err != nil {
			log.Fatalf("  [%s] insert: %v", tf, err)
		}
		log.Printf("[%s] загружено свечей: %d (получено %d)", tf, n, len(ks))
	}

	if *withFunding {
		fs, err := data.FetchFundingREST(sym, startMs, endMs)
		if err != nil {
			log.Printf("  [funding] ошибка: %v", err)
		} else {
			n, err := db.InsertFunding(sym, fs)
			if err != nil {
				log.Fatalf("  [funding] insert: %v", err)
			}
			log.Printf("[funding] загружено записей: %d (получено %d)", n, len(fs))
		}
	}
	log.Println("Готово.")
}
