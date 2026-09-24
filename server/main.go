package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jae-white/stock-n-stash/server/api"
	"github.com/jae-white/stock-n-stash/server/database"
)

const (
	defaultAddress      = ":8080"
	defaultDatabasePath = "stocknstash.db"
)

type serverConfig struct {
	Address      string
	DatabasePath string
}

func configFromEnvironment() serverConfig {
	// Environment variables keep deployment-specific ports and persistent paths
	// out of source control while retaining easy local-development defaults.
	config := serverConfig{
		Address:      os.Getenv("STOCKNSTASH_ADDR"),
		DatabasePath: os.Getenv("STOCKNSTASH_DB_PATH"),
	}
	if config.Address == "" {
		config.Address = defaultAddress
	}
	if config.DatabasePath == "" {
		config.DatabasePath = defaultDatabasePath
	}
	return config
}

func newMux(db *sql.DB) *http.ServeMux {
	// Go 1.22+ ServeMux patterns include the HTTP method and path parameters.
	mux := http.NewServeMux()

	// Process health, used by local diagnostics and service supervision.
	mux.HandleFunc("GET /api/status", api.StatusHandler)

	// Human-managed confirmed inventory.
	mux.HandleFunc("GET /api/inventory", api.GetInventoryHandler(db))
	mux.HandleFunc("POST /api/inventory", api.AddInventoryHandler(db))
	mux.HandleFunc("PATCH /api/inventory/{id}", api.UpdateInventoryHandler(db))
	mux.HandleFunc("DELETE /api/inventory/{id}", api.DeleteInventoryHandler(db))

	// Human review of uncertain recognition results.
	mux.HandleFunc("GET /api/pending", api.GetPendingHandler(db))
	mux.HandleFunc("POST /api/pending/{id}/resolve", api.ResolvePendingHandler(db))

	// Machine-to-machine entry point used by HttpReporter after a scan.
	mux.HandleFunc("POST /api/scans", api.SubmitScanHandler(db))
	return mux
}

func newHTTPServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func run() error {
	config := configFromEnvironment()
	db, err := database.Open(config.DatabasePath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Schema creation is idempotent, so it is safe on every startup.
	if err := database.Initialize(db); err != nil {
		return err
	}

	server := newHTTPServer(config.Address, newMux(db))
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	log.Printf("Stock 'n Stash server listening on %s", config.Address)
	shutdownSignal, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-shutdownSignal.Done():
	}

	// Give active local requests time to finish so a scan is not left half-read
	// when systemd or a developer stops the service.
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownContext)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
