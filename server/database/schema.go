package database

import (
	"database/sql"
	"fmt"
)

// Initialize creates every table required by the API. IF NOT EXISTS makes this
// safe to run whenever the server starts without erasing existing data.
func Initialize(db *sql.DB) error {
	// Confirmed products are stored separately from results that need review.
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS inventory_items (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			upc TEXT,
			quantity INTEGER NOT NULL,
			expiration_date TEXT,
			version INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			deleted_at TEXT
		)
	`)

	if err != nil {
		return err
	}

	if err := ensureColumn(
		db,
		"inventory_items",
		"version",
		"INTEGER NOT NULL DEFAULT 1",
	); err != nil {
		return err
	}

	// Ensure the deleted_at column exists in the inventory_items table.
	if err := ensureColumn(
		db,
		"inventory_items",
		"deleted_at",
		"TEXT",
	); err != nil {
		return err
	}

	// Pending rows keep the model's suggestions until a user resolves them.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS pending_results (
			id TEXT PRIMARY KEY,
			scan_id TEXT NOT NULL,
			association_id TEXT NOT NULL,
			suggested_name TEXT NOT NULL,
			suggested_upc TEXT,
			confidence REAL NOT NULL,
			quantity INTEGER NOT NULL,
			version INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			deleted_at TEXT
		)
	`)
	if err != nil {
		return err
	}

	if err := ensureColumn(
		db,
		"pending_results",
		"version",
		"INTEGER NOT NULL DEFAULT 1",
	); err != nil {
		return err
	}

	if err := ensureColumn(
		db,
		"pending_results",
		"updated_at",
		"TEXT NOT NULL DEFAULT ''",
	); err != nil {
		return err
	}

	if _, err := db.Exec(`
		UPDATE pending_results
		SET updated_at = created_at
		WHERE updated_at = ''
	`); err != nil {
		return err
	}

	if err := ensureColumn(
		db,
		"pending_results",
		"deleted_at",
		"TEXT",
	); err != nil {
		return err
	}

	// A scan ID is recorded only after all of its items are written. This table
	// lets repeated network submissions return success without duplicating items.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS processed_scans (
			scan_id TEXT PRIMARY KEY,
			processed_at TEXT NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	return nil
}

func ensureColumn(
	db *sql.DB,
	table string,
	column string,
	definition string,
) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int

		if err := rows.Scan(
			&cid,
			&name,
			&columnType,
			&notNull,
			&defaultValue,
			&primaryKey,
		); err != nil {
			rows.Close()
			return err
		}

		if name == column {
			rows.Close()
			return nil
		}
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}

	if err := rows.Close(); err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf(
		"ALTER TABLE %s ADD COLUMN %s %s",
		table,
		column,
		definition,
	))
	return err
}
