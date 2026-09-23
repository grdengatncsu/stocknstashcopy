package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jae-white/stock-n-stash/server/database"
	"github.com/jae-white/stock-n-stash/server/models"
)

// SubmitScanHandler receives a completed edge-device scan and stores each item
// either in inventory or in the pending-review queue.
func SubmitScanHandler(db *sql.DB) http.HandlerFunc {
	// Returning a closure attaches the shared database handle while preserving
	// the standard http.HandlerFunc signature expected by ServeMux.
	return func(w http.ResponseWriter, r *http.Request) {
		var report models.ScanReport
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "invalid_request", "message": "failed to parse scan report"}})
			return
		}

		// Validate required fields.

		if strings.TrimSpace(report.ScanID) == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "invalid_request", "message": "scan_id is required"}})
			return
		}
		seenAssociationIDs := make(map[string]struct{})
		if report.Items == nil || len(report.Items) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "invalid_request", "message": "items are required"}})
			return
		}

		for _, item := range report.Items {
			if strings.TrimSpace(item.AssociationID) == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "invalid_request", "message": "association_id cannot be blank"}})
				return
			}
			if strings.TrimSpace(item.Name) == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "invalid_request", "message": "name cannot be blank"}})
				return
			}
			if item.Confidence < 0 || item.Confidence > 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "invalid_request", "message": "confidence must be between 0 and 1"}})
				return
			}
			if item.Quantity <= 0 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "invalid_request", "message": "item quantity must be greater than zero"}})
				return
			}
			if _, exists := seenAssociationIDs[item.AssociationID]; exists {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "invalid_request", "message": "item association_id must be unique"}})
				return
			}
			seenAssociationIDs[item.AssociationID] = struct{}{}
		}

		// Database processing is atomic and checks the scan ID, so a client may
		// safely retry after a lost response.
		alreadyProcessed, err := database.ProcessScanReport(
			db,
			report,
			time.Now().Format(time.RFC3339),
		)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "internal_error", "message": "failed to process scan report"}})
			return
		}

		// Both outcomes use HTTP 200 because a duplicate has already reached the
		// desired final state. The body tells the caller which case occurred.
		status := "accepted"
		if alreadyProcessed {
			status = "already_processed"
		}

		// This response is local to one endpoint, so an anonymous struct avoids a
		// package-wide type that has no other consumer.
		response := struct {
			ScanID string `json:"scan_id"`
			Status string `json:"status"`
		}{
			ScanID: report.ScanID,
			Status: status,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
