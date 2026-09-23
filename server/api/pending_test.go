package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jae-white/stock-n-stash/server/database"
	"github.com/jae-white/stock-n-stash/server/models"
)

func setupPendingAPITestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}

	db.SetMaxOpenConns(1)

	if err := database.Initialize(db); err != nil {
		db.Close()
		t.Fatal(err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TestResolvePendingHandler(t *testing.T) {
	db := setupPendingAPITestDB(t)

	pending := models.PendingResult{
		ID:            "pending-1",
		ScanID:        "scan-1",
		AssociationID: "scan-1:item-1",
		SuggestedName: "Cheerios",
		Confidence:    0.72,
		Quantity:      1,
		CreatedAt:     "2026-09-22T19:00:00Z",
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

	var item models.InventoryItem

	if err := json.NewDecoder(rec.Body).Decode(&item); err != nil {
		t.Fatal(err)
	}

	assertValidUUID(t, item.ID)

	if item.Name != "Cheerios" {
		t.Errorf("expected name %q, got %q", "Cheerios", item.Name)
	}

	if item.Quantity != 1 {
		t.Errorf("expected quantity 1, got %d", item.Quantity)
	}

	_, err := database.GetPendingResultByID(db, "pending-1")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected pending result to be removed, got %v", err)
	}

	stored, err := database.GetInventoryItemByID(db, item.ID)
	if err != nil {
		t.Fatal(err)
	}

	if stored.Name != "Cheerios" {
		t.Errorf("expected stored name %q, got %q", "Cheerios", stored.Name)
	}
}

func TestResolvePendingHandlerWithCorrection(t *testing.T) {
	db := setupPendingAPITestDB(t)

	pending := models.PendingResult{
		ID:            "pending-1",
		ScanID:        "scan-1",
		AssociationID: "scan-1:item-1",
		SuggestedName: "Cheerios",
		Confidence:    0.72,
		Quantity:      1,
		CreatedAt:     "2026-09-22T19:00:00Z",
	}

	if err := database.AddPendingResult(db, pending); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(
		"POST /api/pending/{id}/resolve",
		ResolvePendingHandler(db),
	)

	body := `{
		"expected_version": 1,
		"name": "Honey Nut Cheerios",
		"quantity": 2
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/pending/pending-1/resolve",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var item models.InventoryItem

	if err := json.NewDecoder(rec.Body).Decode(&item); err != nil {
		t.Fatal(err)
	}

	if item.Name != "Honey Nut Cheerios" {
		t.Errorf(
			"expected corrected name %q, got %q",
			"Honey Nut Cheerios",
			item.Name,
		)
	}

	if item.Quantity != 2 {
		t.Errorf("expected corrected quantity 2, got %d", item.Quantity)
	}
}

func TestResolvePendingHandlerNotFound(t *testing.T) {
	db := setupPendingAPITestDB(t)

	mux := http.NewServeMux()
	mux.HandleFunc(
		"POST /api/pending/{id}/resolve",
		ResolvePendingHandler(db),
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/pending/does-not-exist/resolve",
		bytes.NewBufferString(`{"expected_version": 1}`),
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestResolvePendingHandlerInvalidJSON(t *testing.T) {
	db := setupPendingAPITestDB(t)

	pending := models.PendingResult{
		ID:            "pending-1",
		ScanID:        "scan-1",
		AssociationID: "scan-1:item-1",
		SuggestedName: "Cheerios",
		Confidence:    0.72,
		Quantity:      1,
		CreatedAt:     "2026-09-22T19:00:00Z",
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
		bytes.NewBufferString(`{"name":`),
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}

	// Bad JSON must not accidentally remove the pending result.
	_, err := database.GetPendingResultByID(db, "pending-1")
	if err != nil {
		t.Fatalf("expected pending result to remain, got %v", err)
	}
}

func TestResolvePendingHandlerRejectsInvalidCorrections(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		expectedMessage string
	}{
		{
			name:            "blank name",
			body:            `{"expected_version": 1, "name": "   "}`,
			expectedMessage: "name is required",
		},
		{
			name:            "null name",
			body:            `{"expected_version": 1, "name": null}`,
			expectedMessage: "name is required",
		},
		{
			name:            "zero quantity",
			body:            `{"expected_version": 1, "quantity": 0}`,
			expectedMessage: "quantity must be greater than zero",
		},
		{
			name:            "negative quantity",
			body:            `{"expected_version": 1, "quantity": -1}`,
			expectedMessage: "quantity must be greater than zero",
		},
		{
			name:            "null quantity",
			body:            `{"expected_version": 1, "quantity": null}`,
			expectedMessage: "quantity must be greater than zero",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupPendingAPITestDB(t)

			pending := models.PendingResult{
				ID:            "pending-1",
				ScanID:        "scan-1",
				AssociationID: "scan-1:item-1",
				SuggestedName: "Cheerios",
				Confidence:    0.72,
				Quantity:      1,
				CreatedAt:     "2026-09-22T19:00:00Z",
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
				bytes.NewBufferString(test.body),
			)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf(
					"expected status 400, got %d: %s",
					rec.Code,
					rec.Body.String(),
				)
			}

			var response struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}

			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("expected JSON error response: %v", err)
			}

			if response.Error.Code != "invalid_request" {
				t.Errorf(
					"expected error code %q, got %q",
					"invalid_request",
					response.Error.Code,
				)
			}

			if response.Error.Message != test.expectedMessage {
				t.Errorf(
					"expected error message %q, got %q",
					test.expectedMessage,
					response.Error.Message,
				)
			}

			if _, err := database.GetPendingResultByID(db, pending.ID); err != nil {
				t.Fatalf("expected invalid request to keep pending result: %v", err)
			}

			inventory, err := database.GetInventoryItems(db)
			if err != nil {
				t.Fatal(err)
			}

			if len(inventory) != 0 {
				t.Fatalf("expected invalid request not to create inventory")
			}
		})
	}
}
