package main

import (
	"log"
	"net/http"

	"github.com/jae-white/stock-n-stash/server/api"
	"github.com/jae-white/stock-n-stash/server/database"
)

func main() {
	// The relative path keeps the prototype database beside the server when the
	// documented command is run from the server directory.
	db, err := database.Open("stocknstash.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Schema creation is idempotent, so it is safe on every startup.
	if err := database.Initialize(db); err != nil {
		log.Fatal(err)
	}

	// Go 1.22+ ServeMux patterns include the HTTP method and path parameters.
	mux := http.NewServeMux()

	// Health endpoint used to check that the process is reachable.
	mux.HandleFunc("GET /api/status", api.StatusHandler)

	// Human-managed inventory operations.
	mux.HandleFunc("GET /api/inventory", api.GetInventoryHandler(db))
	mux.HandleFunc("POST /api/inventory", api.AddInventoryHandler(db))
	mux.HandleFunc("PATCH /api/inventory/{id}", api.UpdateInventoryHandler(db))
	mux.HandleFunc("DELETE /api/inventory/{id}", api.DeleteInventoryHandler(db))

	// Review queue for results the recognition pipeline was unsure about.
	mux.HandleFunc("GET /api/pending", api.GetPendingHandler(db))
	mux.HandleFunc("POST /api/pending/{id}/resolve", api.ResolvePendingHandler(db))

	// Entry point used by the edge device after a scan is complete.
	mux.HandleFunc("POST /api/scans", api.SubmitScanHandler(db))

	log.Println("Stock 'n Stash server listening on port 8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
