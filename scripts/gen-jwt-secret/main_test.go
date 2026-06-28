package main

import (
	"encoding/base64"
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

func TestGenerateSSHMasterKeyStdBase64(t *testing.T) {
	secret, err := generateSSHMasterKey(32)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw) != 32 {
		t.Fatalf("want 32 bytes, got %d", len(raw))
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
