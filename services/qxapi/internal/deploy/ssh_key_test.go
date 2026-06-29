package deploy

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestParseSSHSignerWithPassphrase(t *testing.T) {
	passphrase := "secret-pass"
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKeyWithPassphrase(priv, "test", []byte(passphrase))
	if err != nil {
		t.Fatal(err)
	}
	pemBytes := pem.EncodeToMemory(block)

	_, err = parseSSHSigner(pemBytes, "")
	if err == nil {
		t.Fatal("expected error without passphrase")
	}

	_, err = parseSSHSigner(pemBytes, "wrong-pass")
	if err == nil {
		t.Fatal("expected error with wrong passphrase")
	}

	signer, err := parseSSHSigner(pemBytes, passphrase)
	if err != nil {
		t.Fatalf("with passphrase: %v", err)
	}
	if signer == nil {
		t.Fatal("expected signer")
	}
}

func TestParseSSHSignerWithoutPassphrase(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	pemBytes := pem.EncodeToMemory(block)

	signer, err := parseSSHSigner(pemBytes, "")
	if err != nil {
		t.Fatalf("without passphrase: %v", err)
	}
	if signer == nil {
		t.Fatal("expected signer")
	}
}
