package config

import (
	"os"
	"strconv"
	"time"

	"github.com/pelletier/go-toml/v2"

	"github.com/qxproject/qx/pkg/reporoot"
)

const configFileName = "qxapi.toml"

type file struct {
	Addr             string `toml:"addr"`
	DatabaseDSN      string `toml:"database_dsn"`
	JWTSecret        string `toml:"jwt_secret"`
	AccessTokenTTL   string `toml:"access_token_ttl"`
	RefreshTokenTTL  string `toml:"refresh_token_ttl"`
	CORSOrigin       string `toml:"cors_origin"`
	SSHMasterKey     string `toml:"ssh_master_key"`
	PublicAPIURL     string `toml:"public_api_url"`
	AgentBinaryPath  string `toml:"agent_binary_path"`
	GinMode          string `toml:"gin_mode"`
	LogLevel         string `toml:"log_level"`
	LogFormat        string `toml:"log_format"`
}

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
	GinMode         string
	LogLevel        string
	LogFormat       string
	ConfigPath      string
}

func Load() Config {
	cfg := defaults()
	if path := reporoot.ConfigFile(".", configFileName); path != "" {
		cfg.ConfigPath = path
		if data, err := os.ReadFile(path); err == nil {
			var f file
			if err := toml.Unmarshal(data, &f); err == nil {
				cfg.applyFile(f)
			}
		}
	}
	cfg.AgentBinaryPath = resolveAgentBinaryPath(cfg.AgentBinaryPath)
	return cfg
}

func defaults() Config {
	return Config{
		Addr:            ":3000",
		DatabaseDSN:     "qx:qx@tcp(localhost:3306)/qx?charset=utf8mb4&parseTime=True&loc=Local",
		JWTSecret:       "dev-secret-change-in-production",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		CORSOrigin:      "http://localhost:5173",
		SSHMasterKey:    devSSHMasterKey(),
		PublicAPIURL:    "http://localhost:3000",
		GinMode:         "debug",
		LogLevel:        "info",
		LogFormat:       "text",
	}
}

func (c *Config) applyFile(f file) {
	if f.Addr != "" {
		c.Addr = f.Addr
	}
	if f.DatabaseDSN != "" {
		c.DatabaseDSN = f.DatabaseDSN
	}
	if f.JWTSecret != "" {
		c.JWTSecret = f.JWTSecret
	}
	if f.AccessTokenTTL != "" {
		if d := parseDuration(f.AccessTokenTTL, c.AccessTokenTTL); d > 0 {
			c.AccessTokenTTL = d
		}
	}
	if f.RefreshTokenTTL != "" {
		if d := parseDuration(f.RefreshTokenTTL, c.RefreshTokenTTL); d > 0 {
			c.RefreshTokenTTL = d
		}
	}
	if f.CORSOrigin != "" {
		c.CORSOrigin = f.CORSOrigin
	}
	if f.SSHMasterKey != "" {
		c.SSHMasterKey = f.SSHMasterKey
	}
	if f.PublicAPIURL != "" {
		c.PublicAPIURL = f.PublicAPIURL
	}
	if f.AgentBinaryPath != "" {
		c.AgentBinaryPath = f.AgentBinaryPath
	}
	if f.GinMode != "" {
		c.GinMode = f.GinMode
	}
	if f.LogLevel != "" {
		c.LogLevel = f.LogLevel
	}
	if f.LogFormat != "" {
		c.LogFormat = f.LogFormat
	}
}

func parseDuration(raw string, fallback time.Duration) time.Duration {
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
