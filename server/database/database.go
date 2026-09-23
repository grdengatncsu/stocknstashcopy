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

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
