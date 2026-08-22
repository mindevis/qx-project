package mcproxy

import (
	"strings"
)

type ApplyChange struct {
	Field string `json:"field"`
	Name  string `json:"name,omitempty"`
	From  string `json:"from,omitempty"`
	To    string `json:"to,omitempty"`
}

func DiffProxyApply(existing ProxyConfig, bind string, backends []Backend, try []string) []ApplyChange {
	var out []ApplyChange
	bind = strings.TrimSpace(bind)
	if existing.Bind != "" && bind != "" && !sameTrim(existing.Bind, bind) {
		out = append(out, ApplyChange{Field: "bind", From: strings.TrimSpace(existing.Bind), To: bind})
	}
	have := map[string]Backend{}
	for _, backend := range existing.Backends {
		alias := strings.ToLower(strings.TrimSpace(backend.Alias))
		if alias == "" {
			continue
		}
		have[alias] = backend
	}
	want := map[string]Backend{}
	for _, backend := range backends {
		alias := strings.ToLower(strings.TrimSpace(backend.Alias))
		if alias == "" || strings.TrimSpace(backend.Address) == "" {
			continue
		}
		want[alias] = backend
		old, ok := have[alias]
		if !ok {
			continue
		}
		if !sameTrim(old.Address, backend.Address) {
			out = append(out, ApplyChange{
				Field: "server",
				Name:  strings.TrimSpace(backend.Alias),
				From:  strings.TrimSpace(old.Address),
				To:    strings.TrimSpace(backend.Address),
			})
		}
	}
	for alias, backend := range have {
		if _, ok := want[alias]; ok {
			continue
		}
		out = append(out, ApplyChange{
			Field: "remove",
			Name:  strings.TrimSpace(backend.Alias),
			From:  strings.TrimSpace(backend.Address),
		})
	}
	if len(existing.Try) > 0 && formatTry(existing.Try) != formatTry(try) {
		out = append(out, ApplyChange{Field: "try", From: formatTry(existing.Try), To: formatTry(try)})
	}
	return out
}

func MergeVelocityToml(existing string, backends []Backend, try []string) string {
	cfg := ParseVelocityToml(existing)
	have := map[string]struct{}{}
	merged := make([]Backend, 0, len(cfg.Backends)+len(backends))
	for _, backend := range cfg.Backends {
		alias := strings.ToLower(strings.TrimSpace(backend.Alias))
		if alias == "" || strings.TrimSpace(backend.Address) == "" {
			continue
		}
		if _, ok := have[alias]; ok {
			continue
		}
		have[alias] = struct{}{}
		merged = append(merged, backend)
	}
	for _, backend := range backends {
		alias := strings.ToLower(strings.TrimSpace(backend.Alias))
		if alias == "" || strings.TrimSpace(backend.Address) == "" {
			continue
		}
		if _, ok := have[alias]; ok {
			continue
		}
		have[alias] = struct{}{}
		merged = append(merged, backend)
	}
	tryOut := cfg.Try
	if len(tryOut) == 0 {
		tryOut = try
	}
	return PatchVelocityToml(existing, "", "", merged, tryOut)
}

func formatTry(try []string) string {
	parts := make([]string, 0, len(try))
	for _, alias := range try {
		alias = strings.TrimSpace(alias)
		if alias != "" {
			parts = append(parts, alias)
		}
	}
	return strings.Join(parts, ", ")
}

func sameTrim(a, b string) bool {
	return strings.TrimSpace(a) == strings.TrimSpace(b)
}
