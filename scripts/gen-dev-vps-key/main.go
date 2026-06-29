// Generates ed25519 SSH keys for local dev dedicated server (Flow C).
// Usage: go run . -dir infra/docker/vps-dev/keys
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

func main() {
	dir := flag.String("dir", "infra/docker/vps-dev/keys", "output directory")
	flag.Parse()

	privPath := filepath.Join(*dir, "dev_id_ed25519")
	pubPath := privPath + ".pub"
	authPath := filepath.Join(*dir, "authorized_keys")

	if _, err := os.Stat(privPath); err == nil {
		fmt.Println("keys already exist:", privPath)
		return
	}

	if err := os.MkdirAll(*dir, 0o700); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	privBlock, err := ssh.MarshalPrivateKey(priv, "qx-dev-vps")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	privPEM := pem.EncodeToMemory(privBlock)
	if err := os.WriteFile(privPath, privPEM, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	pubLine := string(ssh.MarshalAuthorizedKey(sshPub))
	if err := os.WriteFile(pubPath, []byte(pubLine), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(authPath, []byte(pubLine), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("generated:", privPath)
}
