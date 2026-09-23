package database

import (
	"database/sql"
	"testing"
)

func TestInitializeAddsVersionColumnsToExistingDatabase(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	createLegacySchema(t, db)

	if _, err := db.Exec(`
		INSERT INTO inventory_items (
			id, name, upc, quantity, expiration_date, created_at, updated_at
		)
		VALUES (
			'legacy-item', 'Whole Milk', NULL, 1, NULL,
			'2026-09-22T14:00:00Z', '2026-09-22T14:00:00Z'
		)
	`); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`
		INSERT INTO pending_results (
			id, scan_id, association_id, suggested_name, suggested_upc,
			confidence, quantity, created_at
		)
		VALUES (
			'legacy-pending', 'scan-1', 'scan-1:item-1', 'Cheerios', NULL,
			0.65, 1, '2026-09-22T14:01:00Z'
		)
	`); err != nil {
		t.Fatal(err)
	}

	if err := Initialize(db); err != nil {
		t.Fatalf("failed to migrate existing database: %v", err)
	}

	assertLegacyRowVersion(t, db, "inventory_items", "legacy-item", 1)
	assertLegacyRowVersion(t, db, "pending_results", "legacy-pending", 1)

	var deletedAt sql.NullString
	if err := db.QueryRow(
		"SELECT deleted_at FROM inventory_items WHERE id = ?",
		"legacy-item",
	).Scan(&deletedAt); err != nil {
		t.Fatalf("failed to read migrated deleted_at: %v", err)
	}
	if deletedAt.Valid {
		t.Errorf("expected legacy inventory row deleted_at to be null")
	}

	var inventoryName string
	if err := db.QueryRow(
		"SELECT name FROM inventory_items WHERE id = ?",
		"legacy-item",
	).Scan(&inventoryName); err != nil {
		t.Fatalf("failed to read migrated inventory row: %v", err)
	}
	if inventoryName != "Whole Milk" {
		t.Errorf("expected migrated inventory name %q, got %q", "Whole Milk", inventoryName)
	}

	var pendingName string
	if err := db.QueryRow(
		"SELECT suggested_name FROM pending_results WHERE id = ?",
		"legacy-pending",
	).Scan(&pendingName); err != nil {
		t.Fatalf("failed to read migrated pending row: %v", err)
	}
	if pendingName != "Cheerios" {
		t.Errorf("expected migrated pending name %q, got %q", "Cheerios", pendingName)
	}

	if err := Initialize(db); err != nil {
		t.Fatalf("expected migration to be safe on repeated startup: %v", err)
	}
}

func createLegacySchema(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		CREATE TABLE inventory_items (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			upc TEXT,
			quantity INTEGER NOT NULL,
			expiration_date TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);

		CREATE TABLE pending_results (
			id TEXT PRIMARY KEY,
			scan_id TEXT NOT NULL,
			association_id TEXT NOT NULL,
			suggested_name TEXT NOT NULL,
			suggested_upc TEXT,
			confidence REAL NOT NULL,
			quantity INTEGER NOT NULL,
			created_at TEXT NOT NULL
		)
	`); err != nil {
		t.Fatal(err)
	}
}

func assertLegacyRowVersion(
	t *testing.T,
	db *sql.DB,
	table string,
	id string,
	expected int,
) {
	t.Helper()

	var version int
	query := "SELECT version FROM " + table + " WHERE id = ?"
	if err := db.QueryRow(query, id).Scan(&version); err != nil {
		t.Fatalf("failed to read migrated version from %s: %v", table, err)
	}

	if version != expected {
		t.Errorf(
			"expected migrated version %d in %s, got %d",
			expected,
			table,
			version,
		)
	}
}
