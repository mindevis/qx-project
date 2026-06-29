package minecraft

import (
	"strings"
	"testing"
)

func TestSkinServerJVMArgs(t *testing.T) {
	args := SkinServerJVMArgs("https://api.example.com")
	if len(args) != 2 {
		t.Fatalf("args: %v", args)
	}
	if !strings.Contains(args[1], "sessionserver") {
		t.Fatalf("session host: %v", args[1])
	}
}

func TestPrependSkinServerJVMArgs(t *testing.T) {
	in := []string{"-Xmx2G", "-cp", "game.jar", "net.minecraft.client.main.Main"}
	out := PrependSkinServerJVMArgs(in, SkinServerConfig{
		Enabled:  true,
		HostBase: "https://api.example.com",
	})
	if len(out) != len(in)+2 {
		t.Fatalf("len: %d", len(out))
	}
	if !strings.HasPrefix(out[2], "-Dminecraft.api.session.host=") {
		t.Fatalf("prepend order: %v", out)
	}
}
