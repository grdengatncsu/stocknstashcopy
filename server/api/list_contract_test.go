package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jae-white/stock-n-stash/server/database"
	"github.com/jae-white/stock-n-stash/server/models"
)

func TestGetInventoryHandlerUsesContractEnvelope(t *testing.T) {
	db := setupPendingAPITestDB(t)

	item := models.InventoryItem{
		ID:        "item-1",
		Name:      "Whole Milk",
		Quantity:  1,
		CreatedAt: "2026-09-23T14:00:00Z",
		UpdatedAt: "2026-09-23T14:30:00Z",
	}
	newerItem := models.InventoryItem{
		ID:        "item-2",
		Name:      "Orange Juice",
		Quantity:  1,
		CreatedAt: "2026-09-23T13:00:00Z",
		UpdatedAt: "2026-09-23T15:00:00Z",
	}

	if err := database.AddInventoryItem(db, item); err != nil {
		t.Fatal(err)
	}
	if err := database.AddInventoryItem(db, newerItem); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/inventory", GetInventoryHandler(db))

	req := httptest.NewRequest(http.MethodGet, "/api/inventory", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Items      []models.InventoryItem `json:"items"`
		SyncCursor string                 `json:"sync_cursor"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("expected inventory response envelope: %v", err)
	}

	if len(response.Items) != 2 {
		t.Fatalf("expected 2 inventory items, got %d", len(response.Items))
	}

	if response.SyncCursor != newerItem.UpdatedAt {
		t.Errorf(
			"expected sync_cursor %q, got %q",
			newerItem.UpdatedAt,
			response.SyncCursor,
		)
	}
}

func TestGetInventoryHandlerReturnsEmptyItemsArray(t *testing.T) {
	db := setupPendingAPITestDB(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/inventory", GetInventoryHandler(db))

	req := httptest.NewRequest(http.MethodGet, "/api/inventory", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Items      []models.InventoryItem `json:"items"`
		SyncCursor string                 `json:"sync_cursor"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("expected inventory response envelope: %v", err)
	}

	if response.Items == nil {
		t.Fatal("expected items to be an empty array, not null or omitted")
	}

	if len(response.Items) != 0 {
		t.Fatalf("expected no inventory items, got %d", len(response.Items))
	}

	if response.SyncCursor != "" {
		t.Errorf("expected empty sync_cursor, got %q", response.SyncCursor)
	}
}

func TestGetPendingHandlerUsesContractEnvelope(t *testing.T) {
	db := setupPendingAPITestDB(t)

	pending := models.PendingResult{
		ID:            "pending-1",
		ScanID:        "scan-1",
		AssociationID: "scan-1:item-1",
		SuggestedName: "Cheerios",
		Confidence:    0.68,
		Quantity:      1,
		CreatedAt:     "2026-09-23T14:01:00Z",
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
		Items      []models.PendingResult `json:"items"`
		SyncCursor string                 `json:"sync_cursor"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("expected pending response envelope: %v", err)
	}

	if len(response.Items) != 1 {
		t.Fatalf("expected 1 pending result, got %d", len(response.Items))
	}

	if response.Items[0].ID != pending.ID {
		t.Errorf("expected pending ID %q, got %q", pending.ID, response.Items[0].ID)
	}

	if response.SyncCursor != pending.CreatedAt {
		t.Errorf(
			"expected sync_cursor %q, got %q",
			pending.CreatedAt,
			response.SyncCursor,
		)
	}
}

func TestGetPendingHandlerReturnsEmptyItemsArray(t *testing.T) {
	db := setupPendingAPITestDB(t)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/pending", GetPendingHandler(db))

	req := httptest.NewRequest(http.MethodGet, "/api/pending", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Items      []models.PendingResult `json:"items"`
		SyncCursor string                 `json:"sync_cursor"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("expected pending response envelope: %v", err)
	}

	if response.Items == nil {
		t.Fatal("expected items to be an empty array, not null or omitted")
	}

	if len(response.Items) != 0 {
		t.Fatalf("expected no pending results, got %d", len(response.Items))
	}

	if response.SyncCursor != "" {
		t.Errorf("expected empty sync_cursor, got %q", response.SyncCursor)
	}
}
