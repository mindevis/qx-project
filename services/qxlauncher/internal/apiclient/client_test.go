package apiclient

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsUnauthorized(t *testing.T) {
	if IsUnauthorized(nil) {
		t.Fatal("nil should not be unauthorized")
	}
	err := fmt.Errorf("api GET /launcher/launch-requests/pending: 401 {\"error\":{}}")
	if !IsUnauthorized(err) {
		t.Fatalf("expected unauthorized: %v", err)
	}
	if IsUnauthorized(errors.New("network down")) {
		t.Fatal("other errors should not match")
	}
}
