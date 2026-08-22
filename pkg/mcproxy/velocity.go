package mcproxy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

type Backend struct {
	Alias   string
	Address string
}

var aliasPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,62}$`)

func GenerateForwardingSecret() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("forwarding secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func ValidAlias(alias string) bool {
	return aliasPattern.MatchString(strings.TrimSpace(alias))
}

func AliasFromName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '_' || r == '-' || r == ' ':
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-_")
	if out == "" {
		out = "server"
	}
	if len(out) > 63 {
		out = out[:63]
	}
	if !ValidAlias(out) {
		return "server"
	}
	return out
}

func VelocityToml(bind, motd string, backends []Backend, try []string) string {
	bind = strings.TrimSpace(bind)
	if bind == "" {
		bind = "0.0.0.0:25565"
	}
	motd = strings.TrimSpace(motd)
	if motd == "" {
		motd = "Velocity"
	}

	var b strings.Builder
	b.WriteString("config-version = \"2.7\"\n")
	b.WriteString("bind = \"" + escapeTOML(bind) + "\"\n")
	b.WriteString("motd = \"" + escapeTOML(motd) + "\"\n")
	b.WriteString("show-max-players = 500\n")
	b.WriteString("online-mode = true\n")
	b.WriteString("force-key-authentication = true\n")
	b.WriteString("prevent-client-proxy-connections = false\n")
	b.WriteString("player-info-forwarding-mode = \"modern\"\n")
	b.WriteString("forwarding-secret-file = \"forwarding.secret\"\n")
	b.WriteString("announce-forge = false\n")
	b.WriteString("kick-existing-players = false\n")
	b.WriteString("ping-passthrough = \"DISABLED\"\n")
	b.WriteString("enable-player-address-logging = true\n\n")
	b.WriteString("[servers]\n")
	for _, backend := range backends {
		alias := strings.TrimSpace(backend.Alias)
		addr := strings.TrimSpace(backend.Address)
		if alias == "" || addr == "" {
			continue
		}
		b.WriteString(alias + " = \"" + escapeTOML(addr) + "\"\n")
	}
	b.WriteString("try = [")
	first := true
	for _, alias := range try {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}
		if !first {
			b.WriteString(", ")
		}
		first = false
		b.WriteString("\"" + escapeTOML(alias) + "\"")
	}
	b.WriteString("]\n\n")
	b.WriteString("[forced-hosts]\n\n")
	b.WriteString("[advanced]\n")
	b.WriteString("compression-threshold = 256\n")
	b.WriteString("compression-level = -1\n")
	b.WriteString("login-ratelimit = 3000\n")
	b.WriteString("connection-timeout = 5000\n")
	b.WriteString("read-timeout = 30000\n")
	b.WriteString("haproxy-protocol = false\n")
	b.WriteString("tcp-fast-open = false\n")
	b.WriteString("bungee-plugin-message-channel = true\n")
	b.WriteString("show-ping-requests = false\n")
	b.WriteString("failover-on-unexpected-server-disconnect = true\n")
	b.WriteString("announce-proxy-commands = true\n")
	b.WriteString("log-command-executions = false\n")
	b.WriteString("log-player-connections = true\n")
	b.WriteString("accepts-transfers = false\n\n")
	b.WriteString("[query]\n")
	b.WriteString("enabled = false\n")
	b.WriteString("port = 25577\n")
	b.WriteString("map = \"Velocity\"\n")
	b.WriteString("show-plugins = false\n")
	return b.String()
}

func PaperVelocityYAML(secret string) string {
	secret = strings.TrimSpace(secret)
	return "# QX Velocity forwarding\n" +
		"proxies:\n" +
		"  velocity:\n" +
		"    enabled: true\n" +
		"    online-mode: true\n" +
		"    secret: '" + escapeYAML(secret) + "'\n"
}

func PatchPaperVelocityForwarding(existing, secret string) string {
	secret = strings.TrimSpace(secret)
	existing = strings.ReplaceAll(existing, "\r\n", "\n")
	if strings.TrimSpace(existing) == "" {
		return PaperVelocityYAML(secret)
	}
	idx := strings.Index(existing, "velocity:")
	if idx < 0 {
		trimmed := strings.TrimRight(existing, "\n")
		return trimmed + "\n" + PaperVelocityYAML(secret)
	}
	head := existing[:idx]
	rest := existing[idx:]
	rest = strings.Replace(rest, "enabled: false", "enabled: true", 1)
	secretRe := regexp.MustCompile(`secret:\s*['"][^'"]*['"]|secret:\s*\S+`)
	if loc := secretRe.FindStringIndex(rest); loc != nil {
		rest = rest[:loc[0]] + "secret: '" + escapeYAML(secret) + "'" + rest[loc[1]:]
	} else {
		rest = strings.Replace(rest, "velocity:", "velocity:\n    secret: '"+escapeYAML(secret)+"'", 1)
	}
	return head + rest
}

func escapeTOML(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func escapeYAML(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `''`)
	return s
}
