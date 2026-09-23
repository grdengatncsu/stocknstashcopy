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

func addPendingResolutionFixture(t *testing.T, db *sql.DB) models.PendingResult {
	t.Helper()

	upc := "012345678905"
	pending := models.PendingResult{
		ID:            "pending-1",
		ScanID:        "scan-1",
		AssociationID: "scan-1:item-1",
		SuggestedName: "Cheerios",
		SuggestedUPC:  &upc,
		Confidence:    0.72,
		Quantity:      1,
		CreatedAt:     "2026-09-23T14:00:00Z",
	}
	if err := database.AddPendingResult(db, pending); err != nil {
		t.Fatal(err)
	}

	return pending
}

func servePendingResolution(t *testing.T, db *sql.DB, body string) *httptest.ResponseRecorder {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc(
		"POST /api/pending/{id}/resolve",
		ResolvePendingHandler(db),
	)
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/pending/pending-1/resolve",
		bytes.NewBufferString(body),
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	return rec
}

func TestResolvePendingHandlerRequiresExpectedVersion(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		expectedMessage string
	}{
		{
			name:            "missing",
			body:            `{}`,
			expectedMessage: "expected_version is required",
		},
		{
			name:            "null",
			body:            `{"expected_version": null}`,
			expectedMessage: "expected_version is required",
		},
		{
			name:            "zero",
			body:            `{"expected_version": 0}`,
			expectedMessage: "expected_version must be greater than zero",
		},
		{
			name:            "negative",
			body:            `{"expected_version": -1}`,
			expectedMessage: "expected_version must be greater than zero",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupPendingAPITestDB(t)
			pending := addPendingResolutionFixture(t, db)

			rec := servePendingResolution(t, db, test.body)
			assertAPIError(
				t,
				rec,
				http.StatusBadRequest,
				"invalid_request",
				test.expectedMessage,
			)

			if _, err := database.GetPendingResultByID(db, pending.ID); err != nil {
				t.Fatalf("expected invalid request to keep pending result: %v", err)
			}
			items, err := database.GetInventoryItems(db)
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 0 {
				t.Fatalf("expected invalid request not to create inventory")
			}
		})
	}
}

func TestResolvePendingHandlerRejectsStaleVersion(t *testing.T) {
	db := setupPendingAPITestDB(t)
	pending := addPendingResolutionFixture(t, db)

	if _, err := db.Exec(
		"UPDATE pending_results SET version = 2 WHERE id = ?",
		pending.ID,
	); err != nil {
		t.Fatal(err)
	}

	rec := servePendingResolution(t, db, `{"expected_version": 1}`)
	assertAPIError(
		t,
		rec,
		http.StatusConflict,
		"version_conflict",
		"pending result has changed",
	)

	stored, err := database.GetPendingResultByID(db, pending.ID)
	if err != nil {
		t.Fatalf("expected stale resolution to keep pending result: %v", err)
	}
	if stored.Version != 2 {
		t.Errorf("expected stale resolution to keep version 2, got %d", stored.Version)
	}

	items, err := database.GetInventoryItems(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected stale resolution not to create inventory")
	}
}

func TestResolvePendingHandlerCreatesTombstoneAtomically(t *testing.T) {
	db := setupPendingAPITestDB(t)
	pending := addPendingResolutionFixture(t, db)

	rec := servePendingResolution(t, db, `{"expected_version": 1}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var deletedAt sql.NullString
	var version int
	if err := db.QueryRow(`
		SELECT deleted_at, version
		FROM pending_results
		WHERE id = ?
	`, pending.ID).Scan(&deletedAt, &version); err != nil {
		t.Fatalf("expected pending tombstone row to remain stored: %v", err)
	}
	if !deletedAt.Valid {
		t.Fatal("expected resolved pending result to store deleted_at")
	}
	if _, err := time.Parse(time.RFC3339, deletedAt.String); err != nil {
		t.Errorf("expected RFC3339 deleted_at, got %q: %v", deletedAt.String, err)
	}
	if version != 2 {
		t.Errorf("expected pending tombstone version 2, got %d", version)
	}

	if _, err := database.GetPendingResultByID(db, pending.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected normal lookup to hide pending tombstone, got %v", err)
	}
	pendingItems, err := database.GetPendingResults(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(pendingItems) != 0 {
		t.Fatalf("expected pending list to hide tombstone, got %d items", len(pendingItems))
	}

	inventoryItems, err := database.GetInventoryItems(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(inventoryItems) != 1 {
		t.Fatalf("expected exactly one inventory item, got %d", len(inventoryItems))
	}
}

func TestResolvePendingHandlerDistinguishesOmittedAndNullUPC(t *testing.T) {
	t.Run("omitted keeps suggestion", func(t *testing.T) {
		db := setupPendingAPITestDB(t)
		pending := addPendingResolutionFixture(t, db)

		rec := servePendingResolution(t, db, `{"expected_version": 1}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var item models.InventoryItem
		if err := json.NewDecoder(rec.Body).Decode(&item); err != nil {
			t.Fatal(err)
		}
		if item.UPC == nil || pending.SuggestedUPC == nil || *item.UPC != *pending.SuggestedUPC {
			t.Errorf("expected omitted UPC to keep suggested value")
		}
	})

	t.Run("null clears suggestion", func(t *testing.T) {
		db := setupPendingAPITestDB(t)
		addPendingResolutionFixture(t, db)

		rec := servePendingResolution(
			t,
			db,
			`{"expected_version": 1, "upc": null}`,
		)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var item models.InventoryItem
		if err := json.NewDecoder(rec.Body).Decode(&item); err != nil {
			t.Fatal(err)
		}
		if item.UPC != nil {
			t.Errorf("expected explicit null UPC to clear suggestion, got %q", *item.UPC)
		}
	})
}

func TestGetPendingHandlerIncludesSyncMetadata(t *testing.T) {
	db := setupPendingAPITestDB(t)
	pending := addPendingResolutionFixture(t, db)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/pending", GetPendingHandler(db))
	req := httptest.NewRequest(http.MethodGet, "/api/pending", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected one pending result, got %d", len(response.Items))
	}
	if response.Items[0]["updated_at"] != pending.CreatedAt {
		t.Errorf("expected updated_at %q, got %v", pending.CreatedAt, response.Items[0]["updated_at"])
	}
	deletedAt, exists := response.Items[0]["deleted_at"]
	if !exists {
		t.Fatal("expected pending result to include deleted_at")
	}
	if deletedAt != nil {
		t.Errorf("expected active pending result deleted_at to be null, got %v", deletedAt)
	}
}
