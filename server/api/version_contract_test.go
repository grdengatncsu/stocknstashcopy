package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jae-white/stock-n-stash/server/database"
	"github.com/jae-white/stock-n-stash/server/models"
)

func TestAddInventoryHandlerReturnsAndPersistsInitialVersion(t *testing.T) {
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

	var response struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("expected inventory response: %v", err)
	}

	if response.Version != 1 {
		t.Errorf("expected response version 1, got %d", response.Version)
	}

	assertStoredVersion(t, db, "inventory_items", response.ID, 1)
}

func TestUpdateInventoryHandlerIncrementsAndPersistsVersion(t *testing.T) {
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
		bytes.NewBufferString(`{"expected_version":1,"quantity":2}`),
	)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Quantity int `json:"quantity"`
		Version  int `json:"version"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("expected inventory response: %v", err)
	}

	if response.Quantity != 2 {
		t.Errorf("expected quantity 2, got %d", response.Quantity)
	}
	if response.Version != 2 {
		t.Errorf("expected response version 2, got %d", response.Version)
	}

	assertStoredVersion(t, db, "inventory_items", original.ID, 2)
}

func TestResolvePendingHandlerReturnsAndPersistsInitialVersion(t *testing.T) {
	db := setupPendingAPITestDB(t)

	pending := models.PendingResult{
		ID:            "pending-1",
		ScanID:        "scan-1",
		AssociationID: "scan-1:item-1",
		SuggestedName: "Cheerios",
		Confidence:    0.72,
		Quantity:      1,
		CreatedAt:     "2026-09-23T14:00:00Z",
	}
	if err := database.AddPendingResult(db, pending); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(
		"POST /api/pending/{id}/resolve",
		ResolvePendingHandler(db),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/pending/pending-1/resolve",
		bytes.NewBufferString(`{"expected_version": 1}`),
	)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("expected inventory response: %v", err)
	}

	if response.Version != 1 {
		t.Errorf("expected response version 1, got %d", response.Version)
	}

	assertStoredVersion(t, db, "inventory_items", response.ID, 1)
}

func TestGetPendingHandlerReturnsStoredVersion(t *testing.T) {
	db := setupPendingAPITestDB(t)

	pending := models.PendingResult{
		ID:            "pending-1",
		ScanID:        "scan-1",
		AssociationID: "scan-1:item-1",
		SuggestedName: "Cheerios",
		Confidence:    0.72,
		Quantity:      1,
		CreatedAt:     "2026-09-23T14:00:00Z",
	}
	if err := database.AddPendingResult(db, pending); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/pending", GetPendingHandler(db))

	req := httptest.NewRequest(http.MethodGet, "/api/pending", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Items []struct {
			ID      string `json:"id"`
			Version int    `json:"version"`
		} `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("expected pending response envelope: %v", err)
	}

	if len(response.Items) != 1 {
		t.Fatalf("expected 1 pending result, got %d", len(response.Items))
	}
	if response.Items[0].Version != 1 {
		t.Errorf("expected pending version 1, got %d", response.Items[0].Version)
	}

	assertStoredVersion(t, db, "pending_results", pending.ID, 1)
}

func assertStoredVersion(
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
		t.Fatalf("failed to read stored version: %v", err)
	}

	if version != expected {
		t.Errorf("expected stored version %d, got %d", expected, version)
	}
}
