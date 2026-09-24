package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jae-white/stock-n-stash/server/database"
	"github.com/jae-white/stock-n-stash/server/models"
)

// InventoryListResponse wraps active records with the newest update timestamp
// that a future incremental synchronization request can use as a cursor.
type InventoryListResponse struct {
	Items      []models.InventoryItem `json:"items"`
	SyncCursor string                 `json:"sync_cursor"`
}

// GetInventoryHandler returns every confirmed inventory item as JSON.
func GetInventoryHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := database.GetInventoryItems(db)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve inventory")
			return
		}
		syncCursor := ""
		var latestTime time.Time

		// Use the latest valid update time as the initial full-list cursor. Invalid
		// stored timestamps are ignored rather than breaking the whole response.
		for _, item := range items {
			itemTime, err := time.Parse(time.RFC3339, item.UpdatedAt)
			if err != nil {
				continue
			}
			if itemTime.After(latestTime) {
				latestTime = itemTime
				syncCursor = item.UpdatedAt
			}
		}
		writeJSON(w, http.StatusOK, InventoryListResponse{
			Items:      items,
			SyncCursor: syncCursor,
		})
	}
}

// UpdateInventoryRequest uses pointers to distinguish an omitted field from a
// provided zero value, such as quantity 0 or an empty name.
type UpdateInventoryRequest struct {
	Name            Optional[string] `json:"name"`
	UPC             Optional[string] `json:"upc"`
	Quantity        Optional[int]    `json:"quantity"`
	ExpirationDate  Optional[string] `json:"expiration_date"`
	ExpectedVersion Optional[int]    `json:"expected_version"`
}

// UpdateInventoryHandler applies only the fields supplied in a PATCH request.
func UpdateInventoryHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// PathValue reads {id} from the route registered in main.go.
		id := r.PathValue("id")

		// Loading the existing record first preserves fields omitted by PATCH.
		item, err := database.GetInventoryItemByID(db, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeAPIError(w, http.StatusNotFound, "not_found", "inventory item not found")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve inventory item")
			return
		}

		var request UpdateInventoryRequest

		if err := decodeJSONBody(w, r, &request); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "invalid JSON request body")
			return
		}
		// Validate required fields.
		if request.Name.Present && (request.Name.Value == nil || strings.TrimSpace(*request.Name.Value) == "") {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "name is required")
			return
		}

		if request.Quantity.Present && (request.Quantity.Value == nil || *request.Quantity.Value <= 0) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "quantity must be greater than zero")
			return
		}

		if !request.ExpectedVersion.Present || request.ExpectedVersion.Value == nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "expected_version is required")
			return
		}

		if *request.ExpectedVersion.Value <= 0 {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "expected_version must be greater than zero")
			return
		}

		if request.Name.Present {
			item.Name = *request.Name.Value
		}
		if request.UPC.Present {
			item.UPC = request.UPC.Value
		}
		if request.Quantity.Present {
			item.Quantity = *request.Quantity.Value
		}
		if request.ExpirationDate.Present {
			item.ExpirationDate = request.ExpirationDate.Value
		}

		item.UpdatedAt = time.Now().Format(time.RFC3339)
		item.Version = *request.ExpectedVersion.Value

		if err := database.UpdateInventoryItem(db, item); err != nil {
			if errors.Is(err, database.ErrVersionConflict) {
				writeAPIError(w, http.StatusConflict, "version_conflict", "inventory item has changed")
				return
			}

			writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to update inventory item")
			return
		}

		item.Version++

		writeJSON(w, http.StatusOK, item)
	}
}

// DeleteInventoryHandler removes one item selected by its path ID.
func DeleteInventoryHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		ifMatch := r.Header.Get("If-Match")
		if ifMatch == "" {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "If-Match version is required")
			return
		}

		if len(ifMatch) < 3 || ifMatch[0] != '"' || ifMatch[len(ifMatch)-1] != '"' {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "If-Match must contain a positive integer version")
			return
		}

		// Quoted versions follow the HTTP ETag convention and avoid confusing a
		// record version with an arbitrary unquoted request value.
		expectedVersion, err := strconv.Atoi(ifMatch[1 : len(ifMatch)-1])
		if err != nil || expectedVersion <= 0 {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "If-Match must contain a positive integer version")
			return
		}

		// First check whether the item exists.
		_, err = database.GetInventoryItemByID(db, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeAPIError(w, http.StatusNotFound, "not_found", "inventory item not found")
				return
			}

			writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve inventory item")
			return
		}

		deletedAt := time.Now().Format(time.RFC3339)

		if err := database.DeleteInventoryItem(db, id, expectedVersion, deletedAt); err != nil {
			if errors.Is(err, database.ErrVersionConflict) {
				writeAPIError(w, http.StatusConflict, "version_conflict", "inventory item has changed")
				return
			}

			// A database failure is not a stale-client conflict. Returning the
			// correct code lets clients decide whether to refresh or retry later.
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to delete inventory item")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// AddInventoryRequest describes user-supplied fields. IDs and timestamps are
// generated by the server so callers cannot accidentally create conflicts.
type AddInventoryRequest struct {
	Name           Optional[string] `json:"name"`
	UPC            Optional[string] `json:"upc"`
	Quantity       Optional[int]    `json:"quantity"`
	ExpirationDate Optional[string] `json:"expiration_date"`
}

// AddInventoryHandler creates a confirmed inventory item directly, without a
// corresponding edge-device scan.
func AddInventoryHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request AddInventoryRequest

		if err := decodeJSONBody(w, r, &request); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "failed to parse request body")
			return
		}

		// Validate required fields.
		if !request.Name.Present || request.Name.Value == nil || strings.TrimSpace(*request.Name.Value) == "" {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "name is required")
			return
		}

		if !request.Quantity.Present || request.Quantity.Value == nil || *request.Quantity.Value <= 0 {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "quantity must be greater than zero")
			return
		}

		// Generate a new unique inventory ID using UUIDs.
		inventoryID := uuid.NewString()
		now := time.Now().Format(time.RFC3339)
		item := models.InventoryItem{
			ID:        inventoryID,
			Version:   1,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if request.Name.Present {
			item.Name = *request.Name.Value
		}
		if request.UPC.Present {
			item.UPC = request.UPC.Value
		}
		if request.Quantity.Present {
			item.Quantity = *request.Quantity.Value
		}
		if request.ExpirationDate.Present {
			item.ExpirationDate = request.ExpirationDate.Value
		}

		if err := database.AddInventoryItem(db, item); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to add inventory item")
			return
		}

		writeJSON(w, http.StatusCreated, item)
	}
}
