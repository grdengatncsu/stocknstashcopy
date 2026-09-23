package database

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/jae-white/stock-n-stash/server/models"
)

func TestAddPendingResult(t *testing.T) {
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

	result := models.PendingResult{
		ID:            "pending-1",
		ScanID:        "scan-1",
		AssociationID: "scan-1:item-1",
		SuggestedName: "Honey Nut Cheerios",
		SuggestedUPC:  &upc,
		Confidence:    0.62,
		Quantity:      1,
		CreatedAt:     "2026-09-20T22:00:00Z",
	}

	if err := AddPendingResult(db, result); err != nil {
		t.Fatalf("failed to add pending result: %v", err)
	}

	var name string
	var confidence float64

	err = db.QueryRow(`
		SELECT suggested_name, confidence
		FROM pending_results
		WHERE id = ?
	`, result.ID).Scan(&name, &confidence)

	if err != nil {
		t.Fatalf("failed to query pending result: %v", err)
	}

	if name != result.SuggestedName {
		t.Errorf("expected name %q, got %q", result.SuggestedName, name)
	}

	if confidence != result.Confidence {
		t.Errorf("expected confidence %f, got %f", result.Confidence, confidence)
	}
}

func TestGetPendingResultByID(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatal(err)
	}

	upc := "012345678905"

	expected := models.PendingResult{
		ID:            "pending-1",
		ScanID:        "scan-1",
		AssociationID: "scan-1:item-1",
		SuggestedName: "Cheerios",
		SuggestedUPC:  &upc,
		Confidence:    0.72,
		Quantity:      1,
		CreatedAt:     "2026-09-22T19:00:00Z",
	}

	if err := AddPendingResult(db, expected); err != nil {
		t.Fatal(err)
	}

	result, err := GetPendingResultByID(db, "pending-1")
	if err != nil {
		t.Fatal(err)
	}

	if result.ID != expected.ID {
		t.Errorf("expected ID %q, got %q", expected.ID, result.ID)
	}

	if result.SuggestedName != expected.SuggestedName {
		t.Errorf(
			"expected suggested name %q, got %q",
			expected.SuggestedName,
			result.SuggestedName,
		)
	}

	if result.Quantity != expected.Quantity {
		t.Errorf("expected quantity %d, got %d", expected.Quantity, result.Quantity)
	}
}

func TestGetPendingResultByIDNotFound(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatal(err)
	}

	_, err = GetPendingResultByID(db, "does-not-exist")

	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestDeletePendingResult(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatal(err)
	}

	result := models.PendingResult{
		ID:            "pending-1",
		ScanID:        "scan-1",
		AssociationID: "scan-1:item-1",
		SuggestedName: "Cheerios",
		Confidence:    0.72,
		Quantity:      1,
		CreatedAt:     "2026-09-22T19:00:00Z",
	}

	if err := AddPendingResult(db, result); err != nil {
		t.Fatal(err)
	}

	if err := DeletePendingResult(db, "pending-1"); err != nil {
		t.Fatal(err)
	}

	_, err = GetPendingResultByID(db, "pending-1")

	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows after delete, got %v", err)
	}
}

func TestResolvePendingResult(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatal(err)
	}

	pending := models.PendingResult{
		ID:            "pending-1",
		ScanID:        "scan-1",
		AssociationID: "scan-1:item-1",
		SuggestedName: "Cheerios",
		Confidence:    0.72,
		Quantity:      1,
		CreatedAt:     "2026-09-22T19:00:00Z",
	}

	if err := AddPendingResult(db, pending); err != nil {
		t.Fatal(err)
	}

	item := models.InventoryItem{
		ID:        "item-1",
		Name:      "Honey Nut Cheerios",
		Quantity:  1,
		CreatedAt: "2026-09-22T19:05:00Z",
		UpdatedAt: "2026-09-22T19:05:00Z",
	}

	if err := ResolvePendingResult(db, "pending-1", 1, item); err != nil {
		t.Fatal(err)
	}

	stored, err := GetInventoryItemByID(db, "item-1")
	if err != nil {
		t.Fatal(err)
	}

	if stored.Name != "Honey Nut Cheerios" {
		t.Errorf("expected inventory name %q, got %q", "Honey Nut Cheerios", stored.Name)
	}

	_, err = GetPendingResultByID(db, "pending-1")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected pending tombstone to be hidden, got %v", err)
	}

	var deletedAt sql.NullString
	var version int
	if err := db.QueryRow(
		"SELECT deleted_at, version FROM pending_results WHERE id = ?",
		pending.ID,
	).Scan(&deletedAt, &version); err != nil {
		t.Fatalf("expected pending tombstone to remain stored: %v", err)
	}
	if !deletedAt.Valid {
		t.Fatal("expected resolved pending result to have deleted_at")
	}
	if version != 2 {
		t.Errorf("expected pending tombstone version 2, got %d", version)
	}
}

func TestResolvePendingResultRollsBackWhenPendingMissing(t *testing.T) {
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
		Name:      "Cheerios",
		Quantity:  1,
		CreatedAt: "2026-09-22T19:05:00Z",
		UpdatedAt: "2026-09-22T19:05:00Z",
	}

	err = ResolvePendingResult(db, "does-not-exist", 1, item)

	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}

	_, err = GetInventoryItemByID(db, "item-1")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected inventory insert to be rolled back, got %v", err)
	}
}
