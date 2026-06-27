package installer

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type ServerPropertiesConfig struct {
	Name         string
	Address      string
	Port         int
	RconPassword string
}

func rconPortFor(gamePort int) int {
	if gamePort <= 0 {
		gamePort = 25565
	}
	if gamePort <= 65535-10000 {
		return gamePort + 10000
	}
	if gamePort > 1 {
		return gamePort - 1
	}
	return 25575
}

func bindAddress(address string) string {
	address = strings.TrimSpace(address)
	if address == "" {
		return ""
	}
	lower := strings.ToLower(address)
	if lower == "localhost" || lower == "127.0.0.1" || lower == "0.0.0.0" {
		return ""
	}
	if ip := net.ParseIP(address); ip != nil && !ip.IsUnspecified() && !ip.IsLoopback() {
		return address
	}
	return ""
}

func ConfigureServerProperties(workDir string, cfg ServerPropertiesConfig) error {
	return configureServerProperties(workDir, cfg)
}

func configureServerProperties(workDir string, cfg ServerPropertiesConfig) error {
	if strings.TrimSpace(workDir) == "" {
		return fmt.Errorf("missing work dir")
	}
	port := cfg.Port
	if port <= 0 {
		port = 25565
	}
	rconPassword := strings.TrimSpace(cfg.RconPassword)
	if rconPassword == "" {
		return fmt.Errorf("missing rcon password")
	}

	updates := map[string]string{
		"motd":           strings.TrimSpace(cfg.Name),
		"server-port":    fmt.Sprintf("%d", port),
		"enable-rcon":    "true",
		"rcon.port":      fmt.Sprintf("%d", rconPortFor(port)),
		"rcon.password":  rconPassword,
	}
	bind := bindAddress(cfg.Address)
	if bind != "" {
		updates["server-ip"] = bind
	}

	path := filepath.Join(workDir, "server.properties")
	removeKeys := []string(nil)
	if bind == "" {
		removeKeys = []string{"server-ip"}
	}
	return writePropertyFile(path, updates, removeKeys)
}

func writePropertyFile(path string, updates map[string]string, removeKeys []string) error {
	remove := make(map[string]struct{}, len(removeKeys))
	for _, key := range removeKeys {
		remove[key] = struct{}{}
	}
	lines := make([]string, 0, 32)
	if data, err := os.ReadFile(path); err == nil {
		lines = strings.Split(string(data), "\n")
	} else if !os.IsNotExist(err) {
		return err
	}

	seen := make(map[string]struct{}, len(updates))
	out := make([]string, 0, len(lines)+len(updates))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			out = append(out, line)
			continue
		}
		key, _, ok := strings.Cut(trimmed, "=")
		if !ok {
			out = append(out, line)
			continue
		}
		key = strings.TrimSpace(key)
		if _, drop := remove[key]; drop {
			continue
		}
		if value, ok := updates[key]; ok {
			out = append(out, fmt.Sprintf("%s=%s", key, value))
			seen[key] = struct{}{}
			continue
		}
		out = append(out, line)
	}
	for key, value := range updates {
		if _, ok := seen[key]; ok {
			continue
		}
		out = append(out, fmt.Sprintf("%s=%s", key, value))
	}
	content := strings.Join(out, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
