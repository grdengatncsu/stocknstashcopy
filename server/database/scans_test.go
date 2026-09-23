package database

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jae-white/stock-n-stash/server/models"
)

func TestProcessScanReportStoresConfirmedAndPendingItems(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatal(err)
	}

	report := models.ScanReport{
		ScanID: "scan-1",
		Items: []models.ScanReportItem{
			{
				AssociationID:  "scan-1:item-1",
				Name:           "Cheerios",
				Confidence:     0.95,
				Quantity:       1,
				RequiresReview: false,
			},
			{
				AssociationID:  "scan-1:item-2",
				Name:           "Coke Zero",
				Confidence:     0.60,
				Quantity:       1,
				RequiresReview: true,
			},
		},
	}

	alreadyProcessed, err := ProcessScanReport(
		db,
		report,
		"2026-09-22T23:45:00-04:00",
	)

	if err != nil {
		t.Fatal(err)
	}

	if alreadyProcessed {
		t.Fatal("expected first scan submission to be processed")
	}

	inventory, err := GetInventoryItems(db)
	if err != nil {
		t.Fatal(err)
	}

	if len(inventory) != 1 {
		t.Fatalf("expected 1 inventory item, got %d", len(inventory))
	}

	if inventory[0].Name != "Cheerios" {
		t.Errorf("expected inventory item %q, got %q", "Cheerios", inventory[0].Name)
	}
	assertValidUUID(t, inventory[0].ID)

	pending, err := GetPendingResults(db)
	if err != nil {
		t.Fatal(err)
	}

	if len(pending) != 1 {
		t.Fatalf("expected 1 pending result, got %d", len(pending))
	}

	if pending[0].SuggestedName != "Coke Zero" {
		t.Errorf("expected pending result %q, got %q", "Coke Zero", pending[0].SuggestedName)
	}
	assertValidUUID(t, pending[0].ID)

	processed, err := IsScanProcessed(db, "scan-1")
	if err != nil {
		t.Fatal(err)
	}

	if !processed {
		t.Fatal("expected scan-1 to be recorded as processed")
	}
}

func assertValidUUID(t *testing.T, id string) {
	t.Helper()

	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("expected UUID, got %q: %v", id, err)
	}

	if parsed.String() != id {
		t.Errorf("expected canonical UUID %q, got %q", parsed.String(), id)
	}
}

func TestProcessScanReportDuplicateDoesNotDuplicateItems(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	if err := Initialize(db); err != nil {
		t.Fatal(err)
	}

	report := models.ScanReport{
		ScanID: "scan-1",
		Items: []models.ScanReportItem{
			{
				AssociationID:  "scan-1:item-1",
				Name:           "Cheerios",
				Confidence:     0.95,
				Quantity:       1,
				RequiresReview: false,
			},
		},
	}

	firstDuplicate, err := ProcessScanReport(
		db,
		report,
		"2026-09-22T23:45:00-04:00",
	)
	if err != nil {
		t.Fatal(err)
	}

	if firstDuplicate {
		t.Fatal("expected first submission to be new")
	}

	secondDuplicate, err := ProcessScanReport(
		db,
		report,
		"2026-09-22T23:46:00-04:00",
	)
	if err != nil {
		t.Fatal(err)
	}

	if !secondDuplicate {
		t.Fatal("expected second submission to be detected as duplicate")
	}

	inventory, err := GetInventoryItems(db)
	if err != nil {
		t.Fatal(err)
	}

	if len(inventory) != 1 {
		t.Fatalf(
			"expected duplicate scan not to add another item; got %d inventory items",
			len(inventory),
		)
	}
}
