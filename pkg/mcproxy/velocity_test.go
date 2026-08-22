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
