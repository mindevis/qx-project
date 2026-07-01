package updater

import (
	"context"
	"runtime"
	"testing"
)

func TestApplyUnsupportedOS(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-windows test")
	}
	err := Apply(context.Background(), "http://example.com/qx-launcher.exe", "qx-launcher.exe", nil)
	if err == nil {
		t.Fatal("expected error on non-windows")
	}
}
