package database

import (
	"errors"
	"strings"
	"testing"
)

func TestWrapConnectError_unreachable(t *testing.T) {
	inner := errors.New(`dial tcp [::1]:3306: connectex: No connection could be made because the target machine actively refused it.`)
	err := wrapConnectError(inner)
	if !errors.Is(err, inner) {
		t.Fatalf("expected wrapped dial error, got %v", err)
	}
	if strings.Contains(err.Error(), "database unreachable") {
		t.Fatalf("unexpected extra message: %s", err)
	}
}

func TestWrapConnectError_other(t *testing.T) {
	err := wrapConnectError(errors.New("access denied for user"))
	if err == nil || !strings.Contains(err.Error(), "failed to connect to database") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestIsDBUnreachable(t *testing.T) {
	if !isDBUnreachable(errors.New("dial tcp 127.0.0.1:3306: connect: connection refused")) {
		t.Fatal("expected unreachable")
	}
	if isDBUnreachable(errors.New("Error 1045: Access denied")) {
		t.Fatal("auth error should not be unreachable")
	}
}
