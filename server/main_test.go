package main

import (
	"net/http"
	"testing"
	"time"
)

func TestConfigFromEnvironmentUsesDefaults(t *testing.T) {
	t.Setenv("STOCKNSTASH_ADDR", "")
	t.Setenv("STOCKNSTASH_DB_PATH", "")

	config := configFromEnvironment()

	if config.Address != defaultAddress {
		t.Errorf("expected address %q, got %q", defaultAddress, config.Address)
	}
	if config.DatabasePath != defaultDatabasePath {
		t.Errorf("expected database path %q, got %q", defaultDatabasePath, config.DatabasePath)
	}
}

func TestConfigFromEnvironmentUsesDeploymentValues(t *testing.T) {
	t.Setenv("STOCKNSTASH_ADDR", "127.0.0.1:9090")
	t.Setenv("STOCKNSTASH_DB_PATH", "/var/lib/stock-n-stash/inventory.db")

	config := configFromEnvironment()

	if config.Address != "127.0.0.1:9090" {
		t.Errorf("unexpected address %q", config.Address)
	}
	if config.DatabasePath != "/var/lib/stock-n-stash/inventory.db" {
		t.Errorf("unexpected database path %q", config.DatabasePath)
	}
}

func TestHTTPServerHasResourceTimeouts(t *testing.T) {
	server := newHTTPServer(":0", http.NewServeMux())

	if server.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("unexpected header timeout %s", server.ReadHeaderTimeout)
	}
	if server.ReadTimeout != 15*time.Second {
		t.Errorf("unexpected read timeout %s", server.ReadTimeout)
	}
	if server.WriteTimeout != 15*time.Second {
		t.Errorf("unexpected write timeout %s", server.WriteTimeout)
	}
	if server.IdleTimeout != 60*time.Second {
		t.Errorf("unexpected idle timeout %s", server.IdleTimeout)
	}
}
