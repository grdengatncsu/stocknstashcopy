package database

import (
	"database/sql"

	// Register the pure-Go SQLite driver under the name "sqlite". The blank
	// import is intentional: database/sql discovers drivers through init code.
	_ "modernc.org/sqlite"
)

// Open creates a SQLite connection pool and verifies that the database can be
// reached before returning it to the caller.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// The Pi service has one low-volume write path. A single connection avoids
	// SQLite lock contention and also guarantees that connection-local PRAGMAs
	// below apply to every query made by this process.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	pragmas := []string{
		"PRAGMA busy_timeout = 5000",
		"PRAGMA foreign_keys = ON",
		// WAL lets readers continue while the service commits a scan. SQLite
		// automatically keeps in-memory databases in their supported mode.
		"PRAGMA journal_mode = WAL",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, err
		}
	}

	return db, nil
}
