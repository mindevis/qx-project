package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const minBytes = 32

var jwtSecretLine = regexp.MustCompile(`(?m)^jwt_secret\s*=\s*".*"$`)

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

func generateSSHMasterKey(n int) (string, error) {
	if n < minBytes {
		n = minBytes
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

func patchTomlFile(path, secret string) error {
	line := fmt.Sprintf(`jwt_secret = %q`, secret)
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return os.WriteFile(path, []byte(line+"\n"), 0o600)
	}
	content := string(data)
	if jwtSecretLine.MatchString(content) {
		content = jwtSecretLine.ReplaceAllString(content, line)
	} else {
		if len(content) > 0 && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += line + "\n"
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

func main() {
	tomlFile := flag.String("toml", "", "write jwt_secret into this TOML file (e.g. qxapi.toml)")
	sshMaster := flag.Bool("ssh-master", false, "print SSH_MASTER_KEY (standard base64, 32 bytes) for prod")
	bytes := flag.Int("bytes", minBytes, "random bytes before encoding (min 32)")
	flag.Parse()

	var (
		secret string
		err    error
	)
	if *sshMaster {
		secret, err = generateSSHMasterKey(*bytes)
	} else {
		secret, err = generateSecret(*bytes)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "generate:", err)
		os.Exit(1)
	}

	if *tomlFile != "" {
		if *sshMaster {
			fmt.Fprintln(os.Stderr, "-toml applies only to jwt_secret; omit -ssh-master")
			os.Exit(1)
		}
		if err := patchTomlFile(*tomlFile, secret); err != nil {
			fmt.Fprintln(os.Stderr, "toml:", err)
			os.Exit(1)
		}
		fmt.Printf("jwt_secret updated in %s\n", *tomlFile)
		return
	}

	fmt.Println(secret)
}
