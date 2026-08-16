package deploy

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/qxproject/qx/pkg/safepath"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func newHostKeyCallback() (ssh.HostKeyCallback, error) {
	path := knownHostsPath()
	if err := ensureKnownHostsFile(path); err != nil {
		return nil, err
	}
	strict, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("known_hosts: %w", err)
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := strict(hostname, remote, key)
		if err == nil {
			return nil
		}
		var keyErr *knownhosts.KeyError
		if errors.As(err, &keyErr) && len(keyErr.Want) == 0 {
			return appendKnownHost(path, hostname, key)
		}
		return err
	}, nil
}

func knownHostsPath() string {
	if p := strings.TrimSpace(os.Getenv("QX_SSH_KNOWN_HOSTS")); p != "" {
		return p
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".qx", "ssh_known_hosts")
	}
	return filepath.Join(os.TempDir(), "qx_ssh_known_hosts")
}

func ensureKnownHostsFile(path string) error {
	abs, err := safepath.ResolveRoot(path)
	if err != nil {
		return err
	}
	if err := safepath.EnsureParent(abs); err != nil {
		return err
	}
	f, err := safepath.OpenFile(abs, os.O_RDONLY|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	return f.Close()
}

func appendKnownHost(path, hostname string, key ssh.PublicKey) error {
	abs, err := safepath.ResolveRoot(path)
	if err != nil {
		return err
	}
	f, err := safepath.OpenFile(abs, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	hash := knownhosts.HashHostname(hostname)
	line := fmt.Sprintf("%s %s %s\n", hash, key.Type(), base64.StdEncoding.EncodeToString(key.Marshal()))
	_, err = f.WriteString(line)
	return err
}
