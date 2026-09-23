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

func TestAddInventoryHandler(t *testing.T) {
	db := setupPendingAPITestDB(t)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/inventory", AddInventoryHandler(db))

	body := `{
		"name": "Whole Milk",
		"upc": "012345678901",
		"quantity": 1,
		"expiration_date": "2026-10-01"
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/inventory",
		bytes.NewBufferString(body),
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf(
			"expected status 201, got %d: %s",
			rec.Code,
			rec.Body.String(),
		)
	}

	var item models.InventoryItem

	if err := json.NewDecoder(rec.Body).Decode(&item); err != nil {
		t.Fatal(err)
	}

	assertValidUUID(t, item.ID)

	if item.Name != "Whole Milk" {
		t.Errorf("expected name %q, got %q", "Whole Milk", item.Name)
	}

	if item.Quantity != 1 {
		t.Errorf("expected quantity 1, got %d", item.Quantity)
	}

	if item.UPC == nil || *item.UPC != "012345678901" {
		t.Errorf("expected UPC %q", "012345678901")
	}

	if item.ExpirationDate == nil || *item.ExpirationDate != "2026-10-01" {
		t.Errorf("expected expiration date %q", "2026-10-01")
	}

	if item.CreatedAt == "" {
		t.Error("expected created_at to be generated")
	}

	if item.UpdatedAt == "" {
		t.Error("expected updated_at to be generated")
	}

	items, err := database.GetInventoryItems(db)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 persisted inventory item, got %d", len(items))
	}

	if items[0].ID != item.ID {
		t.Errorf(
			"expected persisted ID %q, got %q",
			item.ID,
			items[0].ID,
		)
	}
}

func TestAddInventoryHandlerRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		expectedMessage string
	}{
		{
			name:            "null name",
			body:            `{"name": null, "quantity": 1}`,
			expectedMessage: "name is required",
		},
		{
			name:            "missing name",
			body:            `{"quantity": 1}`,
			expectedMessage: "name is required",
		},
		{
			name:            "blank name",
			body:            `{"name": "   ", "quantity": 1}`,
			expectedMessage: "name is required",
		},
		{
			name:            "missing quantity",
			body:            `{"name": "Whole Milk"}`,
			expectedMessage: "quantity must be greater than zero",
		},
		{
			name:            "null quantity",
			body:            `{"name": "Whole Milk", "quantity": null}`,
			expectedMessage: "quantity must be greater than zero",
		},
		{
			name:            "zero quantity",
			body:            `{"name": "Whole Milk", "quantity": 0}`,
			expectedMessage: "quantity must be greater than zero",
		},
		{
			name:            "negative quantity",
			body:            `{"name": "Whole Milk", "quantity": -1}`,
			expectedMessage: "quantity must be greater than zero",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupPendingAPITestDB(t)

			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/inventory", AddInventoryHandler(db))

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/inventory",
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

			items, err := database.GetInventoryItems(db)
			if err != nil {
				t.Fatal(err)
			}

			if len(items) != 0 {
				t.Fatalf("expected invalid request not to persist an item")
			}
		})
	}
}

func TestUpdateInventoryHandlerRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		expectedMessage string
	}{
		{
			name:            "null name",
			body:            `{"expected_version": 1, "name": null}`,
			expectedMessage: "name is required",
		},
		{
			name:            "blank name",
			body:            `{"expected_version": 1, "name": "   "}`,
			expectedMessage: "name is required",
		},
		{
			name:            "null quantity",
			body:            `{"expected_version": 1, "quantity": null}`,
			expectedMessage: "quantity must be greater than zero",
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupPendingAPITestDB(t)

			original := models.InventoryItem{
				ID:        "item-1",
				Name:      "Whole Milk",
				Quantity:  1,
				CreatedAt: "2026-09-23T09:00:00-04:00",
				UpdatedAt: "2026-09-23T09:00:00-04:00",
			}

			if err := database.AddInventoryItem(db, original); err != nil {
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

			stored, err := database.GetInventoryItemByID(db, original.ID)
			if err != nil {
				t.Fatal(err)
			}

			if stored.Name != original.Name {
				t.Errorf(
					"expected invalid request not to change name; got %q",
					stored.Name,
				)
			}

			if stored.Quantity != original.Quantity {
				t.Errorf(
					"expected invalid request not to change quantity; got %d",
					stored.Quantity,
				)
			}
		})
	}
}

func TestUpdateInventoryHandlerDistinguishesNullFromOmitted(t *testing.T) {
	tests := []struct {
		name                 string
		body                 string
		expectUPC            bool
		expectExpirationDate bool
	}{
		{
			name:                 "omitted nullable fields remain unchanged",
			body:                 `{"expected_version": 1}`,
			expectUPC:            true,
			expectExpirationDate: true,
		},
		{
			name:                 "explicit null clears nullable fields",
			body:                 `{"expected_version": 1, "upc": null, "expiration_date": null}`,
			expectUPC:            false,
			expectExpirationDate: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupPendingAPITestDB(t)

			upc := "012345678901"
			expirationDate := "2026-10-01"
			original := models.InventoryItem{
				ID:             "item-1",
				Name:           "Whole Milk",
				UPC:            &upc,
				Quantity:       1,
				ExpirationDate: &expirationDate,
				CreatedAt:      "2026-09-23T09:00:00-04:00",
				UpdatedAt:      "2026-09-23T09:00:00-04:00",
			}

			if err := database.AddInventoryItem(db, original); err != nil {
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
				bytes.NewBufferString(test.body),
			)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf(
					"expected status 200, got %d: %s",
					rec.Code,
					rec.Body.String(),
				)
			}

			var response models.InventoryItem
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("expected inventory item response: %v", err)
			}

			assertNullableFieldState(
				t,
				"response UPC",
				response.UPC,
				test.expectUPC,
				upc,
			)
			assertNullableFieldState(
				t,
				"response expiration date",
				response.ExpirationDate,
				test.expectExpirationDate,
				expirationDate,
			)

			stored, err := database.GetInventoryItemByID(db, original.ID)
			if err != nil {
				t.Fatal(err)
			}

			assertNullableFieldState(
				t,
				"stored UPC",
				stored.UPC,
				test.expectUPC,
				upc,
			)
			assertNullableFieldState(
				t,
				"stored expiration date",
				stored.ExpirationDate,
				test.expectExpirationDate,
				expirationDate,
			)
		})
	}
}

func assertNullableFieldState(
	t *testing.T,
	field string,
	value *string,
	expectValue bool,
	expected string,
) {
	t.Helper()

	if !expectValue {
		if value != nil {
			t.Errorf("expected %s to be null, got %q", field, *value)
		}
		return
	}

	if value == nil {
		t.Errorf("expected %s to remain %q, got null", field, expected)
		return
	}

	if *value != expected {
		t.Errorf("expected %s %q, got %q", field, expected, *value)
	}
}
