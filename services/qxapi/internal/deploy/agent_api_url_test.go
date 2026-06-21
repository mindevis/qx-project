package deploy

import (
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
