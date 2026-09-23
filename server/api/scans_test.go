package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jae-white/stock-n-stash/server/database"
	"github.com/jae-white/stock-n-stash/server/models"
)

func TestSubmitScanHandlerAccepted(t *testing.T) {
	db := setupPendingAPITestDB(t)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/scans", SubmitScanHandler(db))

	body := `{
		"scan_id": "scan-1",
		"items": [
			{
				"association_id": "scan-1:item-1",
				"name": "Cheerios",
				"upc": null,
				"confidence": 0.95,
				"quantity": 1,
				"requires_review": false
			}
		]
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/scans",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		ScanID string `json:"scan_id"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	if response.ScanID != "scan-1" {
		t.Errorf("expected scan_id %q, got %q", "scan-1", response.ScanID)
	}

	if response.Status != "accepted" {
		t.Errorf("expected status %q, got %q", "accepted", response.Status)
	}

	inventory, err := database.GetInventoryItems(db)
	if err != nil {
		t.Fatal(err)
	}

	if len(inventory) != 1 {
		t.Fatalf("expected 1 inventory item, got %d", len(inventory))
	}

	if inventory[0].Name != "Cheerios" {
		t.Errorf("expected inventory item %q, got %q", "Cheerios", inventory[0].Name)
	}
}

func TestSubmitScanHandlerDuplicate(t *testing.T) {
	db := setupPendingAPITestDB(t)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/scans", SubmitScanHandler(db))

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

	body, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}

	firstReq := httptest.NewRequest(
		http.MethodPost,
		"/api/scans",
		bytes.NewReader(body),
	)

	firstRec := httptest.NewRecorder()
	mux.ServeHTTP(firstRec, firstReq)

	if firstRec.Code != http.StatusOK {
		t.Fatalf(
			"expected first request status 200, got %d: %s",
			firstRec.Code,
			firstRec.Body.String(),
		)
	}

	secondReq := httptest.NewRequest(
		http.MethodPost,
		"/api/scans",
		bytes.NewReader(body),
	)

	secondRec := httptest.NewRecorder()
	mux.ServeHTTP(secondRec, secondReq)

	if secondRec.Code != http.StatusOK {
		t.Fatalf(
			"expected duplicate request status 200, got %d: %s",
			secondRec.Code,
			secondRec.Body.String(),
		)
	}

	var response struct {
		ScanID string `json:"scan_id"`
		Status string `json:"status"`
	}

	if err := json.NewDecoder(secondRec.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	if response.Status != "already_processed" {
		t.Errorf(
			"expected duplicate status %q, got %q",
			"already_processed",
			response.Status,
		)
	}

	inventory, err := database.GetInventoryItems(db)
	if err != nil {
		t.Fatal(err)
	}

	if len(inventory) != 1 {
		t.Fatalf(
			"expected duplicate request not to add inventory; got %d items",
			len(inventory),
		)
	}
}

func TestSubmitScanHandlerInvalidJSON(t *testing.T) {
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

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSubmitScanHandlerRejectsInvalidReport(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		scanID          string
		expectedMessage string
	}{
		{
			name: "missing scan ID",
			body: `{
				"items": [{
					"association_id": "scan-1:item-1",
					"name": "Cheerios",
					"confidence": 0.95,
					"quantity": 1,
					"requires_review": false
				}]
			}`,
			expectedMessage: "scan_id is required",
		},
		{
			name: "blank scan ID",
			body: `{
				"scan_id": "   ",
				"items": [{
					"association_id": "scan-1:item-1",
					"name": "Cheerios",
					"confidence": 0.95,
					"quantity": 1,
					"requires_review": false
				}]
			}`,
			expectedMessage: "scan_id is required",
		},
		{
			name:            "missing items",
			body:            `{"scan_id": "scan-1"}`,
			scanID:          "scan-1",
			expectedMessage: "items are required",
		},
		{
			name:            "empty items",
			body:            `{"scan_id": "scan-1", "items": []}`,
			scanID:          "scan-1",
			expectedMessage: "items are required",
		},
		{
			name: "missing association ID",
			body: `{
				"scan_id": "scan-1",
				"items": [{
					"name": "Cheerios",
					"confidence": 0.95,
					"quantity": 1,
					"requires_review": false
				}]
			}`,
			scanID:          "scan-1",
			expectedMessage: "association_id cannot be blank",
		},
		{
			name: "blank item name",
			body: `{
				"scan_id": "scan-1",
				"items": [{
					"association_id": "scan-1:item-1",
					"name": "   ",
					"confidence": 0.95,
					"quantity": 1,
					"requires_review": false
				}]
			}`,
			scanID:          "scan-1",
			expectedMessage: "name cannot be blank",
		},
		{
			name: "negative confidence",
			body: `{
				"scan_id": "scan-1",
				"items": [{
					"association_id": "scan-1:item-1",
					"name": "Cheerios",
					"confidence": -0.01,
					"quantity": 1,
					"requires_review": false
				}]
			}`,
			scanID:          "scan-1",
			expectedMessage: "confidence must be between 0 and 1",
		},
		{
			name: "confidence above one",
			body: `{
				"scan_id": "scan-1",
				"items": [{
					"association_id": "scan-1:item-1",
					"name": "Cheerios",
					"confidence": 1.01,
					"quantity": 1,
					"requires_review": false
				}]
			}`,
			scanID:          "scan-1",
			expectedMessage: "confidence must be between 0 and 1",
		},
		{
			name: "zero quantity",
			body: `{
				"scan_id": "scan-1",
				"items": [{
					"association_id": "scan-1:item-1",
					"name": "Cheerios",
					"confidence": 0.95,
					"quantity": 0,
					"requires_review": false
				}]
			}`,
			scanID:          "scan-1",
			expectedMessage: "item quantity must be greater than zero",
		},
		{
			name: "duplicate association IDs",
			body: `{
				"scan_id": "scan-1",
				"items": [
					{
						"association_id": "scan-1:item-1",
						"name": "Cheerios",
						"confidence": 0.95,
						"quantity": 1,
						"requires_review": false
					},
					{
						"association_id": "scan-1:item-1",
						"name": "Cheerios",
						"confidence": 0.95,
						"quantity": 1,
						"requires_review": false
					}
				]
			}`,
			scanID:          "scan-1",
			expectedMessage: "item association_id must be unique",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupPendingAPITestDB(t)

			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/scans", SubmitScanHandler(db))

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/scans",
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

			inventory, err := database.GetInventoryItems(db)
			if err != nil {
				t.Fatal(err)
			}
			if len(inventory) != 0 {
				t.Fatalf("expected invalid report not to create inventory")
			}

			pending, err := database.GetPendingResults(db)
			if err != nil {
				t.Fatal(err)
			}
			if len(pending) != 0 {
				t.Fatalf("expected invalid report not to create pending results")
			}

			if test.scanID != "" {
				processed, err := database.IsScanProcessed(db, test.scanID)
				if err != nil {
					t.Fatal(err)
				}
				if processed {
					t.Fatalf("expected invalid report not to mark scan as processed")
				}
			}
		})
	}
}
