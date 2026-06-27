package mcmanifest

import "testing"

func TestNeoForgeVersionPrefix(t *testing.T) {
	cases := map[string]string{
		"1.21.1": "21.1.",
		"1.21":   "21.0.",
		"1.20.6": "20.6.",
	}
	for mc, want := range cases {
		if got := NeoForgeVersionPrefix(mc); got != want {
			t.Fatalf("%q: got %q want %q", mc, got, want)
		}
	}
}

func TestNeoForgeMcVersion(t *testing.T) {
	cases := map[string]string{
		"21.1.234":  "1.21.1",
		"21.0.167":  "1.21",
		"20.6.139":  "1.20.6",
		"26.1.2.76": "",
		"bad":       "",
		"":          "",
	}
	for loader, want := range cases {
		if got := NeoForgeMcVersion(loader); got != want {
			t.Fatalf("%q: got %q want %q", loader, got, want)
		}
	}
}

func TestCompareNeoForgeVersions(t *testing.T) {
	if compareNeoForgeVersions("21.1.10", "21.1.9") <= 0 {
		t.Fatal("expected 21.1.10 > 21.1.9")
	}
	if compareNeoForgeVersions("21.1.9", "21.1.9") != 0 {
		t.Fatal("expected equal")
	}
}
