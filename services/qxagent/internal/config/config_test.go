package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.toml")
	content := `
api_base_url = "https://api.example.com/api/v1"
agent_token = "tok-file"
server_id = "srv-1"
server_root = "/opt/qx/server"
hostname = "vps-1"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.AgentToken != "tok-file" || cfg.ServerID != "srv-1" {
		t.Fatalf("cfg: %+v", cfg)
	}
	if cfg.APIBaseURL != "https://api.example.com/api/v1" {
		t.Fatalf("api base: %q", cfg.APIBaseURL)
	}
	if cfg.ServerRoot != "/opt/qx/server" || cfg.Hostname != "vps-1" {
		t.Fatalf("cfg: %+v", cfg)
	}
}

func TestLoadMissingToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")
	_, err := Load(path)
	if err != ErrMissingAgentToken {
		t.Fatalf("expected ErrMissingAgentToken, got %v", err)
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.toml")
	if err := os.WriteFile(path, []byte("[[[broken"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadFromRepoAgentTOML(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.22\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "services", "qxagent")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "agent.toml"), []byte(`agent_token = "repo-token"`), 0o600); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.AgentToken != "repo-token" {
		t.Fatalf("token: %q", cfg.AgentToken)
	}
}
