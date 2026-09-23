package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jae-white/stock-n-stash/server/database"
	"github.com/jae-white/stock-n-stash/server/models"
)

func TestAddInventoryHandlerReturnsNullDeletedAt(t *testing.T) {
	db := setupPendingAPITestDB(t)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/inventory", AddInventoryHandler(db))

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/inventory",
		bytes.NewBufferString(`{"name":"Whole Milk","quantity":1}`),
	)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("expected inventory response: %v", err)
	}

	deletedAt, exists := response["deleted_at"]
	if !exists {
		t.Fatal("expected response to include deleted_at")
	}
	if deletedAt != nil {
		t.Errorf("expected deleted_at to be null, got %v", deletedAt)
	}
}

func TestDeleteInventoryHandlerCreatesVersionedTombstone(t *testing.T) {
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
	mux.HandleFunc("DELETE /api/inventory/{id}", DeleteInventoryHandler(db))

	req := httptest.NewRequest(http.MethodDelete, "/api/inventory/item-1", nil)
	req.Header.Set("If-Match", `"1"`)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty 204 response body, got %q", rec.Body.String())
	}

	var deletedAt sql.NullString
	var version int
	if err := db.QueryRow(`
		SELECT deleted_at, version
		FROM inventory_items
		WHERE id = ?
	`, original.ID).Scan(&deletedAt, &version); err != nil {
		t.Fatalf("expected tombstone row to remain stored: %v", err)
	}

	if !deletedAt.Valid {
		t.Fatal("expected deleted_at to be stored")
	}
	if _, err := time.Parse(time.RFC3339, deletedAt.String); err != nil {
		t.Errorf("expected RFC3339 deleted_at, got %q: %v", deletedAt.String, err)
	}
	if version != 2 {
		t.Errorf("expected tombstone version 2, got %d", version)
	}

	items, err := database.GetInventoryItems(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected normal inventory list to hide tombstone, got %d items", len(items))
	}

	_, err = database.GetInventoryItemByID(db, original.ID)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected normal item lookup to hide tombstone, got %v", err)
	}
}

func TestDeleteInventoryHandlerRequiresValidIfMatch(t *testing.T) {
	tests := []struct {
		name            string
		ifMatch         string
		expectedMessage string
	}{
		{
			name:            "missing header",
			expectedMessage: "If-Match version is required",
		},
		{
			name:            "unquoted version",
			ifMatch:         "1",
			expectedMessage: "If-Match must contain a positive integer version",
		},
		{
			name:            "noninteger version",
			ifMatch:         `"abc"`,
			expectedMessage: "If-Match must contain a positive integer version",
		},
		{
			name:            "zero version",
			ifMatch:         `"0"`,
			expectedMessage: "If-Match must contain a positive integer version",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupPendingAPITestDB(t)

			item := models.InventoryItem{
				ID:        "item-1",
				Name:      "Whole Milk",
				Quantity:  1,
				CreatedAt: "2026-09-23T14:00:00Z",
				UpdatedAt: "2026-09-23T14:00:00Z",
			}
			if err := database.AddInventoryItem(db, item); err != nil {
				t.Fatal(err)
			}

			mux := http.NewServeMux()
			mux.HandleFunc("DELETE /api/inventory/{id}", DeleteInventoryHandler(db))

			req := httptest.NewRequest(http.MethodDelete, "/api/inventory/item-1", nil)
			if test.ifMatch != "" {
				req.Header.Set("If-Match", test.ifMatch)
			}
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			assertAPIError(
				t,
				rec,
				http.StatusBadRequest,
				"invalid_request",
				test.expectedMessage,
			)

			var deletedAt sql.NullString
			var version int
			if err := db.QueryRow(
				"SELECT deleted_at, version FROM inventory_items WHERE id = ?",
				item.ID,
			).Scan(&deletedAt, &version); err != nil {
				t.Fatal(err)
			}
			if deletedAt.Valid {
				t.Errorf("expected invalid request not to set deleted_at")
			}
			if version != 1 {
				t.Errorf("expected invalid request to keep version 1, got %d", version)
			}
		})
	}
}

func TestDeleteInventoryHandlerRejectsStaleVersion(t *testing.T) {
	db := setupPendingAPITestDB(t)

	item := models.InventoryItem{
		ID:        "item-1",
		Name:      "Whole Milk",
		Quantity:  1,
		CreatedAt: "2026-09-23T14:00:00Z",
		UpdatedAt: "2026-09-23T14:00:00Z",
	}
	if err := database.AddInventoryItem(db, item); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(
		"UPDATE inventory_items SET version = 2 WHERE id = ?",
		item.ID,
	); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/inventory/{id}", DeleteInventoryHandler(db))

	req := httptest.NewRequest(http.MethodDelete, "/api/inventory/item-1", nil)
	req.Header.Set("If-Match", `"1"`)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	assertAPIError(
		t,
		rec,
		http.StatusConflict,
		"version_conflict",
		"inventory item has changed",
	)

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
