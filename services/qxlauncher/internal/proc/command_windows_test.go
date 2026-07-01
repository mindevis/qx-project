//go:build windows

package proc

import (
	"context"
	"testing"
)

func TestCommandHidesConsoleOnWindows(t *testing.T) {
	cmd := Command("cmd", "/c", "exit", "0")
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow {
		t.Fatal("expected HideWindow on Windows")
	}
}

func TestCommandContextHidesConsoleOnWindows(t *testing.T) {
	cmd := CommandContext(context.Background(), "cmd", "/c", "exit", "0")
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow {
		t.Fatal("expected HideWindow on Windows")
	}
}
