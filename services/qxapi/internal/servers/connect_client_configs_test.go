package servers

import "testing"

func TestIsConfigFilePath(t *testing.T) {
	if !isConfigFilePath("client-config/sodium-options.json") {
		t.Fatal("expected json config to be recognized")
	}
	if !isConfigFilePath(`config\fabric-api.toml`) {
		t.Fatal("expected toml config to be recognized")
	}
	if isConfigFilePath("client-config/readme.txt") {
		t.Fatal("expected txt to be skipped")
	}
}

func TestInstanceConfigDestPath(t *testing.T) {
	got := instanceConfigDestPath("client-config/journeymap/client.json")
	if got != "config/journeymap/client.json" {
		t.Fatalf("got %q", got)
	}
	got = instanceConfigDestPath("CLIENT-CONFIG/sodium-options.json")
	if got != "config/sodium-options.json" {
		t.Fatalf("got %q", got)
	}
	got = instanceConfigDestPath("config/fabric-api.toml")
	if got != "config/fabric-api.toml" {
		t.Fatalf("got %q", got)
	}
}
