// dbstats — проверка содержимого локальной базы: покрытие, разрывы, funding.
// Пример:
//
//	go run ./cmd/dbstats -symbol BTCUSDT -tfs 15m,1h,4h -db history.db
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"bot/internal/store"
)

func main() {
	symbol := flag.String("symbol", "BTCUSDT", "торговая пара")
	tfs := flag.String("tfs", "1h", "таймфреймы через запятую")
	dbPath := flag.String("db", "history.db", "путь к SQLite базе")
	flag.Parse()

	db, err := store.OpenReadOnly(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sym := strings.ToUpper(*symbol)
	fmt.Printf("=== %s @ %s ===\n", sym, *dbPath)

	for _, tf := range strings.Split(*tfs, ",") {
		tf = strings.TrimSpace(tf)
		st, err := db.KlineStats(sym, tf)
		if err != nil {
			log.Printf("[%s] %v", tf, err)
			continue
		}
		if st.Count == 0 {
			fmt.Printf("[%-4s] пусто\n", tf)
			continue
		}
		expected := int(st.Last.Sub(st.First)/st.Interval) + 1
		coverage := 100 * float64(st.Count) / float64(expected)
		fmt.Printf("[%-4s] свечей: %d | %s .. %s | разрывов: %d | покрытие: %.2f%%\n",
			tf, st.Count,
			st.First.Format("2006-01-02"), st.Last.Format("2006-01-02"),
			st.Gaps, coverage)
	}

	fs, err := db.FundingStats(sym)
	if err != nil {
		log.Fatal(err)
	}
	if fs.Count > 0 {
		fmt.Printf("[funding] записей: %d | %s .. %s | avg: %.4f%% | min: %.4f%% | max: %.4f%%\n",
			fs.Count,
			fs.First.Format("2006-01-02"), fs.Last.Format("2006-01-02"),
			fs.AvgRate*100, fs.MinRate*100, fs.MaxRate*100)
	} else {
		fmt.Println("[funding] пусто")
	}
}
