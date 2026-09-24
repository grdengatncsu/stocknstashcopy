package database

import (
	"path/filepath"
	"testing"
)

func TestOpenConfiguresSQLiteForPiService(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if db.Stats().MaxOpenConnections != 1 {
		t.Errorf("expected one SQLite connection, got %d", db.Stats().MaxOpenConnections)
	}

	var busyTimeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatal(err)
	}
	if busyTimeout != 5000 {
		t.Errorf("expected 5000 ms busy timeout, got %d", busyTimeout)
	}

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if journalMode != "wal" {
		t.Errorf("expected WAL journal mode, got %q", journalMode)
	}
}
