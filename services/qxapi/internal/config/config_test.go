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
	if cfg.AccessTokenTTL != time.Hour {
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
agent_binary_path = "/opt/qxsystem/agent/qx-agent"
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
	if cfg.PublicAPIURL != "https://api.example.com" || cfg.AgentBinaryPath != "/opt/qxsystem/agent/qx-agent" {
		t.Fatalf("deploy: %+v", cfg)
	}
}

func TestLoadFromEnv(t *testing.T) {
	root := t.TempDir()
	writeRepo(t, root)
	t.Setenv("JWT_SECRET", "from-env")
	t.Setenv("DATABASE_DSN", "env-dsn")
	t.Setenv("QX_PUBLIC_API_URL", "https://prod.example.com")

	cfg := config.Load()
	if cfg.JWTSecret != "from-env" || cfg.DatabaseDSN != "env-dsn" {
		t.Fatalf("env: %+v", cfg)
	}
	if cfg.PublicAPIURL != "https://prod.example.com" {
		t.Fatalf("public url: %s", cfg.PublicAPIURL)
	}
}

func TestResolvedMojangRedirectURI(t *testing.T) {
	t.Run("from public api url", func(t *testing.T) {
		cfg := config.Config{PublicAPIURL: "https://mc.qx-dev.ru"}
		got := cfg.ResolvedMojangRedirectURI()
		want := "https://mc.qx-dev.ru/api/v1/mojang/oauth/callback"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})
	t.Run("trims trailing slash on public api url", func(t *testing.T) {
		cfg := config.Config{PublicAPIURL: "https://mc.qx-dev.ru/"}
		got := cfg.ResolvedMojangRedirectURI()
		want := "https://mc.qx-dev.ru/api/v1/mojang/oauth/callback"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})
	t.Run("explicit override wins", func(t *testing.T) {
		cfg := config.Config{
			PublicAPIURL:      "https://mc.qx-dev.ru",
			MojangRedirectURI: "https://custom.example/callback",
		}
		if got := cfg.ResolvedMojangRedirectURI(); got != "https://custom.example/callback" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestLoadMojangFromEnv(t *testing.T) {
	root := t.TempDir()
	writeRepo(t, root)
	t.Setenv("MOJANG_CLIENT_ID", "azure-app-id")
	t.Setenv("MOJANG_CLIENT_SECRET", "azure-secret")
	t.Setenv("MOJANG_OAUTH_REDIRECT_URI", "https://mc.qx-dev.ru/api/v1/mojang/oauth/callback")
	t.Setenv("QX_PUBLIC_API_URL", "https://mc.qx-dev.ru")

	cfg := config.Load()
	if cfg.MojangClientID != "azure-app-id" || cfg.MojangClientSecret != "azure-secret" {
		t.Fatalf("mojang creds: %+v", cfg)
	}
	if cfg.ResolvedMojangRedirectURI() != "https://mc.qx-dev.ru/api/v1/mojang/oauth/callback" {
		t.Fatalf("redirect: %s", cfg.ResolvedMojangRedirectURI())
	}
}

func TestLoadCurseForgeFromEnv(t *testing.T) {
	root := t.TempDir()
	writeRepo(t, root)
	if err := os.WriteFile(filepath.Join(root, "qxapi.toml"), []byte("curseforge_api_key = \"from-toml\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CURSEFORGE_API_KEY", "from-env")

	cfg := config.Load()
	if cfg.CurseForgeAPIKey != "from-env" {
		t.Fatalf("env should override toml: got %q", cfg.CurseForgeAPIKey)
	}
}

func TestLoadInvalidDurationFallback(t *testing.T) {
	root := t.TempDir()
	writeRepo(t, root)
	if err := os.WriteFile(filepath.Join(root, "qxapi.toml"), []byte("access_token_ttl = \"not-a-duration\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Load()
	if cfg.AccessTokenTTL != time.Hour {
		t.Fatalf("expected fallback, got %v", cfg.AccessTokenTTL)
	}
}
