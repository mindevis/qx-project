package servers

import "testing"

func TestGenerateRconPassword(t *testing.T) {
	pwd, err := generateRconPassword()
	if err != nil {
		t.Fatal(err)
	}
	if len(pwd) != 24 {
		t.Fatalf("expected 24 hex chars, got %d", len(pwd))
	}
}

func TestRconPortFor(t *testing.T) {
	if got := rconPortFor(25566); got != 35566 {
		t.Fatalf("got %d", got)
	}
}
