package notify

import "testing"

func TestEscapeAppleScript(t *testing.T) {
	if escapeAppleScript(`a"b`) != `a\"b` {
		t.Fatalf("apple escape: %q", escapeAppleScript(`a"b`))
	}
}

func TestShowDoesNotPanic(t *testing.T) {
	Show("QX", "test")
}

func TestShowDedupesWithinWindow(t *testing.T) {
	Show("QX", "dup")
	Show("QX", "dup")
}
