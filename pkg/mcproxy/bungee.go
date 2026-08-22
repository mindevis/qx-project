package mcproxy

import (
	"fmt"
	"strings"
)

func BungeeConfigYAML(bind, motd string, backends []Backend, try []string) string {
	bind = strings.TrimSpace(bind)
	if bind == "" {
		bind = "0.0.0.0:25565"
	}
	motd = strings.TrimSpace(motd)
	if motd == "" {
		motd = "BungeeCord"
	}
	priorities := make([]string, 0, len(try))
	for _, alias := range try {
		alias = strings.TrimSpace(alias)
		if alias != "" {
			priorities = append(priorities, alias)
		}
	}
	if len(priorities) == 0 {
		for _, backend := range backends {
			if alias := strings.TrimSpace(backend.Alias); alias != "" {
				priorities = append(priorities, alias)
				break
			}
		}
	}

	var b strings.Builder
	b.WriteString("prevent_proxy_connections: false\n")
	b.WriteString("listeners:\n")
	b.WriteString("- query_port: 25577\n")
	b.WriteString("  motd: '" + escapeYAML(motd) + "'\n")
	b.WriteString("  tab_list: GLOBAL_PING\n")
	b.WriteString("  query_enabled: false\n")
	b.WriteString("  proxy_protocol: false\n")
	b.WriteString("  ping_passthrough: false\n")
	b.WriteString("  bind_local_address: true\n")
	b.WriteString("  host: " + bind + "\n")
	b.WriteString("  max_players: 500\n")
	b.WriteString("  tab_size: 60\n")
	b.WriteString("  force_default_server: false\n")
	b.WriteString("  priorities:\n")
	if len(priorities) == 0 {
		b.WriteString("  - lobby\n")
	} else {
		for _, alias := range priorities {
			b.WriteString("  - " + alias + "\n")
		}
	}
	b.WriteString("remote_ping_cache: -1\n")
	b.WriteString("network_compression_threshold: 256\n")
	b.WriteString("online_mode: true\n")
	b.WriteString("ip_forward: true\n")
	b.WriteString("servers:\n")
	wrote := false
	for _, backend := range backends {
		alias := strings.TrimSpace(backend.Alias)
		addr := strings.TrimSpace(backend.Address)
		if alias == "" || addr == "" {
			continue
		}
		wrote = true
		b.WriteString("  " + alias + ":\n")
		b.WriteString("    motd: '&1" + escapeYAML(alias) + "'\n")
		b.WriteString("    address: " + addr + "\n")
		b.WriteString("    restricted: false\n")
	}
	if !wrote {
		b.WriteString("  lobby:\n")
		b.WriteString("    motd: '&1Lobby'\n")
		b.WriteString("    address: 127.0.0.1:25566\n")
		b.WriteString("    restricted: false\n")
	}
	return b.String()
}

func PatchSpigotBungeeCord(existing string) string {
	existing = strings.ReplaceAll(existing, "\r\n", "\n")
	if strings.TrimSpace(existing) == "" {
		return "settings:\n  bungeecord: true\n"
	}
	if strings.Contains(existing, "bungeecord:") {
		existing = strings.Replace(existing, "bungeecord: false", "bungeecord: true", 1)
		if strings.Contains(existing, "bungeecord: true") {
			return existing
		}
		return strings.Replace(existing, "bungeecord:", "bungeecord: true", 1)
	}
	trimmed := strings.TrimRight(existing, "\n")
	if strings.Contains(existing, "settings:") {
		return trimmed + "\n  bungeecord: true\n"
	}
	return trimmed + "\nsettings:\n  bungeecord: true\n"
}

func PaperBungeeYAML() string {
	return "proxies:\n  bungee-cord:\n    online-mode: true\n"
}

func PatchPaperBungeeForwarding(existing string) string {
	existing = strings.ReplaceAll(existing, "\r\n", "\n")
	if strings.TrimSpace(existing) == "" {
		return PaperBungeeYAML()
	}
	idx := strings.Index(existing, "bungee-cord:")
	if idx < 0 {
		trimmed := strings.TrimRight(existing, "\n")
		return trimmed + "\n" + PaperBungeeYAML()
	}
	head := existing[:idx]
	rest := existing[idx:]
	rest = strings.Replace(rest, "online-mode: false", "online-mode: true", 1)
	return head + rest
}

func BungeeCordJarURL(build string) string {
	build = strings.TrimSpace(build)
	return fmt.Sprintf("https://ci.md-5.net/job/BungeeCord/%s/artifact/bootstrap/target/BungeeCord.jar", build)
}
