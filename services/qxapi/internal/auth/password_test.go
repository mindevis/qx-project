package auth

import (
	"errors"
	"testing"
)

func TestHashPasswordAndCheck(t *testing.T) {
	hash, err := HashPassword("password123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !CheckPassword(hash, "password123") {
		t.Fatal("expected password to match")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("expected wrong password to fail")
	}
}

func TestHashTokenDeterministic(t *testing.T) {
	a := HashToken("token-a")
	b := HashToken("token-a")
	if a != b || a == "" {
		t.Fatalf("expected stable hash, got %q %q", a, b)
	}
	if HashToken("token-b") == a {
		t.Fatal("expected different tokens to hash differently")
	}
}

func TestHashPasswordError(t *testing.T) {
	old := hashPasswordFn
	hashPasswordFn = func(_ []byte, _ int) ([]byte, error) {
		return nil, errors.New("hash failed")
	}
	t.Cleanup(func() { hashPasswordFn = old })

	if _, err := HashPassword("password123"); err == nil {
		t.Fatal("expected hash error")
	}
}
