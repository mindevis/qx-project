package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigureServerProperties(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "server.properties"), []byte("max-players=10\nserver-port=25565\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := configureServerProperties(dir, ServerPropertiesConfig{
		Name:         "qRPG",
		Address:      "203.0.113.10",
		Port:         25566,
		RconPassword: "secret-pass",
	}); err != nil {
		t.Fatalf("configure: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "server.properties"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{
		"motd=qRPG",
		"server-port=25566",
		"server-ip=203.0.113.10",
		"enable-rcon=true",
		"rcon.port=35566",
		"rcon.password=secret-pass",
		"max-players=10",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("missing %q in:\n%s", want, content)
		}
	}
}

func TestBindAddressSkipsLocalhost(t *testing.T) {
	if got := bindAddress("localhost"); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := bindAddress("203.0.113.5"); got != "203.0.113.5" {
		t.Fatalf("got %q", got)
	}
}

func TestRconPortFor(t *testing.T) {
	if got := rconPortFor(25565); got != 35565 {
		t.Fatalf("got %d", got)
	}
}
