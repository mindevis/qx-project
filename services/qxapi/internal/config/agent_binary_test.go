package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAgentBinaryPathFromEnv(t *testing.T) {
	t.Setenv("QX_AGENT_BINARY_PATH", "/opt/qx/qx-agent")
	got := resolveAgentBinaryPath()
	if got != "/opt/qx/qx-agent" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveAgentBinaryPathFindsExisting(t *testing.T) {
	t.Setenv("QX_AGENT_BINARY_PATH", "")
	dir := t.TempDir()
	bin := filepath.Join(dir, "qx-agent-linux")
	if err := os.WriteFile(bin, []byte{0}, 0o600); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := os.MkdirAll("bin", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("bin/qx-agent-linux", []byte{0}, 0o600); err != nil {
		t.Fatal(err)
	}

	got := resolveAgentBinaryPath()
	if got != "bin/qx-agent-linux" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveAgentBinaryPathDefault(t *testing.T) {
	t.Setenv("QX_AGENT_BINARY_PATH", "")
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(wd) })

	got := resolveAgentBinaryPath()
	if got != agentBinaryCandidates[0] {
		t.Fatalf("got %q", got)
	}
}

func TestResolveAgentBinaryPathEnvFallback(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if err := os.MkdirAll("../../bin", 0o755); err != nil {
		t.Fatal(err)
	}
	bin := "../../bin/qx-agent-linux"
	if err := os.WriteFile(bin, []byte{0}, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(bin) })

	t.Setenv("QX_AGENT_BINARY_PATH", "bin/qx-agent-linux")
	got := resolveAgentBinaryPath()
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
