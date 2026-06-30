package browser

import "testing"

func TestOpenFolderEmptyPath(t *testing.T) {
	if err := OpenFolder(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}
