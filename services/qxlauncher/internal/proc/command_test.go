package proc

import (
	"context"
	"runtime"
	"testing"
)

func TestCommandHidesConsoleOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows only")
	}
	cmd := Command("cmd", "/c", "exit", "0")
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow {
		t.Fatal("expected HideWindow on Windows")
	}
}

func TestCommandContextHidesConsoleOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows only")
	}
	cmd := CommandContext(context.Background(), "cmd", "/c", "exit", "0")
	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.HideWindow {
		t.Fatal("expected HideWindow on Windows")
	}
}
