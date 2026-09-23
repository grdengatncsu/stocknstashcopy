package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jae-white/stock-n-stash/server/database"
	"github.com/jae-white/stock-n-stash/server/models"
)

type PendingListResponse struct {
	Items      []models.PendingResult `json:"items"`
	SyncCursor string                 `json:"sync_cursor"`
}

// GetPendingHandler returns all recognition results that still need review.
func GetPendingHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pendingResults, err := database.GetPendingResults(db)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)

			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "internal_error",
					"message": "failed to retrieve pending results",
				},
			})
			return
		}

		syncCursor := ""
		var latestTime time.Time

		for _, item := range pendingResults {
			itemTime, err := time.Parse(time.RFC3339, item.CreatedAt)
			if err != nil {
				continue
			}
			if itemTime.After(latestTime) {
				latestTime = itemTime
				syncCursor = item.CreatedAt
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PendingListResponse{
			Items:      pendingResults,
			SyncCursor: syncCursor,
		})
	}
}

// ResolvePendingRequest contains optional human corrections. Nil fields keep
// the recognition pipeline's suggested values.
type ResolvePendingRequest struct {
	Name            Optional[string] `json:"name"`
	UPC             Optional[string] `json:"upc"`
	Quantity        Optional[int]    `json:"quantity"`
	ExpirationDate  Optional[string] `json:"expiration_date"`
	ExpectedVersion Optional[int]    `json:"expected_version"`
}

// ResolvePendingHandler turns one reviewed result into a confirmed inventory
// item and removes it from the pending queue.
func ResolvePendingHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pendingID := r.PathValue("id")

		var req ResolvePendingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
		if req.Name.Present && (req.Name.Value == nil || strings.TrimSpace(*req.Name.Value) == "") {
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

		if req.Quantity.Present && (req.Quantity.Value == nil || *req.Quantity.Value <= 0) {
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

		if !req.ExpectedVersion.Present || req.ExpectedVersion.Value == nil {
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

		if *req.ExpectedVersion.Value <= 0 {
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

		pendingResult, err := database.GetPendingResultByID(db, pendingID)
		if errors.Is(err, sql.ErrNoRows) {
			// A missing review ID is a client-visible 404, not a server failure.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "not_found",
					"message": "pending result not found",
				},
			})
			return
		} else if err != nil {
			// Other database failures are internal errors.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "internal_error",
					"message": "failed to retrieve pending result",
				},
			})
			return
		}

		// Begin with the model's suggestions, then overwrite only the values the
		// reviewer actually supplied.
		name := pendingResult.SuggestedName
		upc := pendingResult.SuggestedUPC
		quantity := pendingResult.Quantity
		var expirationDate *string

		if req.Name.Present {
			name = *req.Name.Value
		}
		if req.UPC.Present {
			upc = req.UPC.Value
		}
		if req.Quantity.Present {
			quantity = *req.Quantity.Value
		}

		if req.ExpirationDate.Present {
			expirationDate = req.ExpirationDate.Value
		}

		now := time.Now().Format(time.RFC3339)
		inventoryID := uuid.NewString()

		inventoryItem := models.InventoryItem{
			ID:             inventoryID,
			Name:           name,
			UPC:            upc,
			Quantity:       quantity,
			ExpirationDate: expirationDate,
			Version:        1,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		// The database layer inserts and removes within one transaction so the
		// item cannot disappear or exist in both queues after a partial failure.
		err = database.ResolvePendingResult(db, pendingID, *req.ExpectedVersion.Value, inventoryItem)
		if err != nil {
			if err == database.ErrVersionConflict {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{
						"code":    "version_conflict",
						"message": "pending result has changed",
					},
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{
					"code":    "internal_error",
					"message": "failed to resolve pending result",
				},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(inventoryItem)
	}
}
