package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/jae-white/stock-n-stash/server/database"
	"github.com/jae-white/stock-n-stash/server/models"
)

// PendingListResponse wraps unresolved results with a cursor representing the
// newest result returned by this request.
type PendingListResponse struct {
	Items      []models.PendingResult `json:"items"`
	SyncCursor string                 `json:"sync_cursor"`
}

// GetPendingHandler returns all recognition results that still need review.
func GetPendingHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pendingResults, err := database.GetPendingResults(db)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve pending results")
			return
		}

		syncCursor := ""
		var latestTime time.Time

		// Find the newest valid creation time for the response cursor.
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

		writeJSON(w, http.StatusOK, PendingListResponse{
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
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "failed to parse request body")
			return
		}

		// Validate required fields.
		if req.Name.Present && (req.Name.Value == nil || strings.TrimSpace(*req.Name.Value) == "") {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "name is required")
			return
		}

		if req.Quantity.Present && (req.Quantity.Value == nil || *req.Quantity.Value <= 0) {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "quantity must be greater than zero")
			return
		}

		if !req.ExpectedVersion.Present || req.ExpectedVersion.Value == nil {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "expected_version is required")
			return
		}

		if *req.ExpectedVersion.Value <= 0 {
			writeAPIError(w, http.StatusBadRequest, "invalid_request", "expected_version must be greater than zero")
			return
		}

		pendingResult, err := database.GetPendingResultByID(db, pendingID)
		if errors.Is(err, sql.ErrNoRows) {
			// A missing review ID is a client-visible 404, not a server failure.
			writeAPIError(w, http.StatusNotFound, "not_found", "pending result not found")
			return
		} else if err != nil {
			// Other database failures are internal errors.
			writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve pending result")
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
			if errors.Is(err, database.ErrVersionConflict) {
				writeAPIError(w, http.StatusConflict, "version_conflict", "pending result has changed")
				return
			}

			writeAPIError(w, http.StatusInternalServerError, "internal_error", "failed to resolve pending result")
			return
		}

		writeJSON(w, http.StatusOK, inventoryItem)
	}
}
