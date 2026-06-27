package notify

import "testing"

func TestEscapeHelpers(t *testing.T) {
	if escapePS("it's") != "it''s" {
		t.Fatalf("ps escape: %q", escapePS("it's"))
	}
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
