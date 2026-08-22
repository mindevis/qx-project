package ollama

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/qxproject/qx/pkg/safepath"
)

const (
	DefaultRoot       = "/opt/qxsystem/ollama"
	DefaultListenAddr = "127.0.0.1:11434"
	downloadUserAgent = "QXProject/1.0 (https://github.com/qxproject/qx)"
	systemdUnitName   = "qx-ollama.service"
	systemdUnitPath   = "/etc/systemd/system/qx-ollama.service"
	githubDownload    = "https://github.com/ollama/ollama/releases/latest/download/"
)

var (
	ErrNotInstalled = errors.New("ollama is not installed")
	ErrNotRunning   = errors.New("ollama is not running")
	ErrInvalidName  = errors.New("invalid ollama model name")
	ErrUnsupported  = errors.New("unsupported architecture for ollama")
)

var (
	runtimeGOARCH  = runtime.GOARCH
	downloadFileFn = downloadFile
	extractTGZFn   = extractTGZ
	hasSystemdFn   = hasSystemd
	commandContext = exec.CommandContext
	httpGet        = http.DefaultClient.Do
	osStat         = os.Stat
	writeFileFn    = os.WriteFile
)

type Manager struct {
	RootDir    string
	DryRun     bool
	ListenAddr string
	mu         sync.Mutex
	proc       *exec.Cmd
}

type InstallResult struct {
	Version    string
	BinPath    string
	RootDir    string
	ModelsDir  string
	ListenAddr string
}

type Status struct {
	Installed  bool
	Running    bool
	Version    string
	BinPath    string
	RootDir    string
	ModelsDir  string
	ListenAddr string
}

type Model struct {
	Name       string
	Size       int64
	Digest     string
	ModifiedAt string
}

func RootFromServerRoot(serverRoot string) string {
	serverRoot = strings.TrimSpace(serverRoot)
	if serverRoot == "" {
		return DefaultRoot
	}
	return filepath.Join(filepath.Dir(serverRoot), "ollama")
}

func NewManager(rootDir string, dryRun bool) *Manager {
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		rootDir = DefaultRoot
	}
	return &Manager{
		RootDir:    rootDir,
		DryRun:     dryRun,
		ListenAddr: DefaultListenAddr,
	}
}

func (m *Manager) BinPath() string {
	return filepath.Join(m.RootDir, "bin", "ollama")
}

func (m *Manager) ModelsDir() string {
	return filepath.Join(m.RootDir, "models")
}

func LinuxArch() (string, error) {
	switch runtimeGOARCH {
	case "amd64":
		return "amd64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupported, runtimeGOARCH)
	}
}

func DownloadURL(arch string) string {
	return githubDownload + "ollama-linux-" + arch + ".tgz"
}

func (m *Manager) Install(ctx context.Context, onLog func(string)) (InstallResult, error) {
	arch, err := LinuxArch()
	if err != nil {
		return InstallResult{}, err
	}
	binPath := m.BinPath()
	modelsDir := m.ModelsDir()
	result := InstallResult{
		BinPath:    binPath,
		RootDir:    m.RootDir,
		ModelsDir:  modelsDir,
		ListenAddr: m.listenAddr(),
	}
	logLine(onLog, "[QX] Installing Ollama for linux/"+arch+" in "+m.RootDir)
	if m.DryRun {
		result.Version = "dry-run"
		logLine(onLog, "[QX] Ollama install dry-run")
		return result, nil
	}

	if err := os.MkdirAll(filepath.Join(m.RootDir, "bin"), 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("mkdir ollama: %w", err)
	}
	if err := os.MkdirAll(modelsDir, 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("mkdir models: %w", err)
	}

	archive := filepath.Join(m.RootDir, "ollama-linux-"+arch+".tgz")
	url := DownloadURL(arch)
	logLine(onLog, "[QX] Downloading Ollama…")
	if err := downloadFileFn(ctx, url, archive); err != nil {
		return InstallResult{}, err
	}
	logLine(onLog, "[QX] Extracting Ollama archive…")
	if err := extractTGZFn(archive, m.RootDir); err != nil {
		return InstallResult{}, err
	}
	_ = os.Remove(archive)
	if err := os.Chmod(binPath, 0o755); err != nil {
		if alt := filepath.Join(m.RootDir, "ollama"); fileExists(alt) {
			if err := os.MkdirAll(filepath.Dir(binPath), 0o755); err != nil {
				return InstallResult{}, err
			}
			if err := os.Rename(alt, binPath); err != nil {
				return InstallResult{}, err
			}
			_ = os.Chmod(binPath, 0o755)
		} else {
			return InstallResult{}, fmt.Errorf("ollama binary missing after extract: %w", err)
		}
	}
	if !fileExists(binPath) {
		return InstallResult{}, errors.New("ollama binary missing after extract")
	}
	result.Version = m.readCLIVersion(ctx)
	logLine(onLog, "[QX] Ollama ready: "+result.Version)
	return result, nil
}

func (m *Manager) Start(ctx context.Context) error {
	status := m.Status(ctx)
	if !status.Installed && !m.DryRun {
		return ErrNotInstalled
	}
	if status.Running {
		return nil
	}
	if m.DryRun {
		return nil
	}
	if hasSystemdFn() {
		if err := m.writeUnit(); err == nil {
			_ = m.run(ctx, "systemctl", "daemon-reload")
			if err := m.run(ctx, "systemctl", "enable", "--now", systemdUnitName); err == nil {
				return m.waitReady(ctx, 30*time.Second)
			}
		}
	}
	return m.startProcess(ctx)
}

func (m *Manager) Stop(ctx context.Context) error {
	if m.DryRun {
		m.killProcess()
		return nil
	}
	if hasSystemdFn() {
		_ = m.run(ctx, "systemctl", "stop", systemdUnitName)
	}
	m.killProcess()
	return nil
}

func (m *Manager) Status(ctx context.Context) Status {
	st := Status{
		BinPath:    m.BinPath(),
		RootDir:    m.RootDir,
		ModelsDir:  m.ModelsDir(),
		ListenAddr: m.listenAddr(),
	}
	st.Installed = m.DryRun || fileExists(st.BinPath)
	if !st.Installed {
		return st
	}
	st.Version = m.readCLIVersion(ctx)
	st.Running = m.DryRun || m.apiAlive(ctx)
	if st.Running && st.Version == "" {
		st.Version = m.readAPIVersion(ctx)
	}
	return st
}

func (m *Manager) ListModels(ctx context.Context) ([]Model, error) {
	if m.DryRun {
		return []Model{}, nil
	}
	if !m.apiAlive(ctx) {
		return nil, ErrNotRunning
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.apiURL("/api/tags"), nil)
	if err != nil {
		return nil, err
	}
	res, err := httpGet(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama tags: http %d", res.StatusCode)
	}
	var payload struct {
		Models []struct {
			Name       string `json:"name"`
			Size       int64  `json:"size"`
			Digest     string `json:"digest"`
			ModifiedAt string `json:"modified_at"`
		} `json:"models"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(payload.Models))
	for _, item := range payload.Models {
		out = append(out, Model{
			Name:       item.Name,
			Size:       item.Size,
			Digest:     item.Digest,
			ModifiedAt: item.ModifiedAt,
		})
	}
	return out, nil
}

func (m *Manager) PullModel(ctx context.Context, name string, onLog func(string)) error {
	name = strings.TrimSpace(name)
	if err := ValidateModelName(name); err != nil {
		return err
	}
	logLine(onLog, "[QX] Pulling Ollama model "+name+"…")
	if m.DryRun {
		return nil
	}
	if !m.apiAlive(ctx) {
		return ErrNotRunning
	}
	body := strings.NewReader(`{"name":` + jsonString(name) + `,"stream":false}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.apiURL("/api/pull"), body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := httpGet(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = fmt.Sprintf("http %d", res.StatusCode)
		}
		return fmt.Errorf("ollama pull: %s", msg)
	}
	logLine(onLog, "[QX] Model "+name+" installed")
	return nil
}

func (m *Manager) DeleteModel(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if err := ValidateModelName(name); err != nil {
		return err
	}
	if m.DryRun {
		return nil
	}
	if !m.apiAlive(ctx) {
		return ErrNotRunning
	}
	body := strings.NewReader(`{"name":` + jsonString(name) + `}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, m.apiURL("/api/delete"), body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := httpGet(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = fmt.Sprintf("http %d", res.StatusCode)
		}
		return fmt.Errorf("ollama delete: %s", msg)
	}
	return nil
}

func ValidateModelName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 200 {
		return ErrInvalidName
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-' || r == ':' || r == '/':
		default:
			return ErrInvalidName
		}
	}
	if strings.Contains(name, "..") {
		return ErrInvalidName
	}
	return nil
}

func (m *Manager) listenAddr() string {
	if strings.TrimSpace(m.ListenAddr) == "" {
		return DefaultListenAddr
	}
	return m.ListenAddr
}

func (m *Manager) apiURL(path string) string {
	return "http://" + m.listenAddr() + path
}

func (m *Manager) apiAlive(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.apiURL("/api/version"), nil)
	if err != nil {
		return false
	}
	res, err := httpGet(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return res.StatusCode == http.StatusOK
}

func (m *Manager) readAPIVersion(ctx context.Context) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.apiURL("/api/version"), nil)
	if err != nil {
		return ""
	}
	res, err := httpGet(req)
	if err != nil {
		return ""
	}
	defer res.Body.Close()
	var payload struct {
		Version string `json:"version"`
	}
	if json.NewDecoder(res.Body).Decode(&payload) != nil {
		return ""
	}
	return strings.TrimSpace(payload.Version)
}

func (m *Manager) readCLIVersion(ctx context.Context) string {
	bin := m.BinPath()
	if !fileExists(bin) {
		return ""
	}
	cmd := commandContext(ctx, bin, "--version")
	cmd.Env = m.env()
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(out))
	line = strings.TrimPrefix(line, "ollama version is ")
	return strings.TrimSpace(line)
}

func (m *Manager) env() []string {
	return append(os.Environ(),
		"OLLAMA_MODELS="+m.ModelsDir(),
		"OLLAMA_HOST="+m.listenAddr(),
	)
}

func (m *Manager) writeUnit() error {
	body := fmt.Sprintf(`[Unit]
Description=QX Ollama
After=network-online.target

[Service]
Type=exec
ExecStart=%s serve
WorkingDirectory=%s
Environment=OLLAMA_MODELS=%s
Environment=OLLAMA_HOST=%s
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`, m.BinPath(), m.RootDir, m.ModelsDir(), m.listenAddr())
	return writeFileFn(systemdUnitPath, []byte(body), 0o644)
}

func (m *Manager) startProcess(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.proc != nil && m.proc.Process != nil {
		return nil
	}
	cmd := commandContext(context.Background(), m.BinPath(), "serve")
	cmd.Dir = m.RootDir
	cmd.Env = m.env()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start ollama: %w", err)
	}
	m.proc = cmd
	go func() { _ = cmd.Wait() }()
	return m.waitReady(ctx, 30*time.Second)
}

func (m *Manager) killProcess() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.proc != nil && m.proc.Process != nil {
		_ = m.proc.Process.Kill()
	}
	m.proc = nil
}

func (m *Manager) waitReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if m.apiAlive(ctx) {
			return nil
		}
		time.Sleep(400 * time.Millisecond)
	}
	return errors.New("ollama did not become ready")
}

func (m *Manager) run(ctx context.Context, name string, args ...string) error {
	cmd := commandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%s: %s", err.Error(), msg)
	}
	return nil
}

func hasSystemd() bool {
	_, err := osStat("/run/systemd/system")
	return err == nil
}

func fileExists(path string) bool {
	_, err := osStat(path)
	return err == nil
}

func logLine(onLog func(string), line string) {
	if onLog != nil {
		onLog(line)
	}
}

func jsonString(v string) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", downloadUserAgent)
	res, err := httpGet(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: http %d", url, res.StatusCode)
	}
	return safepath.WriteStreamAtomic(dest, res.Body)
}

func extractTGZ(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.ToSlash(hdr.Name)
		if name == "" || strings.Contains(name, "..") {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)|0o644)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
}
