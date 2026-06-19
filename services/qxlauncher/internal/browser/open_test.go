package browser

import "testing"

func TestOpenEmptyURL(t *testing.T) {
	if err := Open(""); err == nil {
		t.Fatal("expected error for empty url")
	}
}
