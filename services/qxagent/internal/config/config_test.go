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

	t.Setenv("QX_AGENT_CONFIG", path)
	t.Setenv("QX_AGENT_TOKEN", "")
	t.Setenv("QX_API_BASE_URL", "")

	cfg, err := Load()
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

func TestLoadEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.toml")
	if err := os.WriteFile(path, []byte(`agent_token = "from-file"`), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("QX_AGENT_CONFIG", path)
	t.Setenv("QX_AGENT_TOKEN", "from-env")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.AgentToken != "from-env" {
		t.Fatalf("token: %q", cfg.AgentToken)
	}
}

func TestLoadFromEnvOnly(t *testing.T) {
	t.Setenv("QX_AGENT_CONFIG", filepath.Join(t.TempDir(), "missing.toml"))
	t.Setenv("QX_AGENT_TOKEN", "env-token")
	t.Setenv("QX_API_BASE_URL", "http://localhost:3000/api/v1")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.AgentToken != "env-token" {
		t.Fatalf("token: %q", cfg.AgentToken)
	}
}

func TestLoadMissingToken(t *testing.T) {
	t.Setenv("QX_AGENT_CONFIG", filepath.Join(t.TempDir(), "missing.toml"))
	t.Setenv("QX_AGENT_TOKEN", "")

	_, err := Load()
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
	t.Setenv("QX_AGENT_CONFIG", path)
	t.Setenv("QX_AGENT_TOKEN", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected parse error")
	}
}
