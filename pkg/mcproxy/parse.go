package mcproxy

import (
	"net"
	"regexp"
	"strconv"
	"strings"
)

type ProxyConfig struct {
	Bind     string
	Motd     string
	Backends []Backend
	Try      []string
}

var (
	tomlBindRe   = regexp.MustCompile(`(?m)^\s*bind\s*=\s*"([^"]*)"`)
	tomlMotdRe   = regexp.MustCompile(`(?m)^\s*motd\s*=\s*"([^"]*)"`)
	tomlKVRe     = regexp.MustCompile(`^\s*([A-Za-z0-9][A-Za-z0-9_-]*)\s*=\s*"([^"]*)"\s*$`)
	tomlTryRe    = regexp.MustCompile(`(?i)^\s*try\s*=\s*\[(.*)\]\s*$`)
	tomlTryItem  = regexp.MustCompile(`"([^"]*)"`)
	yamlPrioLine = regexp.MustCompile(`^\s*-\s+([A-Za-z0-9][A-Za-z0-9_-]*)\s*$`)
)

func ParseVelocityToml(content string) ProxyConfig {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	cfg := ProxyConfig{}
	if m := tomlBindRe.FindStringSubmatch(content); len(m) == 2 {
		cfg.Bind = strings.TrimSpace(m[1])
	}
	if m := tomlMotdRe.FindStringSubmatch(content); len(m) == 2 {
		cfg.Motd = strings.TrimSpace(m[1])
	}
	section := sectionBody(content, "[servers]")
	if section == "" {
		return cfg
	}
	for _, raw := range strings.Split(section, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if m := tomlTryRe.FindStringSubmatch(line); len(m) == 2 {
			for _, item := range tomlTryItem.FindAllStringSubmatch(m[1], -1) {
				if alias := strings.TrimSpace(item[1]); alias != "" {
					cfg.Try = append(cfg.Try, alias)
				}
			}
			continue
		}
		if m := tomlKVRe.FindStringSubmatch(line); len(m) == 3 {
			cfg.Backends = append(cfg.Backends, Backend{Alias: m[1], Address: strings.TrimSpace(m[2])})
		}
	}
	return cfg
}

func ParseBungeeConfig(content string) ProxyConfig {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	cfg := ProxyConfig{}
	cfg.Try = yamlListAfter(content, "priorities:")
	servers := yamlMapBlock(content, "servers:")
	for alias, body := range servers {
		addr := yamlKeyedValue(body, "address")
		if alias == "" || addr == "" {
			continue
		}
		cfg.Backends = append(cfg.Backends, Backend{Alias: alias, Address: addr})
	}
	if m := regexp.MustCompile(`(?m)^\s*host:\s*(\S+)`).FindStringSubmatch(content); len(m) == 2 {
		cfg.Bind = strings.TrimSpace(m[1])
	}
	if m := regexp.MustCompile(`(?m)^\s*motd:\s*'([^']*)'`).FindStringSubmatch(content); len(m) == 2 {
		cfg.Motd = strings.TrimSpace(m[1])
	}
	return cfg
}

func ProxyAddrPort(addr string) int {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return 0
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		if i := strings.LastIndex(addr, ":"); i >= 0 {
			port = addr[i+1:]
		} else {
			return 0
		}
	}
	n, err := strconv.Atoi(strings.TrimSpace(port))
	if err != nil || n < 1 || n > 65535 {
		return 0
	}
	return n
}

func sectionBody(content, header string) string {
	idx := strings.Index(content, header)
	if idx < 0 {
		return ""
	}
	rest := content[idx+len(header):]
	if next := strings.Index(rest, "\n["); next >= 0 {
		rest = rest[:next]
	}
	return rest
}

func yamlListAfter(content, key string) []string {
	idx := indexYAMLKey(content, key)
	if idx < 0 {
		return nil
	}
	rest := content[idx:]
	if nl := strings.Index(rest, "\n"); nl >= 0 {
		rest = rest[nl+1:]
	} else {
		return nil
	}
	var out []string
	for _, raw := range strings.Split(rest, "\n") {
		if raw == "" {
			continue
		}
		if !strings.HasPrefix(raw, " ") && !strings.HasPrefix(raw, "\t") {
			break
		}
		if m := yamlPrioLine.FindStringSubmatch(raw); len(m) == 2 {
			out = append(out, m[1])
			continue
		}
		if strings.TrimSpace(raw) != "" && !strings.HasPrefix(strings.TrimSpace(raw), "-") && !strings.HasPrefix(strings.TrimSpace(raw), "#") {
			break
		}
	}
	return out
}

func yamlMapBlock(content, key string) map[string]string {
	idx := indexYAMLKey(content, key)
	if idx < 0 {
		return nil
	}
	rest := content[idx:]
	if nl := strings.Index(rest, "\n"); nl >= 0 {
		rest = rest[nl+1:]
	} else {
		return nil
	}
	out := map[string]string{}
	var current string
	var body strings.Builder
	flush := func() {
		if current == "" {
			return
		}
		out[current] = body.String()
		body.Reset()
	}
	for _, raw := range strings.Split(rest, "\n") {
		if raw == "" {
			continue
		}
		if !strings.HasPrefix(raw, " ") && !strings.HasPrefix(raw, "\t") {
			break
		}
		trimmed := strings.TrimSpace(raw)
		indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
		if indent == 2 && strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, " ") {
			flush()
			current = strings.TrimSuffix(trimmed, ":")
			continue
		}
		if current != "" {
			body.WriteString(raw)
			body.WriteByte('\n')
		}
	}
	flush()
	return out
}

func yamlKeyedValue(block, key string) string {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `:\s*(.+?)\s*$`)
	m := re.FindStringSubmatch(block)
	if len(m) != 2 {
		return ""
	}
	v := strings.TrimSpace(m[1])
	v = strings.Trim(v, `"'`)
	return v
}

func indexYAMLKey(content, key string) int {
	re := regexp.MustCompile(`(?m)^[ \t]*` + regexp.QuoteMeta(key))
	loc := re.FindStringIndex(content)
	if loc == nil {
		return -1
	}
	return loc[0]
}
