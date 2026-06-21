package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("API_ADDR", "")
	t.Setenv("DATABASE_DSN", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ACCESS_TOKEN_TTL", "")
	t.Setenv("REFRESH_TOKEN_TTL", "")
	t.Setenv("CORS_ORIGIN", "")
	t.Setenv("GIN_MODE", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("LOG_FORMAT", "")

	cfg := Load()
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

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("API_ADDR", ":4000")
	t.Setenv("DATABASE_DSN", "custom-dsn")
	t.Setenv("JWT_SECRET", "jwt")
	t.Setenv("ACCESS_TOKEN_TTL", "30m")
	t.Setenv("REFRESH_TOKEN_TTL", "120")
	t.Setenv("CORS_ORIGIN", "http://example.com")
	t.Setenv("GIN_MODE", "release")
	t.Setenv("LOG_LEVEL", "warning")
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("QX_PUBLIC_API_URL", "https://api.example.com")
	t.Setenv("QX_AGENT_BINARY_PATH", "/opt/qx/qx-agent")

	cfg := Load()
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
		t.Fatalf("log env: %+v", cfg)
	}
	if cfg.PublicAPIURL != "https://api.example.com" || cfg.AgentBinaryPath != "/opt/qx/qx-agent" {
		t.Fatalf("deploy env: %+v", cfg)
	}
}

func TestDurationEnvInvalid(t *testing.T) {
	t.Setenv("ACCESS_TOKEN_TTL", "not-a-duration")
	cfg := Load()
	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Fatalf("expected fallback, got %v", cfg.AccessTokenTTL)
	}
}
