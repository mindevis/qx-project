package apiclient

import (
	"context"
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

func TestIsUnavailable(t *testing.T) {
	if IsUnavailable(nil) {
		t.Fatal("nil should not be unavailable")
	}
	cases := []error{
		errors.New(`dial tcp [::1]:3000: connectex: No connection could be made because the target machine actively refused it.`),
		errors.New("connection refused"),
		errors.New("no such host"),
		context.DeadlineExceeded,
	}
	for _, err := range cases {
		if !IsUnavailable(err) {
			t.Fatalf("expected unavailable: %v", err)
		}
	}
	if IsUnavailable(fmt.Errorf("api GET: 401")) {
		t.Fatal("401 should not be unavailable")
	}
}
