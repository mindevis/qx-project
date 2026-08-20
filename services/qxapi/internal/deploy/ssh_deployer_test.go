package deploy

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/qxproject/qx/services/qxapi/internal/crypto"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

const devKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestSSHDeployerMissingBinary(t *testing.T) {
	enc, _ := crypto.NewEncryptor(devKey)
	d := NewSSH(SSHConfig{Encryptor: enc})
	err := d.Deploy(context.Background(), "srv-1", models.SSHCredential{}, "token")
	if !errors.Is(err, ErrBinaryNotConfigured) {
		t.Fatalf("expected ErrBinaryNotConfigured, got %v", err)
	}
}

func TestSSHDeployerInvalidKey(t *testing.T) {
	enc, _ := crypto.NewEncryptor(devKey)
	keyPEM, _ := enc.Encrypt([]byte("not-a-key"))
	dir := t.TempDir()
	bin := filepath.Join(dir, "qx-agent")
	if err := os.WriteFile(bin, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	d := NewSSH(SSHConfig{
		Encryptor:  enc,
		BinaryPath: bin,
		Dial: func(context.Context, string, *ssh.ClientConfig) (any, error) {
			t.Fatal("dial should not run with invalid key")
			return nil, nil
		},
	})
	err := d.Deploy(context.Background(), "srv-1", models.SSHCredential{
		PrivateKeyEnc: keyPEM,
	}, "token")
	if err == nil || !errors.Is(err, ErrInvalidSSHKey) {
		t.Fatalf("expected invalid key error, got %v", err)
	}
}

type testSSHClient struct{}

func (testSSHClient) Close() error { return nil }

func noopVerifyOS(any) error { return nil }

func TestDefaultVerifyOSInvalidClient(t *testing.T) {
	err := defaultVerifyOS(testSSHClient{})
	if err == nil || err.Error() != "invalid ssh client" {
		t.Fatalf("expected invalid client error, got %v", err)
	}
}

func TestVerifyHostKernel(t *testing.T) {
	if err := verifyHostKernel("Linux\n"); err != nil {
		t.Fatalf("linux: %v", err)
	}
	if err := verifyHostKernel("Darwin"); !errors.Is(err, ErrNonLinuxHost) {
		t.Fatalf("darwin: %v", err)
	}
}

func TestSSHDeployerProvisionSuccess(t *testing.T) {
	enc, _ := crypto.NewEncryptor(devKey)
	dir := t.TempDir()
	bin := filepath.Join(dir, "qx-agent")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var capturedConfig string
	d := NewSSH(SSHConfig{
		Encryptor:  enc,
		APIBaseURL: "https://api.example.com",
		BinaryPath: bin,
		Dial: func(context.Context, string, *ssh.ClientConfig) (any, error) {
			return testSSHClient{}, nil
		},
		VerifyOS: noopVerifyOS,
		RunRemote: func(_ any, apiURL, serverID, agentToken string, binary []byte, _ string) error {
			if serverID != "srv-42" || agentToken != "jwt-token" {
				t.Fatalf("unexpected ids: %s %s", serverID, agentToken)
			}
			if apiURL != "https://api.example.com" {
				t.Fatalf("unexpected api url: %s", apiURL)
			}
			if len(binary) == 0 {
				t.Fatal("expected binary payload")
			}
			capturedConfig = buildAgentConfig(apiURL, serverID, agentToken)
			return nil
		},
	})
	err := d.Deploy(context.Background(), "srv-42", models.SSHCredential{
		Host: "10.0.0.2", Port: 2222, Username: "ubuntu",
		PrivateKeyEnc: mustEncryptValidKey(t, enc),
	}, "jwt-token")
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if !strings.Contains(capturedConfig, `agent_token = "jwt-token"`) {
		t.Fatalf("config missing token: %q", capturedConfig)
	}
}

func TestSSHDeployerNonLinuxHost(t *testing.T) {
	enc, _ := crypto.NewEncryptor(devKey)
	dir := t.TempDir()
	bin := filepath.Join(dir, "qx-agent")
	_ = os.WriteFile(bin, []byte("bin"), 0o755)

	d := NewSSH(SSHConfig{
		Encryptor:  enc,
		BinaryPath: bin,
		Dial: func(context.Context, string, *ssh.ClientConfig) (any, error) {
			return testSSHClient{}, nil
		},
		VerifyOS: func(any) error {
			return fmt.Errorf("%w: Darwin", ErrNonLinuxHost)
		},
	})
	err := d.Deploy(context.Background(), "srv-1", models.SSHCredential{
		Host:          "10.0.0.1",
		Port:          22,
		Username:      "root",
		PrivateKeyEnc: mustEncryptValidKey(t, enc),
	}, "token")
	if !errors.Is(err, ErrNonLinuxHost) {
		t.Fatalf("expected ErrNonLinuxHost, got %v", err)
	}
}

func TestDefaultVerifyOSHookRunsBeforeProvision(t *testing.T) {
	enc, _ := crypto.NewEncryptor(devKey)
	dir := t.TempDir()
	bin := filepath.Join(dir, "qx-agent")
	_ = os.WriteFile(bin, []byte("bin"), 0o755)
	called := false
	d := NewSSH(SSHConfig{
		Encryptor:  enc,
		BinaryPath: bin,
		Dial: func(context.Context, string, *ssh.ClientConfig) (any, error) {
			return testSSHClient{}, nil
		},
		VerifyOS: func(any) error {
			called = true
			return nil
		},
		RunRemote: func(any, string, string, string, []byte, string) error { return nil },
	})
	err := d.Deploy(context.Background(), "srv-1", models.SSHCredential{
		PrivateKeyEnc: mustEncryptValidKey(t, enc),
	}, "token")
	if err != nil || !called {
		t.Fatalf("verify os: err=%v called=%v", err, called)
	}
}

func TestSSHDeployerDialFailure(t *testing.T) {
	enc, _ := crypto.NewEncryptor(devKey)
	dir := t.TempDir()
	bin := filepath.Join(dir, "qx-agent")
	_ = os.WriteFile(bin, []byte("bin"), 0o755)

	d := NewSSH(SSHConfig{
		Encryptor:  enc,
		BinaryPath: bin,
		Dial: func(context.Context, string, *ssh.ClientConfig) (any, error) {
			return nil, errors.New("connection refused")
		},
	})
	err := d.Deploy(context.Background(), "srv-1", models.SSHCredential{
		Host: "10.0.0.1", Port: 22, Username: "root",
		PrivateKeyEnc: mustEncryptValidKey(t, enc),
	}, "token")
	if err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("expected dial error, got %v", err)
	}
}

func TestBuildAgentConfigAppendsAPIPath(t *testing.T) {
	body := buildAgentConfig("http://localhost:3000", "id-1", "tok")
	if !strings.Contains(body, `api_base_url = "http://localhost:3000/api/v1"`) {
		t.Fatalf("unexpected config: %q", body)
	}
}

func TestBuildAgentConfigDefaultAPIBase(t *testing.T) {
	body := buildAgentConfig("", "id-1", "tok")
	if !strings.Contains(body, `api_base_url = "http://localhost:3000/api/v1"`) {
		t.Fatalf("unexpected config: %q", body)
	}
}

func TestBuildSystemdUnit(t *testing.T) {
	unit := buildSystemdUnit()
	if !strings.Contains(unit, "ExecStart=/opt/qxsystem/agent/qx-agent") {
		t.Fatal("missing ExecStart")
	}
	if strings.Contains(unit, "EnvironmentFile") {
		t.Fatal("systemd unit should read agent.toml directly")
	}
	if !strings.Contains(unit, "KillMode=process") {
		t.Fatal("systemd unit should not kill Minecraft on agent restart")
	}
}

func TestSSHDeployerTimeout(t *testing.T) {
	enc, _ := crypto.NewEncryptor(devKey)
	dir := t.TempDir()
	bin := filepath.Join(dir, "qx-agent")
	_ = os.WriteFile(bin, []byte("bin"), 0o755)

	d := NewSSH(SSHConfig{
		Encryptor:  enc,
		BinaryPath: bin,
		Dial: func(context.Context, string, *ssh.ClientConfig) (any, error) {
			return testSSHClient{}, nil
		},
		VerifyOS: noopVerifyOS,
		RunRemote: func(any, string, string, string, []byte, string) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		},
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	err := d.Deploy(ctx, "srv-1", models.SSHCredential{
		PrivateKeyEnc: mustEncryptValidKey(t, enc),
	}, "token")
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout, got %v", err)
	}
}

func TestSSHDeployerRunRemoteFailure(t *testing.T) {
	enc, _ := crypto.NewEncryptor(devKey)
	dir := t.TempDir()
	bin := filepath.Join(dir, "qx-agent")
	_ = os.WriteFile(bin, []byte("bin"), 0o755)

	d := NewSSH(SSHConfig{
		Encryptor:  enc,
		BinaryPath: bin,
		Dial: func(context.Context, string, *ssh.ClientConfig) (any, error) {
			return testSSHClient{}, nil
		},
		VerifyOS: noopVerifyOS,
		RunRemote: func(any, string, string, string, []byte, string) error {
			return errors.New("systemctl failed")
		},
	})
	err := d.Deploy(context.Background(), "srv-1", models.SSHCredential{
		Host:          "10.0.0.1",
		Port:          22,
		Username:      "root",
		PrivateKeyEnc: mustEncryptValidKey(t, enc),
	}, "token")
	if err == nil || !strings.Contains(err.Error(), "systemctl failed") {
		t.Fatalf("expected provision error, got %v", err)
	}
}

func mustEncryptValidKey(t *testing.T, enc *crypto.Encryptor) []byte {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "test")
	if err != nil {
		t.Fatal(err)
	}
	pemBytes := pem.EncodeToMemory(block)
	out, err := enc.Encrypt(pemBytes)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
