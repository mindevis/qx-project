package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"strings"
)

const minBytes = 32

func generateSecret(n int) (string, error) {
	if n < minBytes {
		n = minBytes
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func patchEnvFile(path, secret string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return os.WriteFile(path, []byte("JWT_SECRET="+secret+"\n"), 0o600)
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, "JWT_SECRET=") {
			lines[i] = "JWT_SECRET=" + secret
			found = true
			break
		}
	}
	if !found {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "JWT_SECRET="+secret)
	}

	out := strings.Join(lines, "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return os.WriteFile(path, []byte(out), 0o600)
}

func main() {
	envFile := flag.String("env", "", "write JWT_SECRET into this .env file (e.g. .env)")
	bytes := flag.Int("bytes", minBytes, "random bytes before encoding (min 32)")
	flag.Parse()

	secret, err := generateSecret(*bytes)
	if err != nil {
		fmt.Fprintln(os.Stderr, "generate:", err)
		os.Exit(1)
	}

	if *envFile == "" {
		fmt.Println(secret)
		return
	}

	if err := patchEnvFile(*envFile, secret); err != nil {
		fmt.Fprintln(os.Stderr, "env:", err)
		os.Exit(1)
	}
	fmt.Printf("JWT_SECRET updated in %s\n", *envFile)
}
