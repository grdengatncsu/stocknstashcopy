package api

import (
	"net/http"
)

// StatusResponse is the JSON body returned by the health endpoint.
type StatusResponse struct {
	Status string `json:"status"`
}

// StatusHandler provides a lightweight health check without querying the
// database or changing application state.
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	// HARDWARE TODO: Once the physical adapters exist, decide whether this
	// endpoint should expose camera, load-cell, lighting, Hailo, and sync health
	// or whether a separate authenticated diagnostics endpoint should do so.
	writeJSON(w, http.StatusOK, StatusResponse{Status: "ok"})
}
