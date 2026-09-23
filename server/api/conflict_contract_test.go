package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jae-white/stock-n-stash/server/database"
	"github.com/jae-white/stock-n-stash/server/models"
)

func TestUpdateInventoryHandlerRequiresExpectedVersion(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		expectedMessage string
	}{
		{
			name:            "missing expected version",
			body:            `{"quantity": 2}`,
			expectedMessage: "expected_version is required",
		},
		{
			name:            "null expected version",
			body:            `{"expected_version": null, "quantity": 2}`,
			expectedMessage: "expected_version is required",
		},
		{
			name:            "zero expected version",
			body:            `{"expected_version": 0, "quantity": 2}`,
			expectedMessage: "expected_version must be greater than zero",
		},
		{
			name:            "negative expected version",
			body:            `{"expected_version": -1, "quantity": 2}`,
			expectedMessage: "expected_version must be greater than zero",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupPendingAPITestDB(t)

			original := models.InventoryItem{
				ID:        "item-1",
				Name:      "Whole Milk",
				Quantity:  1,
				CreatedAt: "2026-09-23T14:00:00Z",
				UpdatedAt: "2026-09-23T14:00:00Z",
			}
			if err := database.AddInventoryItem(db, original); err != nil {
				t.Fatal(err)
			}

			mux := http.NewServeMux()
			mux.HandleFunc("PATCH /api/inventory/{id}", UpdateInventoryHandler(db))

			req := httptest.NewRequest(
				http.MethodPatch,
				"/api/inventory/item-1",
				bytes.NewBufferString(test.body),
			)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			assertAPIError(
				t,
				rec,
				http.StatusBadRequest,
				"invalid_request",
				test.expectedMessage,
			)

			stored, err := database.GetInventoryItemByID(db, original.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Quantity != original.Quantity {
				t.Errorf("expected invalid request not to change quantity")
			}
			if stored.Version != 1 {
				t.Errorf("expected invalid request to keep version 1, got %d", stored.Version)
			}
		})
	}
}

func TestUpdateInventoryHandlerRejectsStaleVersion(t *testing.T) {
	db := setupPendingAPITestDB(t)

	original := models.InventoryItem{
		ID:        "item-1",
		Name:      "Whole Milk",
		Quantity:  1,
		CreatedAt: "2026-09-23T14:00:00Z",
		UpdatedAt: "2026-09-23T14:00:00Z",
	}
	if err := database.AddInventoryItem(db, original); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(
		"UPDATE inventory_items SET version = 2 WHERE id = ?",
		original.ID,
	); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("PATCH /api/inventory/{id}", UpdateInventoryHandler(db))

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/inventory/item-1",
		bytes.NewBufferString(`{"expected_version":1,"quantity":9}`),
	)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	assertAPIError(
		t,
		rec,
		http.StatusConflict,
		"version_conflict",
		"inventory item has changed",
	)

	stored, err := database.GetInventoryItemByID(db, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Quantity != original.Quantity {
		t.Errorf("expected stale update not to change quantity; got %d", stored.Quantity)
	}
	if stored.Version != 2 {
		t.Errorf("expected stale update to keep version 2, got %d", stored.Version)
	}
}
