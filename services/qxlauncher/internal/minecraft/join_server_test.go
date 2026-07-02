package minecraft

import "testing"

func TestNormalizeJoinEndpoint(t *testing.T) {
	host, port := NormalizeJoinEndpoint("play.example.com", 25565)
	if host != "play.example.com" || port != 25565 {
		t.Fatalf("got %q %d", host, port)
	}

	host, port = NormalizeJoinEndpoint("play.example.com:19132", 25565)
	if host != "play.example.com" || port != 19132 {
		t.Fatalf("embedded port: got %q %d", host, port)
	}

	got := JoinServerQuickPlayValue("play.example.com:19132", 25565)
	if got != "play.example.com:19132" {
		t.Fatalf("quick play value: %q", got)
	}
}
