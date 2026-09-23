package api

import (
	"testing"

	"github.com/google/uuid"
)

func assertValidUUID(t *testing.T, id string) {
	t.Helper()

	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("expected UUID, got %q: %v", id, err)
	}

	if parsed.String() != id {
		t.Errorf("expected canonical UUID %q, got %q", parsed.String(), id)
	}
}
