package deploy

import (
	"path/filepath"
	"testing"
)

func TestKnownHostsPathOverride(t *testing.T) {
	t.Setenv("QX_SSH_KNOWN_HOSTS", "/data/known_hosts")
	if got := knownHostsPath(); got != "/data/known_hosts" {
		t.Fatalf("got %q", got)
	}
}

func TestNewHostKeyCallbackCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "known_hosts")
	t.Setenv("QX_SSH_KNOWN_HOSTS", path)
	cb, err := newHostKeyCallback()
	if err != nil {
		t.Fatal(err)
	}
	if cb == nil {
		t.Fatal("nil callback")
	}
}
