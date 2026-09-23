package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jae-white/stock-n-stash/server/database"
	"github.com/jae-white/stock-n-stash/server/models"
)

// Inventory list response with items and sync cursor.
type InventoryListResponse struct {
	Items      []models.InventoryItem `json:"items"`
	SyncCursor string                 `json:"sync_cursor"`
}

// GetInventoryHandler returns every confirmed inventory item as JSON.
func GetInventoryHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := database.GetInventoryItems(db)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)

			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "internal_error",
					"message": "failed to retrieve inventory",
				},
			})
			return
		}
		syncCursor := ""
		var latestTime time.Time

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
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InventoryListResponse{
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
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{
						"code":    "not_found",
						"message": "inventory item not found",
					},
				})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "internal_error",
					"message": "failed to retrieve inventory item",
				},
			})
			return
		}

		var request UpdateInventoryRequest

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "invalid_request",
					"message": "invalid JSON request body",
				},
			})
			return
		}
		// Validate required fields.
		if request.Name.Present && (request.Name.Value == nil || strings.TrimSpace(*request.Name.Value) == "") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "invalid_request",
					"message": "name is required",
				},
			})
			return
		}

		if request.Quantity.Present && (request.Quantity.Value == nil || *request.Quantity.Value <= 0) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "invalid_request",
					"message": "quantity must be greater than zero",
				},
			})
			return
		}

		if !request.ExpectedVersion.Present || request.ExpectedVersion.Value == nil {
			//400 invalid_request
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "invalid_request",
					"message": "expected_version is required",
				},
			})
			return
		}

		if *request.ExpectedVersion.Value <= 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "invalid_request",
					"message": "expected_version must be greater than zero",
				},
			})
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
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{
						"code":    "version_conflict",
						"message": "inventory item has changed",
					},
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "internal_error",
					"message": "failed to update inventory item",
				},
			})
			return
		}

		item.Version++

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(item)
	}
}

// DeleteInventoryHandler removes one item selected by its path ID.
func DeleteInventoryHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		ifMatch := r.Header.Get("If-Match")
		if ifMatch == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "invalid_request",
					"message": "If-Match version is required",
				},
			})
			return
		}

		if len(ifMatch) < 3 || ifMatch[0] != '"' || ifMatch[len(ifMatch)-1] != '"' {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "invalid_request",
					"message": "If-Match must contain a positive integer version",
				},
			})
			return
		}

		// Item exists, so delete it.
		expectedVersion, err := strconv.Atoi(ifMatch[1 : len(ifMatch)-1])
		if err != nil || expectedVersion <= 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "invalid_request",
					"message": "If-Match must contain a positive integer version",
				},
			})
			return
		}

		// First check whether the item exists.
		_, err = database.GetInventoryItemByID(db, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)

				json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{
						"code":    "not_found",
						"message": "inventory item not found",
					},
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)

			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "internal_error",
					"message": "failed to retrieve inventory item",
				},
			})
			return
		}

		deletedAt := time.Now().Format(time.RFC3339)

		if err := database.DeleteInventoryItem(db, id, expectedVersion, deletedAt); err != nil {
			w.Header().Set("Content-Type", "application/json")
			if errors.Is(err, database.ErrVersionConflict) {
				w.WriteHeader(http.StatusConflict)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}

			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "version_conflict",
					"message": "inventory item has changed",
				},
			})
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

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "invalid_request",
					"message": "failed to parse request body",
				},
			})
			return
		}

		// Validate required fields.
		if !request.Name.Present || request.Name.Value == nil || strings.TrimSpace(*request.Name.Value) == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "invalid_request",
					"message": "name is required",
				},
			})
			return
		}

		if !request.Quantity.Present || request.Quantity.Value == nil || *request.Quantity.Value <= 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "invalid_request",
					"message": "quantity must be greater than zero",
				},
			})
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
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "internal_error",
					"message": "failed to add inventory item",
				},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(item)
	}
}
