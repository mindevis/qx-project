package config

import (
	"errors"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const DefaultConfigPath = "/etc/qx-agent/agent.toml"

var ErrMissingAgentToken = errors.New("agent_token is required")

type File struct {
	APIBaseURL string `toml:"api_base_url"`
	AgentToken string `toml:"agent_token"`
	ServerID   string `toml:"server_id"`
	ServerRoot string `toml:"server_root"`
	WSURL      string `toml:"ws_url"`
	Hostname   string `toml:"hostname"`
	DryRun     bool   `toml:"dry_run"`
}

type Runtime struct {
	APIBaseURL string
	AgentToken string
	ServerID   string
	ServerRoot string
	WSURL      string
	Hostname   string
	DryRun     bool
	ConfigPath string
}

func Load() (Runtime, error) {
	path := os.Getenv("QX_AGENT_CONFIG")
	if path == "" {
		path = DefaultConfigPath
	}

	cfg := Runtime{ConfigPath: path}
	if data, err := os.ReadFile(path); err == nil {
		var file File
		if err := toml.Unmarshal(data, &file); err != nil {
			return Runtime{}, err
		}
		cfg.applyFile(file)
	}

	cfg.applyEnv()
	if strings.TrimSpace(cfg.AgentToken) == "" {
		return Runtime{}, ErrMissingAgentToken
	}
	return cfg, nil
}

func (c *Runtime) applyFile(f File) {
	if f.APIBaseURL != "" {
		c.APIBaseURL = f.APIBaseURL
	}
	if f.AgentToken != "" {
		c.AgentToken = f.AgentToken
	}
	if f.ServerID != "" {
		c.ServerID = f.ServerID
	}
	if f.ServerRoot != "" {
		c.ServerRoot = f.ServerRoot
	}
	if f.WSURL != "" {
		c.WSURL = f.WSURL
	}
	if f.Hostname != "" {
		c.Hostname = f.Hostname
	}
	if f.DryRun {
		c.DryRun = true
	}
}

func (c *Runtime) applyEnv() {
	if v := os.Getenv("QX_API_BASE_URL"); v != "" {
		c.APIBaseURL = v
	}
	if v := os.Getenv("QX_AGENT_TOKEN"); v != "" {
		c.AgentToken = v
	}
	if v := os.Getenv("QX_SERVER_ID"); v != "" {
		c.ServerID = v
	}
	if v := os.Getenv("QX_SERVER_ROOT"); v != "" {
		c.ServerRoot = v
	}
	if v := os.Getenv("QX_AGENT_WS_URL"); v != "" {
		c.WSURL = v
	}
	if v := os.Getenv("QX_AGENT_HOSTNAME"); v != "" {
		c.Hostname = v
	}
	if os.Getenv("QX_AGENT_DRY_RUN") == "1" {
		c.DryRun = true
	}
}
