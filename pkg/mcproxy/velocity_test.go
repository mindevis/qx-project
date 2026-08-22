package mcproxy

import (
	"strings"
	"testing"
)

func TestAliasFromName(t *testing.T) {
	if got := AliasFromName("Lobby #1"); got != "lobby-1" {
		t.Fatalf("got %q", got)
	}
	if got := AliasFromName("  "); got != "server" {
		t.Fatalf("empty: %q", got)
	}
	if KeepMOTD("qrpg-world-proxy", "qrpg-world-proxy", "Velocity") != "Velocity" {
		t.Fatal("instance name should not stay as motd")
	}
	if KeepMOTD("A Velocity Server", "qrpg-world-proxy", "Velocity") != "A Velocity Server" {
		t.Fatal("custom motd should stay")
	}
	if !ValidAlias("lobby") || ValidAlias("Lobby") || ValidAlias("") {
		t.Fatal("valid alias checks")
	}
}

func TestVelocityToml(t *testing.T) {
	toml := VelocityToml("0.0.0.0:25565", "QX", []Backend{
		{Alias: "lobby", Address: "127.0.0.1:25566"},
		{Alias: "survival", Address: "127.0.0.1:25567"},
	}, []string{"lobby"})
	for _, want := range []string{
		`bind = "0.0.0.0:25565"`,
		`lobby = "127.0.0.1:25566"`,
		`survival = "127.0.0.1:25567"`,
		`try = ["lobby"]`,
		`player-info-forwarding-mode = "modern"`,
	} {
		if !strings.Contains(toml, want) {
			t.Fatalf("missing %q in:\n%s", want, toml)
		}
	}
}

func TestPatchPaperVelocityForwarding(t *testing.T) {
	existing := "proxies:\n  bungee-cord:\n    online-mode: true\n  velocity:\n    enabled: false\n    online-mode: true\n    secret: ''\n"
	got := PatchPaperVelocityForwarding(existing, "abc123")
	if !strings.Contains(got, "enabled: true") {
		t.Fatalf("expected enabled true:\n%s", got)
	}
	if !strings.Contains(got, "secret: 'abc123'") {
		t.Fatalf("expected secret:\n%s", got)
	}
	if strings.Contains(got, "enabled: false") {
		t.Fatalf("left enabled false:\n%s", got)
	}
	empty := PatchPaperVelocityForwarding("", "xyz")
	if !strings.Contains(empty, "secret: 'xyz'") {
		t.Fatalf("empty file:\n%s", empty)
	}
}

func TestGenerateForwardingSecret(t *testing.T) {
	a, err := GenerateForwardingSecret()
	if err != nil || len(a) < 16 {
		t.Fatalf("secret: %q err=%v", a, err)
	}
}

func TestBungeeConfigYAML(t *testing.T) {
	yaml := BungeeConfigYAML("0.0.0.0:25565", "QX", []Backend{
		{Alias: "lobby", Address: "127.0.0.1:25566"},
	}, []string{"lobby"})
	for _, want := range []string{
		"host: 0.0.0.0:25565",
		"ip_forward: true",
		"  lobby:",
		"    address: 127.0.0.1:25566",
		"  - lobby",
	} {
		if !strings.Contains(yaml, want) {
			t.Fatalf("missing %q in:\n%s", want, yaml)
		}
	}
}

func TestPatchSpigotBungeeCord(t *testing.T) {
	got := PatchSpigotBungeeCord("settings:\n  bungeecord: false\n")
	if !strings.Contains(got, "bungeecord: true") || strings.Contains(got, "bungeecord: false") {
		t.Fatalf("got:\n%s", got)
	}
	empty := PatchSpigotBungeeCord("")
	if !strings.Contains(empty, "bungeecord: true") {
		t.Fatalf("empty:\n%s", empty)
	}
}

func TestParseVelocityTomlRoundTrip(t *testing.T) {
	toml := VelocityToml("0.0.0.0:25565", "QX", []Backend{
		{Alias: "lobby", Address: "127.0.0.1:25566"},
		{Alias: "survival", Address: "127.0.0.1:25567"},
	}, []string{"lobby"})
	cfg := ParseVelocityToml(toml)
	if cfg.Bind != "0.0.0.0:25565" || len(cfg.Backends) != 2 || cfg.Backends[0].Alias != "lobby" {
		t.Fatalf("cfg: %+v", cfg)
	}
	if cfg.Backends[1].Address != "127.0.0.1:25567" {
		t.Fatalf("survival: %+v", cfg.Backends[1])
	}
	if len(cfg.Try) != 1 || cfg.Try[0] != "lobby" {
		t.Fatalf("try: %+v", cfg.Try)
	}
}

func TestDiffProxyApplyAndMergeKeepsExisting(t *testing.T) {
	existing := VelocityToml("0.0.0.0:25577", "Custom MOTD", []Backend{
		{Alias: "hub", Address: "127.0.0.1:25566"},
		{Alias: "sky", Address: "127.0.0.1:25570"},
	}, []string{"hub"})
	cfg := ParseVelocityToml(existing)
	changes := DiffProxyApply(cfg, "0.0.0.0:25565", []Backend{
		{Alias: "hub", Address: "127.0.0.1:25567"},
		{Alias: "lobby", Address: "127.0.0.1:25566"},
	}, []string{"lobby"})
	if len(changes) < 3 {
		t.Fatalf("changes: %+v", changes)
	}
	merged := MergeVelocityToml(existing, []Backend{
		{Alias: "hub", Address: "127.0.0.1:25567"},
		{Alias: "lobby", Address: "127.0.0.1:25566"},
	}, []string{"lobby"})
	got := ParseVelocityToml(merged)
	if got.Motd != "Custom MOTD" || got.Bind != "0.0.0.0:25577" {
		t.Fatalf("kept settings: %+v", got)
	}
	if len(got.Try) != 1 || got.Try[0] != "hub" {
		t.Fatalf("try should stay hub: %+v", got.Try)
	}
	byAlias := map[string]string{}
	for _, b := range got.Backends {
		byAlias[b.Alias] = b.Address
	}
	if byAlias["hub"] != "127.0.0.1:25566" {
		t.Fatalf("hub address should stay: %+v", byAlias)
	}
	if byAlias["sky"] != "127.0.0.1:25570" {
		t.Fatalf("extra sky should stay: %+v", byAlias)
	}
	if byAlias["lobby"] != "127.0.0.1:25566" {
		t.Fatalf("lobby should be added: %+v", byAlias)
	}
}

func TestPatchVelocityTomlKeepsAdvancedDropsInstanceBackend(t *testing.T) {
	existing := VelocityToml("0.0.0.0:25565", "My Network", []Backend{
		{Alias: "qrpg-world-proxy", Address: "127.0.0.1:25565"},
	}, []string{"qrpg-world-proxy"})
	got := PatchVelocityToml(existing, "0.0.0.0:25565", "Velocity", []Backend{
		{Alias: "lobby", Address: "127.0.0.1:25566"},
	}, []string{"lobby"})
	if !strings.Contains(got, `motd = "Velocity"`) {
		t.Fatalf("motd:\n%s", got)
	}
	if strings.Contains(got, "qrpg-world-proxy") {
		t.Fatalf("left proxy instance in servers:\n%s", got)
	}
	if !strings.Contains(got, `lobby = "127.0.0.1:25566"`) || !strings.Contains(got, `try = ["lobby"]`) {
		t.Fatalf("servers:\n%s", got)
	}
	if !strings.Contains(got, "[advanced]") {
		t.Fatalf("lost advanced:\n%s", got)
	}
}

func TestParseBungeeConfigRoundTrip(t *testing.T) {
	yaml := BungeeConfigYAML("0.0.0.0:25565", "QX", []Backend{
		{Alias: "lobby", Address: "127.0.0.1:25566"},
		{Alias: "sky", Address: "127.0.0.1:25568"},
	}, []string{"lobby"})
	cfg := ParseBungeeConfig(yaml)
	if cfg.Bind != "0.0.0.0:25565" {
		t.Fatalf("bind: %q", cfg.Bind)
	}
	if len(cfg.Try) != 1 || cfg.Try[0] != "lobby" {
		t.Fatalf("try: %+v", cfg.Try)
	}
	byAlias := map[string]string{}
	for _, b := range cfg.Backends {
		byAlias[b.Alias] = b.Address
	}
	if byAlias["lobby"] != "127.0.0.1:25566" || byAlias["sky"] != "127.0.0.1:25568" {
		t.Fatalf("backends: %+v", cfg.Backends)
	}
}

func TestProxyAddrPort(t *testing.T) {
	if got := ProxyAddrPort("127.0.0.1:25566"); got != 25566 {
		t.Fatalf("got %d", got)
	}
	if got := ProxyAddrPort("[::1]:25565"); got != 25565 {
		t.Fatalf("v6: %d", got)
	}
}
