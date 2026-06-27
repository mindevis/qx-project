package fs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPatchServerProperties(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.properties")
	if err := os.WriteFile(path, []byte("max-players=10\nonline-mode=true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	props, err := ReadServerProperties(dir)
	if err != nil || len(props) != 2 {
		t.Fatalf("read: err=%v props=%+v", err, props)
	}
	if !props[1].Boolean {
		t.Fatalf("expected boolean flag on online-mode")
	}
	if err := PatchServerProperties(dir, map[string]string{"online-mode": "false", "pvp": "true"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !stringsContains(body, "online-mode=false") || !stringsContains(body, "pvp=true") {
		t.Fatalf("patch result: %q", body)
	}
}

func TestSafePathAndListDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mods", "a.jar"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := safePath(dir, "../etc/passwd"); err == nil {
		t.Fatal("expected path escape error")
	}
	entries, err := ListDir(dir, "mods")
	if err != nil || len(entries) != 1 || entries[0].Name != "a.jar" {
		t.Fatalf("list mods: err=%v entries=%+v", err, entries)
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
