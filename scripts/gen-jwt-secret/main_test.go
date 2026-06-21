package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateSecretLength(t *testing.T) {
	secret, err := generateSecret(32)
	if err != nil {
		t.Fatal(err)
	}
	if len(secret) < 32 {
		t.Fatalf("secret too short: %d", len(secret))
	}
}

func TestGenerateSecretRejectsShortByteCount(t *testing.T) {
	secret, err := generateSecret(8)
	if err != nil {
		t.Fatal(err)
	}
	if secret == "" {
		t.Fatal("expected secret")
	}
}

func TestPatchTomlFileCreateAndUpdate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "qxapi.toml")

	if err := patchTomlFile(path, "first"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `jwt_secret = "first"`) {
		t.Fatalf("create: %q", data)
	}

	if err := patchTomlFile(path, "second"); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `jwt_secret = "second"`) {
		t.Fatalf("update: %q", content)
	}
	if strings.Contains(content, `jwt_secret = "first"`) {
		t.Fatalf("old secret still present: %q", content)
	}
}
