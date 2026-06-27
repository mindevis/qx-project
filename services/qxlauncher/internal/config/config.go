package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"

	"github.com/qxproject/qx/pkg/reporoot"
)

const (
	defaultLauncherTOML = "launcher.toml"
	// DefaultDataDirName is the launcher config and cache root under the user home directory.
	DefaultDataDirName = ".qxlauncher"
)

// UserDataDir returns ~/.qxlauncher (or platform equivalent).
func UserDataDir(home string) string {
	return filepath.Join(home, DefaultDataDirName)
}

type file struct {
	APIBaseURL       string `toml:"api_base_url"`
	WebBaseURL       string `toml:"web_base_url"`
	DeviceTokenPath  string `toml:"device_token_path"`
	LinkMaxPolls     int    `toml:"link_max_polls"`
	SkipTray         bool   `toml:"skip_tray"`
	LaunchDryRun     bool   `toml:"launch_dry_run"`
	DeviceID         string `toml:"device_id"`
	Email            string `toml:"email"`
	Password         string `toml:"password"`
	LogLevel         string `toml:"log_level"`
	LogFormat        string `toml:"log_format"`
	SkipJavaDownload bool   `toml:"skip_java_download"`
	JavaPath         string `toml:"java_path"`
}

type Config struct {
	APIBaseURL       string
	WebBaseURL       string
	DeviceTokenPath  string
	LinkMaxPolls     int
	SkipTray         bool
	LaunchDryRun     bool
	DeviceID         string
	Email            string
	Password         string
	LogLevel         string
	LogFormat        string
	SkipJavaDownload bool
	JavaPath         string
	ConfigPath       string
}

func Load() Config {
	cfg := defaults()
	userPath := userLauncherConfigPath()
	if userPath != "" {
		cfg.applyFile(loadTOML(userPath))
	}
	repoPath := reporoot.ConfigFile(".", defaultLauncherTOML)
	if repoPath != "" {
		cfg.applyFile(loadTOML(repoPath))
		cfg.ConfigPath = repoPath
	} else if userPath != "" {
		cfg.ConfigPath = userPath
	}
	return cfg
}

func defaults() Config {
	home, _ := os.UserHomeDir()
	tokenPath := filepath.Join(UserDataDir(home), "device_token")
	return Config{
		APIBaseURL:      "http://localhost:3000/api/v1",
		WebBaseURL:      "http://localhost:5173",
		DeviceTokenPath: tokenPath,
		LinkMaxPolls:    60,
		LogLevel:        "info",
		LogFormat:       "text",
	}
}

func userLauncherConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(UserDataDir(home), defaultLauncherTOML)
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}

func loadTOML(path string) file {
	if path == "" {
		return file{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return file{}
	}
	var f file
	if err := toml.Unmarshal(data, &f); err != nil {
		return file{}
	}
	return f
}

func (c *Config) applyFile(f file) {
	if f.APIBaseURL != "" {
		c.APIBaseURL = f.APIBaseURL
	}
	if f.WebBaseURL != "" {
		c.WebBaseURL = f.WebBaseURL
	}
	if f.DeviceTokenPath != "" {
		c.DeviceTokenPath = f.DeviceTokenPath
	}
	if f.LinkMaxPolls > 0 {
		c.LinkMaxPolls = f.LinkMaxPolls
	}
	if f.SkipTray {
		c.SkipTray = true
	}
	if f.LaunchDryRun {
		c.LaunchDryRun = true
	}
	if f.DeviceID != "" {
		c.DeviceID = f.DeviceID
	}
	if f.Email != "" {
		c.Email = f.Email
	}
	if f.Password != "" {
		c.Password = f.Password
	}
	if f.LogLevel != "" {
		c.LogLevel = f.LogLevel
	}
	if f.LogFormat != "" {
		c.LogFormat = f.LogFormat
	}
	if f.SkipJavaDownload {
		c.SkipJavaDownload = true
	}
	if f.JavaPath != "" {
		c.JavaPath = f.JavaPath
	}
}
