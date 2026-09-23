package database

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/jae-white/stock-n-stash/server/models"
)

func TestAddInventoryItem(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	// SQLite :memory: databases are per connection.
	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}

	upc := "012345678901"
	expiration := "2026-10-15"

	item := models.InventoryItem{
		ID:             "item-1",
		Name:           "Honey Nut Cheerios",
		UPC:            &upc,
		Quantity:       1,
		ExpirationDate: &expiration,
		CreatedAt:      "2026-09-20T18:00:00-04:00",
		UpdatedAt:      "2026-09-20T18:00:00-04:00",
	}

	if err := AddInventoryItem(db, item); err != nil {
		t.Fatalf("failed to add inventory item: %v", err)
	}

	var name string
	var quantity int

	err = db.QueryRow(
		"SELECT name, quantity FROM inventory_items WHERE id = ?",
		item.ID,
	).Scan(&name, &quantity)

	if err != nil {
		t.Fatalf("failed to query inserted item: %v", err)
	}

	if name != item.Name {
		t.Errorf("expected name %q, got %q", item.Name, name)
	}

	if quantity != item.Quantity {
		t.Errorf("expected quantity %d, got %d", item.Quantity, quantity)
	}
}

func TestGetInventoryItems(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}

	upc := "012345678901"

	first := models.InventoryItem{
		ID:        "item-1",
		Name:      "Honey Nut Cheerios",
		UPC:       &upc,
		Quantity:  1,
		CreatedAt: "2026-09-20T18:00:00-04:00",
		UpdatedAt: "2026-09-20T18:00:00-04:00",
	}

	second := models.InventoryItem{
		ID:        "item-2",
		Name:      "Milk",
		UPC:       nil,
		Quantity:  2,
		CreatedAt: "2026-09-20T18:01:00-04:00",
		UpdatedAt: "2026-09-20T18:01:00-04:00",
	}

	if err := AddInventoryItem(db, first); err != nil {
		t.Fatalf("failed to add first inventory item: %v", err)
	}

	if err := AddInventoryItem(db, second); err != nil {
		t.Fatalf("failed to add second inventory item: %v", err)
	}

	items, err := GetInventoryItems(db)
	if err != nil {
		t.Fatalf("failed to get inventory items: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 inventory items, got %d", len(items))
	}

	if items[0].Name != first.Name {
		t.Errorf("expected first item name %q, got %q", first.Name, items[0].Name)
	}

	if items[1].Name != second.Name {
		t.Errorf("expected second item name %q, got %q", second.Name, items[1].Name)
	}

	if items[1].UPC != nil {
		t.Errorf("expected second item UPC to be nil, got %v", *items[1].UPC)
	}
}

func TestUpdateInventoryItem(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}

	original := models.InventoryItem{
		ID:        "item-1",
		Name:      "Milk",
		Quantity:  1,
		Version:   1,
		CreatedAt: "2026-09-20T18:00:00-04:00",
		UpdatedAt: "2026-09-20T18:00:00-04:00",
	}

	if err := AddInventoryItem(db, original); err != nil {
		t.Fatalf("failed to add inventory item: %v", err)
	}

	updated := original
	updated.Name = "Whole Milk"
	updated.Quantity = 2
	updated.UpdatedAt = "2026-09-20T19:00:00-04:00"

	if err := UpdateInventoryItem(db, updated); err != nil {
		t.Fatalf("failed to update inventory item: %v", err)
	}

	var name string
	var quantity int
	var createdAt string
	var updatedAt string

	err = db.QueryRow(`
		SELECT name, quantity, created_at, updated_at
		FROM inventory_items
		WHERE id = ?
	`, original.ID).Scan(
		&name,
		&quantity,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		t.Fatalf("failed to query updated item: %v", err)
	}

	if name != "Whole Milk" {
		t.Errorf("expected name %q, got %q", "Whole Milk", name)
	}

	if quantity != 2 {
		t.Errorf("expected quantity 2, got %d", quantity)
	}

	if createdAt != original.CreatedAt {
		t.Errorf("created_at changed: expected %q, got %q", original.CreatedAt, createdAt)
	}

	if updatedAt != updated.UpdatedAt {
		t.Errorf("expected updated_at %q, got %q", updated.UpdatedAt, updatedAt)
	}
}

func TestUpdateInventoryItemMissingID(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}

	item := models.InventoryItem{
		ID:        "does-not-exist",
		Name:      "Milk",
		Quantity:  1,
		Version:   1,
		CreatedAt: "2026-09-20T18:00:00-04:00",
		UpdatedAt: "2026-09-20T19:00:00-04:00",
	}

	if err := UpdateInventoryItem(db, item); err == nil {
		t.Fatal("expected error when updating nonexistent inventory item")
	}
}

func TestUpdateInventoryItemRejectsStaleVersion(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatal(err)
	}

	original := models.InventoryItem{
		ID:        "item-1",
		Name:      "Whole Milk",
		Quantity:  1,
		Version:   1,
		CreatedAt: "2026-09-23T14:00:00Z",
		UpdatedAt: "2026-09-23T14:00:00Z",
	}
	if err := AddInventoryItem(db, original); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(
		"UPDATE inventory_items SET version = 2 WHERE id = ?",
		original.ID,
	); err != nil {
		t.Fatal(err)
	}

	stale := original
	stale.Quantity = 9
	stale.UpdatedAt = "2026-09-23T15:00:00Z"

	err = UpdateInventoryItem(db, stale)
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}

	stored, err := GetInventoryItemByID(db, original.ID)
	if err != nil {
		t.Fatal(err)
	}

	if stored.Quantity != original.Quantity {
		t.Errorf(
			"expected stale update not to change quantity; got %d",
			stored.Quantity,
		)
	}
	if stored.Version != 2 {
		t.Errorf("expected stored version 2, got %d", stored.Version)
	}
}

func TestGetInventoryItemByID(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}

	item := models.InventoryItem{
		ID:        "item-1",
		Name:      "Milk",
		Quantity:  2,
		CreatedAt: "2026-09-20T18:00:00-04:00",
		UpdatedAt: "2026-09-20T18:00:00-04:00",
	}

	if err := AddInventoryItem(db, item); err != nil {
		t.Fatalf("failed to add inventory item: %v", err)
	}

	got, err := GetInventoryItemByID(db, item.ID)
	if err != nil {
		t.Fatalf("failed to get inventory item: %v", err)
	}

	if got.ID != item.ID {
		t.Errorf("expected ID %q, got %q", item.ID, got.ID)
	}

	if got.Name != item.Name {
		t.Errorf("expected name %q, got %q", item.Name, got.Name)
	}

	if got.Quantity != item.Quantity {
		t.Errorf("expected quantity %d, got %d", item.Quantity, got.Quantity)
	}
}

func TestGetInventoryItemByIDNotFound(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}

	_, err = GetInventoryItemByID(db, "does-not-exist")
	if err == nil {
		t.Fatal("expected error for missing inventory item")
	}

	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestDeleteInventoryItem(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}

	item := models.InventoryItem{
		ID:        "item-1",
		Name:      "Milk",
		Quantity:  1,
		CreatedAt: "2026-09-20T18:00:00-04:00",
		UpdatedAt: "2026-09-20T18:00:00-04:00",
	}

	if err := AddInventoryItem(db, item); err != nil {
		t.Fatalf("failed to add inventory item: %v", err)
	}

	deletedAt := "2026-09-23T15:00:00Z"
	if err := DeleteInventoryItem(db, item.ID, 1, deletedAt); err != nil {
		t.Fatalf("failed to delete inventory item: %v", err)
	}

	var storedDeletedAt sql.NullString
	var storedVersion int
	if err := db.QueryRow(`
		SELECT deleted_at, version
		FROM inventory_items
		WHERE id = ?
	`, item.ID).Scan(&storedDeletedAt, &storedVersion); err != nil {
		t.Fatalf("expected tombstone row to remain: %v", err)
	}
	if !storedDeletedAt.Valid || storedDeletedAt.String != deletedAt {
		t.Errorf("expected deleted_at %q, got %v", deletedAt, storedDeletedAt)
	}
	if storedVersion != 2 {
		t.Errorf("expected tombstone version 2, got %d", storedVersion)
	}

	_, err = GetInventoryItemByID(db, item.ID)
	if err == nil {
		t.Fatal("expected deleted item to be missing")
	}

	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows after delete, got %v", err)
	}

	items, err := GetInventoryItems(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected normal inventory list to hide tombstone")
	}
}

func TestDeleteInventoryItemMissingID(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}

	if err := DeleteInventoryItem(
		db,
		"does-not-exist",
		1,
		"2026-09-23T15:00:00Z",
	); err == nil {
		t.Fatal("expected error when deleting nonexistent inventory item")
	}
}

func TestDeleteInventoryItemRejectsStaleVersion(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatal(err)
	}

	item := models.InventoryItem{
		ID:        "item-1",
		Name:      "Whole Milk",
		Quantity:  1,
		CreatedAt: "2026-09-23T14:00:00Z",
		UpdatedAt: "2026-09-23T14:00:00Z",
	}
	if err := AddInventoryItem(db, item); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(
		"UPDATE inventory_items SET version = 2 WHERE id = ?",
		item.ID,
	); err != nil {
		t.Fatal(err)
	}

	err = DeleteInventoryItem(db, item.ID, 1, "2026-09-23T15:00:00Z")
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}

	var deletedAt sql.NullString
	var version int
	if err := db.QueryRow(
		"SELECT deleted_at, version FROM inventory_items WHERE id = ?",
		item.ID,
	).Scan(&deletedAt, &version); err != nil {
		t.Fatal(err)
	}
	if deletedAt.Valid {
		t.Errorf("expected stale delete not to set deleted_at")
	}
	if version != 2 {
		t.Errorf("expected stale delete to keep version 2, got %d", version)
	}
}
