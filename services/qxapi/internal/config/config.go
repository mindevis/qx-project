package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr            string
	DatabaseDSN     string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	CORSOrigin      string
	SSHMasterKey    string
	PublicAPIURL    string
	AgentBinaryPath string
	SSHDeployDryRun bool
	GinMode         string
	LogLevel        string
	LogFormat       string
}

func Load() Config {
	return Config{
		Addr:            getenv("API_ADDR", ":3000"),
		DatabaseDSN:     getenv("DATABASE_DSN", "qx:qx@tcp(localhost:3306)/qx?charset=utf8mb4&parseTime=True&loc=Local"),
		JWTSecret:       getenv("JWT_SECRET", "dev-secret-change-in-production"),
		AccessTokenTTL:  durationEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: durationEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		CORSOrigin:      getenv("CORS_ORIGIN", "http://localhost:5173"),
		SSHMasterKey:    getenv("SSH_MASTER_KEY", devSSHMasterKey()),
		PublicAPIURL:    getenv("QX_PUBLIC_API_URL", "http://localhost:3000"),
		AgentBinaryPath: getenv("QX_AGENT_BINARY_PATH", ""),
		SSHDeployDryRun: os.Getenv("QX_SSH_DEPLOY_DRY_RUN") == "1",
		GinMode:         getenv("GIN_MODE", "debug"),
		LogLevel:        getenv("LOG_LEVEL", "info"),
		LogFormat:       getenv("LOG_FORMAT", "text"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	if sec, err := strconv.Atoi(raw); err == nil {
		return time.Duration(sec) * time.Second
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d
	}
	return fallback
}

func devSSHMasterKey() string {
	return "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
}
