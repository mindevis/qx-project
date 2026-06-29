package config

import (
	"os"
	"strconv"
	"strings"
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
	MojangClientID     string `toml:"mojang_client_id"`
	MojangClientSecret string `toml:"mojang_client_secret"`
	MojangRedirectURI  string `toml:"mojang_oauth_redirect_uri"`
	CurseForgeAPIKey   string `toml:"curseforge_api_key"`
	ModrinthUserAgent  string `toml:"modrinth_user_agent"`
	CosmeticsDataDir      string `toml:"cosmetics_data_dir"`
	SkinServerPublicURL   string `toml:"skin_server_public_url"`
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
	PublicAPIURL       string
	AgentBinaryPath    string
	MojangClientID     string
	MojangClientSecret string
	MojangRedirectURI  string
	CurseForgeAPIKey   string
	ModrinthUserAgent  string
	CosmeticsDataDir      string
	SkinServerPublicURL   string
	GinMode            string
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
	cfg.applyEnv()
	cfg.AgentBinaryPath = resolveAgentBinaryPath(cfg.AgentBinaryPath)
	return cfg
}

// applyEnv overrides config from process environment (prod Docker Compose).
// Env wins over qxapi.toml so secrets stay out of images.
func (c *Config) applyEnv() {
	if v := os.Getenv("API_ADDR"); v != "" {
		c.Addr = v
	}
	if v := os.Getenv("GIN_MODE"); v != "" {
		c.GinMode = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		c.LogFormat = v
	}
	if v := os.Getenv("DATABASE_DSN"); v != "" {
		c.DatabaseDSN = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		c.JWTSecret = v
	}
	if v := os.Getenv("ACCESS_TOKEN_TTL"); v != "" {
		if d := parseDuration(v, c.AccessTokenTTL); d > 0 {
			c.AccessTokenTTL = d
		}
	}
	if v := os.Getenv("REFRESH_TOKEN_TTL"); v != "" {
		if d := parseDuration(v, c.RefreshTokenTTL); d > 0 {
			c.RefreshTokenTTL = d
		}
	}
	if v := os.Getenv("CORS_ORIGIN"); v != "" {
		c.CORSOrigin = v
	}
	if v := os.Getenv("SSH_MASTER_KEY"); v != "" {
		c.SSHMasterKey = v
	}
	if v := os.Getenv("QX_PUBLIC_API_URL"); v != "" {
		c.PublicAPIURL = v
	}
	if v := os.Getenv("QX_AGENT_BINARY_PATH"); v != "" {
		c.AgentBinaryPath = v
	}
	if v := os.Getenv("CURSEFORGE_API_KEY"); v != "" {
		c.CurseForgeAPIKey = v
	}
	if v := os.Getenv("MODRINTH_USER_AGENT"); v != "" {
		c.ModrinthUserAgent = v
	}
	if v := os.Getenv("COSMETICS_DATA_DIR"); v != "" {
		c.CosmeticsDataDir = v
	}
	if v := os.Getenv("SKIN_SERVER_PUBLIC_URL"); v != "" {
		c.SkinServerPublicURL = v
	}
}

func defaults() Config {
	return Config{
		Addr:            ":3000",
		DatabaseDSN:     "qx:qx@tcp(localhost:3306)/qx?charset=utf8mb4&parseTime=True&loc=Local",
		JWTSecret:       "dev-secret-change-in-production",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		CORSOrigin:      "http://localhost:5173",
		SSHMasterKey:    devSSHMasterKey(),
		PublicAPIURL:    "http://localhost:3000",
		CosmeticsDataDir: "data/cosmetics",
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
	if f.MojangClientID != "" {
		c.MojangClientID = f.MojangClientID
	}
	if f.MojangClientSecret != "" {
		c.MojangClientSecret = f.MojangClientSecret
	}
	if f.MojangRedirectURI != "" {
		c.MojangRedirectURI = f.MojangRedirectURI
	}
	if f.CurseForgeAPIKey != "" {
		c.CurseForgeAPIKey = f.CurseForgeAPIKey
	}
	if f.ModrinthUserAgent != "" {
		c.ModrinthUserAgent = f.ModrinthUserAgent
	}
	if f.CosmeticsDataDir != "" {
		c.CosmeticsDataDir = f.CosmeticsDataDir
	}
	if f.SkinServerPublicURL != "" {
		c.SkinServerPublicURL = f.SkinServerPublicURL
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

func (c Config) ResolvedMojangRedirectURI() string {
	if uri := strings.TrimSpace(c.MojangRedirectURI); uri != "" {
		return uri
	}
	return strings.TrimRight(strings.TrimSpace(c.PublicAPIURL), "/") + "/api/v1/mojang/oauth/callback"
}

func (c Config) ResolvedSkinServerPublicURL() string {
	if uri := strings.TrimSpace(c.SkinServerPublicURL); uri != "" {
		return uri
	}
	return strings.TrimSpace(c.PublicAPIURL)
}
