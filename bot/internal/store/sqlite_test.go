package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"bot/internal/market"
)

func TestReadOnlyHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history #1.db")
	if _, err := OpenReadOnly(path); err == nil {
		t.Fatal("opened missing history")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("read created database")
	}
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ks := []market.Kline{{OpenTime: time.Now().UTC().Truncate(time.Millisecond), Open: 100, High: 101, Low: 99, Close: 100, Volume: 1}}
	if _, err := db.InsertKlines("A", "1h", ks); err != nil {
		t.Fatal(err)
	}
	ro, err := OpenReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	cs, err := ro.LoadKlines("A", "1h")
	if err != nil || len(cs) != 1 || cs[0].C != 100 {
		t.Fatalf("read: %v %v", cs, err)
	}
	if _, err := ro.InsertKlines("B", "1h", ks); err == nil {
		t.Fatal("read-only connection accepted write")
	}
}
