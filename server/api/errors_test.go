package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jae-white/stock-n-stash/server/database"
	"github.com/jae-white/stock-n-stash/server/models"
)

func assertAPIError(
	t *testing.T,
	rec *httptest.ResponseRecorder,
	expectedStatus int,
	expectedCode string,
	expectedMessage string,
) {
	t.Helper()

	if rec.Code != expectedStatus {
		t.Fatalf(
			"expected status %d, got %d: %s",
			expectedStatus,
			rec.Code,
			rec.Body.String(),
		)
	}

	if contentType := rec.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("expected JSON content type, got %q", contentType)
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

	if response.Error.Code != expectedCode {
		t.Errorf(
			"expected error code %q, got %q",
			expectedCode,
			response.Error.Code,
		)
	}

	if response.Error.Message != expectedMessage {
		t.Errorf(
			"expected error message %q, got %q",
			expectedMessage,
			response.Error.Message,
		)
	}
}

func TestAddInventoryHandlerMalformedJSONError(t *testing.T) {
	db := setupPendingAPITestDB(t)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/inventory", AddInventoryHandler(db))

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/inventory",
		bytes.NewBufferString(`{"name":`),
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assertAPIError(
		t,
		rec,
		http.StatusBadRequest,
		"invalid_request",
		"failed to parse request body",
	)
}

func TestUpdateInventoryHandlerMalformedJSONError(t *testing.T) {
	db := setupPendingAPITestDB(t)

	item := models.InventoryItem{
		ID:        "item-1",
		Name:      "Whole Milk",
		Quantity:  1,
		CreatedAt: "2026-09-23T09:00:00-04:00",
		UpdatedAt: "2026-09-23T09:00:00-04:00",
	}
	if err := database.AddInventoryItem(db, item); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(
		"PATCH /api/inventory/{id}",
		UpdateInventoryHandler(db),
	)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/inventory/item-1",
		bytes.NewBufferString(`{"quantity":`),
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assertAPIError(
		t,
		rec,
		http.StatusBadRequest,
		"invalid_request",
		"invalid JSON request body",
	)
}

func TestResolvePendingHandlerMalformedJSONError(t *testing.T) {
	db := setupPendingAPITestDB(t)

	pending := models.PendingResult{
		ID:            "pending-1",
		ScanID:        "scan-1",
		AssociationID: "scan-1:item-1",
		SuggestedName: "Cheerios",
		Confidence:    0.72,
		Quantity:      1,
		CreatedAt:     "2026-09-23T09:00:00-04:00",
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

	assertAPIError(
		t,
		rec,
		http.StatusBadRequest,
		"invalid_request",
		"failed to parse request body",
	)
}

func TestSubmitScanHandlerMalformedJSONError(t *testing.T) {
	db := setupPendingAPITestDB(t)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/scans", SubmitScanHandler(db))

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/scans",
		bytes.NewBufferString(`{"scan_id":`),
	)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assertAPIError(
		t,
		rec,
		http.StatusBadRequest,
		"invalid_request",
		"failed to parse scan report",
	)
}
