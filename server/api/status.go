package api

import (
	"encoding/json"
	"log"
	"net/http"
)

// StatusResponse is the JSON body returned by the health endpoint.
type StatusResponse struct {
	Status string `json:"status"`
}

// StatusHandler provides a lightweight health check without querying the
// database or changing application state.
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := StatusResponse{Status: "ok"}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}
