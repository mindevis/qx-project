package main

import (
	"os"
	"testing"
)

func TestMainExitOnError(t *testing.T) {
	code := 0
	exit = func(c int) { code = c }
	t.Cleanup(func() { exit = os.Exit })

	main()
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
}
