package config

import (
	"os"
	"testing"
)

func chdirTo(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

func TestResolveAgentBinaryPathFromConfig(t *testing.T) {
	got := resolveAgentBinaryPath("/opt/qx/qx-agent")
	if got != "/opt/qx/qx-agent" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveAgentBinaryPathFindsExisting(t *testing.T) {
	dir := t.TempDir()
	chdirTo(t, dir)

	if err := os.MkdirAll("bin", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("bin/qx-agent-linux", []byte{0}, 0o600); err != nil {
		t.Fatal(err)
	}

	got := resolveAgentBinaryPath("")
	if got != "bin/qx-agent-linux" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveAgentBinaryPathDefault(t *testing.T) {
	dir := t.TempDir()
	chdirTo(t, dir)

	got := resolveAgentBinaryPath("")
	if got != agentBinaryCandidates[0] {
		t.Fatalf("got %q", got)
	}
}

func TestResolveAgentBinaryPathConfigFallback(t *testing.T) {
	dir := t.TempDir()
	chdirTo(t, dir)

	if err := os.MkdirAll("../../bin", 0o755); err != nil {
		t.Fatal(err)
	}
	bin := "../../bin/qx-agent-linux"
	if err := os.WriteFile(bin, []byte{0}, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(bin) })

	got := resolveAgentBinaryPath("bin/qx-agent-linux")
	if got != bin {
		t.Fatalf("got %q want %q", got, bin)
	}
}

func TestAgentBinaryAbs(t *testing.T) {
	if AgentBinaryAbs("") != "" {
		t.Fatal("empty path")
	}
	abs := AgentBinaryAbs(".")
	if abs == "" {
		t.Fatal("expected abs path")
	}
}
