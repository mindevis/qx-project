package deploy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	qxlog "github.com/qxproject/qx/pkg/log"
	"github.com/qxproject/qx/services/qxapi/internal/crypto"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

const (
	agentInstallPath = "/opt/qxsystem/agent/qx-agent"
	agentConfigPath  = "/etc/qxsystem/agent/agent.toml"
	serverRoot       = "/opt/qxsystem/server"
	agentUnitPath    = "/etc/systemd/system/qx-agent.service"
	uploadPath       = "/tmp/qx-agent-upload"
)

var (
	ErrBinaryNotConfigured = errors.New("agent binary path not configured")
	ErrInvalidSSHKey       = errors.New("invalid ssh private key")
	ErrNonLinuxHost        = errors.New("ssh host is not linux")
)

type SSHConfig struct {
	Encryptor  *crypto.Encryptor
	APIBaseURL string
	BinaryPath string
	Dial       func(ctx context.Context, addr string, config *ssh.ClientConfig) (any, error)
	VerifyOS   func(client any) error
	RunRemote  func(client any, apiURL, serverID, agentToken string, binary []byte, hostsFix string) error
}

type SSHDeployer struct {
	cfg SSHConfig
}

func NewSSH(cfg SSHConfig) *SSHDeployer {
	if cfg.Dial == nil {
		cfg.Dial = defaultDial
	}
	if cfg.VerifyOS == nil {
		cfg.VerifyOS = defaultVerifyOS
	}
	if cfg.RunRemote == nil {
		cfg.RunRemote = runRemoteProvision
	}
	return &SSHDeployer{cfg: cfg}
}

func (d *SSHDeployer) Deploy(ctx context.Context, serverID string, cred models.SSHCredential, agentToken string) error {
	if strings.TrimSpace(d.cfg.BinaryPath) == "" {
		return ErrBinaryNotConfigured
	}
	binary, err := os.ReadFile(d.cfg.BinaryPath)
	if err != nil {
		return fmt.Errorf("read agent binary: %w", err)
	}

	pem, err := d.cfg.Encryptor.Decrypt(cred.PrivateKeyEnc)
	if err != nil {
		return fmt.Errorf("decrypt ssh key: %w", err)
	}
	passphrase := ""
	if len(cred.PrivateKeyPassphraseEnc) > 0 {
		raw, err := d.cfg.Encryptor.Decrypt(cred.PrivateKeyPassphraseEnc)
		if err != nil {
			return fmt.Errorf("decrypt ssh key passphrase: %w", err)
		}
		passphrase = string(raw)
	}
	signer, err := parseSSHSigner(pem, passphrase)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", cred.Host, cred.Port)
	slog.Info("ssh deploy",
		"direction", qxlog.DirectionOut,
		"transport", qxlog.TransportSSH,
		"server_id", serverID,
		"host", addr,
	)
	hostKeyCallback, err := newHostKeyCallback()
	if err != nil {
		return fmt.Errorf("ssh host key: %w", err)
	}
	clientConfig := &ssh.ClientConfig{
		User:            cred.Username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: hostKeyCallback,
		Timeout:         30 * time.Second,
	}

	client, err := d.cfg.Dial(ctx, addr, clientConfig)
	if err != nil {
		return fmt.Errorf("ssh connect %s: %w", addr, err)
	}
	defer func() {
		if closer, ok := client.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}()

	if err := d.cfg.VerifyOS(client); err != nil {
		return err
	}

	remoteAPI, hostsFix, err := d.resolveRemoteAgentAPI(client, cred)
	if err != nil {
		return err
	}

	deployCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- d.cfg.RunRemote(client, remoteAPI, serverID, agentToken, binary, hostsFix)
	}()

	select {
	case <-deployCtx.Done():
		return fmt.Errorf("ssh deploy timeout: %w", deployCtx.Err())
	case err := <-done:
		if err != nil {
			return fmt.Errorf("ssh provision: %w", err)
		}
		return nil
	}
}

func defaultDial(_ context.Context, addr string, config *ssh.ClientConfig) (any, error) {
	return ssh.Dial("tcp", addr, config)
}

func defaultVerifyOS(clientAny any) error {
	client, ok := clientAny.(*ssh.Client)
	if !ok {
		return errors.New("invalid ssh client")
	}
	out, err := runSSHCommand(client, "uname -s")
	if err != nil {
		return fmt.Errorf("detect host os: %w", err)
	}
	return verifyHostKernel(out)
}

func verifyHostKernel(unameOutput string) error {
	kernel := strings.TrimSpace(unameOutput)
	if !strings.EqualFold(kernel, "Linux") {
		return fmt.Errorf("%w: %s", ErrNonLinuxHost, kernel)
	}
	return nil
}

func (d *SSHDeployer) resolveRemoteAgentAPI(clientAny any, cred models.SSHCredential) (string, string, error) {
	apiURL := d.cfg.APIBaseURL
	remoteAPI := agentAPIURL(apiURL, cred)

	client, ok := clientAny.(*ssh.Client)
	if !ok {
		return remoteAPI, "", nil
	}

	u, err := url.Parse(strings.TrimSpace(apiURL))
	if err != nil || u.Hostname() == "" {
		return remoteAPI, "", nil
	}
	apiHost := u.Hostname()

	out, err := runSSHCommand(client, fmt.Sprintf("getent ahostsv4 %q 2>/dev/null | awk 'NR==1{print $1}'", apiHost))
	if err != nil {
		out, _ = runSSHCommand(client, fmt.Sprintf("getent hosts %q 2>/dev/null | awk 'NR==1{print $1}'", apiHost))
	}
	hostnameOut, _ := runSSHCommand(client, "hostname -f 2>/dev/null || hostname")

	hostsFix := apiHostsOverrideScript(apiURL, cred, remoteAPIResolution{
		ResolvedIP:     strings.TrimSpace(out),
		RemoteHostname: strings.TrimSpace(hostnameOut),
	})
	return remoteAPI, hostsFix, nil
}

func runRemoteProvision(clientAny any, apiURL, serverID, agentToken string, binary []byte, hostsFix string) error {
	client, ok := clientAny.(*ssh.Client)
	if !ok {
		return errors.New("invalid ssh client")
	}
	if err := uploadBinary(client, uploadPath, binary); err != nil {
		return err
	}

	envBody := buildAgentConfig(apiURL, serverID, agentToken)
	unitBody := buildSystemdUnit()
	script := fmt.Sprintf(`set -e
SUDO=""
if [ "$(id -u)" -ne 0 ]; then SUDO="sudo"; fi
$SUDO mkdir -p /opt/qxsystem/agent /opt/qxsystem/server /etc/qxsystem/agent
$SUDO install -m 755 %s %s
$SUDO tee %s > /dev/null <<'QXCFG'
%sQXCFG
$SUDO chmod 600 %s
$SUDO tee %s > /dev/null <<'QXUNIT'
%sQXUNIT
%s$SUDO systemctl daemon-reload
$SUDO systemctl enable qx-agent
$SUDO systemctl restart qx-agent
`, uploadPath, agentInstallPath, agentConfigPath, envBody, agentConfigPath, agentUnitPath, unitBody, hostsFix)

	out, err := runSSHCommand(client, script)
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(out))
	}
	return nil
}

func uploadBinary(client *ssh.Client, dst string, content []byte) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	pipe, err := session.StdinPipe()
	if err != nil {
		return err
	}
	go func() {
		_, _ = pipe.Write(content)
		_ = pipe.Close()
	}()
	cmd := fmt.Sprintf("cat > %s && chmod 755 %s", dst, dst)
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("upload agent binary: %w", err)
	}
	return nil
}

func runSSHCommand(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	out, err := session.CombinedOutput(cmd)
	return string(out), err
}

func buildAgentConfig(apiURL, serverID, agentToken string) string {
	apiBase := strings.TrimRight(strings.TrimSpace(apiURL), "/")
	if apiBase == "" {
		apiBase = "http://localhost:3000"
	}
	if !strings.HasSuffix(apiBase, "/api/v1") {
		apiBase += "/api/v1"
	}
	return fmt.Sprintf(`api_base_url = %q
agent_token = %q
server_id = %q
server_root = %q
`, apiBase, agentToken, serverID, serverRoot)
}

func buildSystemdUnit() string {
	return fmt.Sprintf(`[Unit]
Description=QX Agent
After=network-online.target

[Service]
ExecStart=%s
WorkingDirectory=%s
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`, agentInstallPath, serverRoot)
}
