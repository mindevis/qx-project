package config

import (
	"errors"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/qxproject/qx/pkg/reporoot"
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
	LogLevel   string `toml:"log_level"`
	LogFormat  string `toml:"log_format"`
}

type Runtime struct {
	APIBaseURL string
	AgentToken string
	ServerID   string
	ServerRoot string
	WSURL      string
	Hostname   string
	DryRun     bool
	LogLevel   string
	LogFormat  string
	ConfigPath string
}

func Load(configPath string) (Runtime, error) {
	for _, path := range candidatePaths(configPath) {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		cfg := Runtime{ConfigPath: path}
		if err := cfg.unmarshalTOML(data); err != nil {
			return Runtime{}, err
		}
		if strings.TrimSpace(cfg.AgentToken) != "" {
			return cfg, nil
		}
	}
	return Runtime{}, ErrMissingAgentToken
}

func candidatePaths(override string) []string {
	var paths []string
	if override != "" {
		paths = append(paths, override)
	}
	if p := reporoot.ConfigFile(".", "agent.toml"); p != "" {
		paths = append(paths, p)
	}
	paths = append(paths, DefaultConfigPath)
	return paths
}

func (c *Runtime) unmarshalTOML(data []byte) error {
	var file File
	if err := toml.Unmarshal(data, &file); err != nil {
		return err
	}
	c.applyFile(file)
	return nil
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
	if f.LogLevel != "" {
		c.LogLevel = f.LogLevel
	}
	if f.LogFormat != "" {
		c.LogFormat = f.LogFormat
	}
}
