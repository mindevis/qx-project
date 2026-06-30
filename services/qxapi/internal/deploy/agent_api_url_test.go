package deploy

import (
	"strings"
	"testing"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func TestAgentAPIURLDevVPS(t *testing.T) {
	got := agentAPIURL("http://localhost:3000", models.SSHCredential{Host: "localhost"})
	want := "http://host.docker.internal:3000"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAgentAPIURLDevVPS127(t *testing.T) {
	got := agentAPIURL("http://127.0.0.1:3000", models.SSHCredential{Host: "127.0.0.1", Port: 2222})
	if got != "http://host.docker.internal:3000" {
		t.Fatalf("got %q", got)
	}
}

func TestAgentAPIURLProductionUnchanged(t *testing.T) {
	got := agentAPIURL("https://api.qx.example.com", models.SSHCredential{Host: "203.0.113.10"})
	want := "https://api.qx.example.com"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAgentAPIURLLocalAPIRemoteSSH(t *testing.T) {
	got := agentAPIURL("http://localhost:3000", models.SSHCredential{Host: "203.0.113.10"})
	if got != "http://localhost:3000" {
		t.Fatalf("got %q", got)
	}
}

func TestAgentAPIURLEmptyDefaultsLocalhost(t *testing.T) {
	got := agentAPIURL("", models.SSHCredential{Host: "localhost"})
	if got != "http://host.docker.internal:3000" {
		t.Fatalf("got %q", got)
	}
}

func TestAgentAPIURLHostnameMatchNotCoLocated(t *testing.T) {
	got := agentAPIURL("https://mc.qx-dev.ru", models.SSHCredential{Host: "mc.qx-dev.ru"})
	want := "https://mc.qx-dev.ru"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestIsCoLocatedAPIHostnameMatchOnly(t *testing.T) {
	if isCoLocatedAPI("mc.qx-dev.ru", "mc.qx-dev.ru") {
		t.Fatal("hostname string match alone must not imply co-located")
	}
}

func TestIsCoLocatedAPIRemoteHost(t *testing.T) {
	if isCoLocatedAPI("203.0.113.10", "api.qx.example.com") {
		t.Fatal("expected remote host not to match unrelated API hostname")
	}
}

func TestIsCoLocatedAPILocalhostSSH(t *testing.T) {
	if isCoLocatedAPI("localhost", "mc.qx-dev.ru") {
		t.Fatal("localhost SSH should not be treated as co-located prod")
	}
}

func TestNeedsAPIHostsOverrideLoopback(t *testing.T) {
	if !needsAPIHostsOverride("https://mc.qx-dev.ru", models.SSHCredential{Host: "203.0.113.10"}, remoteAPIResolution{
		ResolvedIP: "127.0.1.1",
	}) {
		t.Fatal("expected hosts override when API hostname resolves to loopback on target")
	}
}

func TestNeedsAPIHostsOverrideHostnameCollision(t *testing.T) {
	if !needsAPIHostsOverride("https://mc.qx-dev.ru", models.SSHCredential{Host: "203.0.113.10"}, remoteAPIResolution{
		ResolvedIP:     "203.0.113.99",
		RemoteHostname: "mc.qx-dev.ru",
	}) {
		t.Fatal("expected hosts override when dedicated server hostname matches API hostname")
	}
}

func TestNeedsAPIHostsOverrideRemoteOK(t *testing.T) {
	if needsAPIHostsOverride("https://api.qx.example.com", models.SSHCredential{Host: "203.0.113.10"}, remoteAPIResolution{
		ResolvedIP:     "198.51.100.1",
		RemoteHostname: "game-1.internal",
	}) {
		t.Fatal("expected no override for normal remote resolution")
	}
}

func TestAPIHostsOverrideScriptUsesEtcHosts(t *testing.T) {
	script := apiHostsOverrideScript("https://mc.qx-dev.ru", models.SSHCredential{Host: "203.0.113.10"}, remoteAPIResolution{
		ResolvedIP: "127.0.1.1",
	})
	if script == "" {
		t.Fatal("expected hosts override script")
	}
	if !strings.Contains(script, "/etc/hosts") {
		t.Fatalf("expected /etc/hosts edit, got: %s", script)
	}
	if strings.Contains(script, "/etc/hosts.d") {
		t.Fatal("must not use /etc/hosts.d — glibc ignores it")
	}
}

func TestIsLoopbackIP(t *testing.T) {
	if !isLoopbackIP("127.0.1.1") || !isLoopbackIP("127.0.0.1") {
		t.Fatal("expected loopback")
	}
	if isLoopbackIP("203.0.113.10") {
		t.Fatal("expected public IP not loopback")
	}
}
