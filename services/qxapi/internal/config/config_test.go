package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qxproject/qx/services/qxapi/internal/config"
)

func chdirRepo(t *testing.T, root string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
}

func writeRepo(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "services", "qxapi")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	chdirRepo(t, sub)
}

func TestLoadDefaults(t *testing.T) {
	root := t.TempDir()
	writeRepo(t, root)

	cfg := config.Load()
	if cfg.Addr != ":3000" {
		t.Fatalf("addr: %s", cfg.Addr)
	}
	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Fatalf("access ttl: %v", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 7*24*time.Hour {
		t.Fatalf("refresh ttl: %v", cfg.RefreshTokenTTL)
	}
	if cfg.LogLevel != "info" || cfg.LogFormat != "text" {
		t.Fatalf("log defaults: %+v", cfg)
	}
}

func TestLoadFromTOML(t *testing.T) {
	root := t.TempDir()
	writeRepo(t, root)
	toml := `addr = ":4000"
database_dsn = "custom-dsn"
jwt_secret = "jwt"
access_token_ttl = "30m"
refresh_token_ttl = "120"
cors_origin = "http://example.com"
gin_mode = "release"
log_level = "warning"
log_format = "json"
public_api_url = "https://api.example.com"
agent_binary_path = "/opt/qx/qx-agent"
`
	if err := os.WriteFile(filepath.Join(root, "qxapi.toml"), []byte(toml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := config.Load()
	if cfg.Addr != ":4000" || cfg.DatabaseDSN != "custom-dsn" || cfg.JWTSecret != "jwt" {
		t.Fatalf("cfg: %+v", cfg)
	}
	if cfg.AccessTokenTTL != 30*time.Minute {
		t.Fatalf("access: %v", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 120*time.Second {
		t.Fatalf("refresh: %v", cfg.RefreshTokenTTL)
	}
	if cfg.CORSOrigin != "http://example.com" || cfg.GinMode != "release" {
		t.Fatalf("cors/gin: %+v", cfg)
	}
	if cfg.LogLevel != "warning" || cfg.LogFormat != "json" {
		t.Fatalf("log: %+v", cfg)
	}
	if cfg.PublicAPIURL != "https://api.example.com" || cfg.AgentBinaryPath != "/opt/qx/qx-agent" {
		t.Fatalf("deploy: %+v", cfg)
	}
}

func TestLoadInvalidDurationFallback(t *testing.T) {
	root := t.TempDir()
	writeRepo(t, root)
	if err := os.WriteFile(filepath.Join(root, "qxapi.toml"), []byte("access_token_ttl = \"not-a-duration\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Load()
	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Fatalf("expected fallback, got %v", cfg.AccessTokenTTL)
	}
}
