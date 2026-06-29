package minecraft

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureOfflineSkin(t *testing.T) {
	dir := t.TempDir()
	uuid := "11111111-2222-3333-4444-555555555555"

	if err := EnsureOfflineSkin(dir, uuid, ModelSteve); err != nil {
		t.Fatalf("steve: %v", err)
	}
	skinPath := filepath.Join(dir, "skins", strings.ReplaceAll(uuid, "-", "")+".png")
	if _, err := os.Stat(skinPath); err != nil {
		t.Fatalf("skin file: %v", err)
	}

	if err := EnsureOfflineSkin(dir, uuid, ModelAlex); err != nil {
		t.Fatalf("alex: %v", err)
	}
	if _, err := os.Stat(skinPath); err != nil {
		t.Fatalf("alex skin file: %v", err)
	}
}

func TestNormalizeSkinModel(t *testing.T) {
	if NormalizeSkinModel("ALEX") != ModelAlex {
		t.Fatal("alex")
	}
	if NormalizeSkinModel("") != ModelSteve {
		t.Fatal("default steve")
	}
}

func TestEnsureOfflineSkinValidation(t *testing.T) {
	if err := EnsureOfflineSkin("", "uuid", ModelSteve); err == nil {
		t.Fatal("expected error")
	}
}
