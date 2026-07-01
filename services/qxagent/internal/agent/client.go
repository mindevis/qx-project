package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxagent/internal/fs"
	"github.com/qxproject/qx/services/qxagent/internal/installer"
)

type Config struct {
	WSURL      string
	Token      string
	Hostname   string
	Version    string
	ServerRoot string
	DryRun     bool
}

type Client struct {
	cfg     Config
	runner  *ProcessRunner
	dialer  *websocket.Dialer
	writeMu sync.Mutex
}

func NewClient(cfg Config) *Client {
	if cfg.Version == "" {
		cfg.Version = "0.1.0"
	}
	return &Client{
		cfg:    cfg,
		runner: &ProcessRunner{DryRun: cfg.DryRun},
		dialer: websocket.DefaultDialer,
	}
}

func WSURLFromAPI(apiBase string) string {
	raw := apiBase
	if u, err := url.Parse(apiBase); err == nil && u.Scheme != "" {
		switch u.Scheme {
		case "https":
			u.Scheme = "wss"
		case "http":
			u.Scheme = "ws"
		}
		u.Path = "/agent/v1/connect"
		u.RawQuery = ""
		return u.String()
	}
	if len(raw) > 0 && raw[len(raw)-1] == '/' {
		raw = raw[:len(raw)-1]
	}
	if len(raw) >= 7 && raw[len(raw)-7:] == "/api/v1" {
		raw = raw[:len(raw)-7]
	}
	return raw + "/agent/v1/connect"
}

func (c *Client) Run(ctx context.Context) error {
	backoff := time.Second
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := c.connectOnce(ctx)
		if err == nil || errors.Is(err, context.Canceled) {
			return err
		}
		slog.Warn("agent connection lost", "err", err, "retry_in", backoff)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func (c *Client) connectOnce(ctx context.Context) error {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+c.cfg.Token)
	if c.cfg.Hostname != "" {
		header.Set("X-Agent-Hostname", c.cfg.Hostname)
	}
	header.Set("X-Agent-Version", c.cfg.Version)

	conn, _, err := c.dialer.DialContext(ctx, c.cfg.WSURL, header)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	slog.Info("agent connected", "url", c.cfg.WSURL)

	c.runner.SetOutputHandler(func(stream, line string) {
		c.emitConsoleStream(conn, c.runner.ConsoleGameServerID(), stream, line)
	})
	c.runner.SetStatusHandler(func(payload protocol.ServerStatusPayload) {
		c.emitServerStatus(conn, payload)
	})
	c.reportServerStatus(conn)

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.readLoop(conn)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case <-heartbeat.C:
			payload, _ := json.Marshal(protocol.HeartbeatPayload{})
			_ = c.writeEnvelope(conn, protocol.Envelope{
				V:       protocol.Version,
				Type:    protocol.TypeEvtAgentHeartbeat,
				TS:      time.Now().UTC().Format(time.RFC3339),
				Payload: payload,
			})
		}
	}
}

func (c *Client) readLoop(conn *websocket.Conn) error {
	cache := newRequestCache(1000)
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var env protocol.Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}
		if env.Type == protocol.TypeCmdServerInstall {
			go c.runInstallAsync(conn, cache, env)
			continue
		}
		if env.RequestID != "" {
			if cached, ok := cache.Get(env.RequestID); ok {
				if err := c.writeEnvelope(conn, cached); err != nil {
					return err
				}
				continue
			}
		}
		res, err := c.dispatchCommand(env)
		if err != nil {
			slog.Error("command failed", "type", env.Type, "err", err)
			continue
		}
		if res == nil {
			continue
		}
		if env.RequestID != "" {
			cache.Set(env.RequestID, *res)
		}
		if err := c.writeEnvelope(conn, *res); err != nil {
			return err
		}
	}
}

func (c *Client) writeEnvelope(conn *websocket.Conn, env protocol.Envelope) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return conn.WriteJSON(env)
}

func (c *Client) emitConsole(conn *websocket.Conn, line string) {
	c.emitConsoleStream(conn, "", "stdout", line)
}

func (c *Client) emitConsoleStream(conn *websocket.Conn, gameServerID, stream, line string) {
	payload, _ := json.Marshal(protocol.ConsoleOutputPayload{
		Stream:       stream,
		Line:         line,
		GameServerID: gameServerID,
	})
	_ = c.writeEnvelope(conn, protocol.Envelope{
		V:       protocol.Version,
		Type:    protocol.TypeEvtConsoleOutput,
		TS:      time.Now().UTC().Format(time.RFC3339),
		Payload: payload,
	})
}

func (c *Client) emitServerStatus(conn *websocket.Conn, status protocol.ServerStatusPayload) {
	payload, _ := json.Marshal(status)
	_ = c.writeEnvelope(conn, protocol.Envelope{
		V:       protocol.Version,
		Type:    protocol.TypeEvtServerStatus,
		TS:      time.Now().UTC().Format(time.RFC3339),
		Payload: payload,
	})
}

func (c *Client) reportServerStatus(conn *websocket.Conn) {
	c.emitServerStatus(conn, c.runner.CurrentStatus())
}

func (c *Client) runInstallAsync(conn *websocket.Conn, cache *requestCache, env protocol.Envelope) {
	res, err := c.buildInstallResponse(conn, env)
	if err != nil {
		slog.Error("install command failed", "err", err)
		return
	}
	if res == nil {
		return
	}
	if env.RequestID != "" {
		cache.Set(env.RequestID, *res)
	}
	if err := c.writeEnvelope(conn, *res); err != nil {
		slog.Error("install response write failed", "err", err)
	}
}

func (c *Client) buildInstallResponse(conn *websocket.Conn, env protocol.Envelope) (*protocol.Envelope, error) {
	ts := time.Now().UTC().Format(time.RFC3339)
	var payload protocol.ServerInstallPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return nil, err
	}
	installCtx, cancel := context.WithTimeout(context.Background(), forgeInstallTimeout())
	defer cancel()
	spec, err := installer.Install(installCtx, installer.Options{
		DryRun: c.cfg.DryRun,
		OnLog: func(line string) {
			c.emitConsoleStream(conn, payload.GameServerID, "stdout", line)
		},
		JavaRoot: installer.JavaRootFromServerRoot(c.cfg.ServerRoot),
	}, installer.InstallConfig{
		ServerType:    payload.ServerType,
		WorkDir:       payload.WorkDir,
		MCVersion:     payload.MCVersion,
		LoaderVersion: payload.LoaderVersion,
		Name:          payload.Name,
		Address:       payload.Address,
		Port:          payload.Port,
		RconPassword:  payload.RconPassword,
	})
	var resPayload []byte
	if err != nil {
		resPayload, _ = json.Marshal(map[string]string{"error": err.Error()})
	} else {
		resPayload, _ = json.Marshal(protocol.ServerInstallResult{
			WorkDir:   spec.WorkDir,
			JarPath:   spec.JarPath,
			Command:   spec.Command,
			Args:      spec.Args,
			JVMArgs:   spec.JVMArgs,
			ExtraArgs: spec.ExtraArgs,
			JavaBin:   spec.JavaBin,
		})
	}
	return &protocol.Envelope{
		V:         protocol.Version,
		Type:      protocol.TypeResServerInstall,
		RequestID: env.RequestID,
		TS:        ts,
		Payload:   resPayload,
	}, nil
}

func forgeInstallTimeout() time.Duration {
	return 25 * time.Minute
}

func (c *Client) dispatchCommand(env protocol.Envelope) (*protocol.Envelope, error) {
	ts := time.Now().UTC().Format(time.RFC3339)
	switch env.Type {
	case protocol.TypeCmdServerInstall:
		return nil, nil
	case protocol.TypeCmdServerStart:
		var payload protocol.ServerStartPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return nil, err
		}
		pid, err := c.runner.Start(payload)
		var resPayload []byte
		if err != nil {
			resPayload, _ = json.Marshal(map[string]string{"error": err.Error()})
		} else {
			resPayload, _ = json.Marshal(protocol.ServerStartResult{PID: pid})
		}
		return &protocol.Envelope{
			V:         protocol.Version,
			Type:      protocol.TypeResServerStart,
			RequestID: env.RequestID,
			TS:        ts,
			Payload:   resPayload,
		}, nil
	case protocol.TypeCmdServerStop:
		var payload protocol.ServerStopPayload
		if len(env.Payload) > 0 {
			_ = json.Unmarshal(env.Payload, &payload)
		}
		timeout := time.Duration(payload.TimeoutSec) * time.Second
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		exitCode, err := c.runner.Stop(payload.Graceful, timeout)
		resPayload, _ := json.Marshal(protocol.ServerStopResult{ExitCode: exitCode})
		if err != nil {
			resPayload, _ = json.Marshal(map[string]string{"error": err.Error()})
		}
		return &protocol.Envelope{
			V:         protocol.Version,
			Type:      protocol.TypeResServerStop,
			RequestID: env.RequestID,
			TS:        ts,
			Payload:   resPayload,
		}, nil
	case protocol.TypeCmdServerConfigure:
		var payload protocol.ServerConfigurePayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return nil, err
		}
		err := installer.ConfigureServerProperties(payload.WorkDir, installer.ServerPropertiesConfig{
			Name:         payload.Name,
			Address:      payload.Address,
			Port:         payload.Port,
			RconPassword: payload.RconPassword,
		})
		var resPayload []byte
		if err != nil {
			resPayload, _ = json.Marshal(map[string]string{"error": err.Error()})
		} else {
			resPayload, _ = json.Marshal(map[string]string{"status": "ok"})
		}
		return &protocol.Envelope{
			V:         protocol.Version,
			Type:      protocol.TypeResServerConfigure,
			RequestID: env.RequestID,
			TS:        ts,
			Payload:   resPayload,
		}, nil
	case protocol.TypeCmdConsoleInput:
		var payload protocol.ConsoleInputPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return nil, err
		}
		if err := c.runner.WriteInput(payload.Line); err != nil {
			return nil, err
		}
		return nil, nil
	case protocol.TypeCmdConsoleAttach:
		var payload protocol.ConsoleAttachPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return nil, err
		}
		c.runner.AttachConsole(payload)
		return nil, nil
	case protocol.TypeCmdServerPropertiesGet:
		var payload protocol.GameServerWorkDirPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return nil, err
		}
		props, err := fs.ReadServerProperties(payload.WorkDir)
		var resPayload []byte
		if err != nil {
			resPayload, _ = json.Marshal(map[string]string{"error": err.Error()})
		} else {
			resPayload, _ = json.Marshal(protocol.ServerPropertiesResult{Properties: props})
		}
		return &protocol.Envelope{
			V:         protocol.Version,
			Type:      protocol.TypeResServerPropertiesGet,
			RequestID: env.RequestID,
			TS:        ts,
			Payload:   resPayload,
		}, nil
	case protocol.TypeCmdServerPropertiesPatch:
		var payload protocol.ServerPropertiesPatchPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return nil, err
		}
		err := fs.PatchServerProperties(payload.WorkDir, payload.Updates)
		var resPayload []byte
		if err != nil {
			resPayload, _ = json.Marshal(map[string]string{"error": err.Error()})
		} else {
			resPayload, _ = json.Marshal(map[string]string{"status": "ok"})
		}
		return &protocol.Envelope{
			V:         protocol.Version,
			Type:      protocol.TypeResServerPropertiesPatch,
			RequestID: env.RequestID,
			TS:        ts,
			Payload:   resPayload,
		}, nil
	case protocol.TypeCmdServerFilesList:
		var payload protocol.ServerFilesPathPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return nil, err
		}
		entries, err := fs.ListDir(payload.WorkDir, payload.Path)
		var resPayload []byte
		if err != nil {
			resPayload, _ = json.Marshal(map[string]string{"error": err.Error()})
		} else {
			resPayload, _ = json.Marshal(protocol.ServerFilesListResult{Entries: entries})
		}
		return &protocol.Envelope{
			V:         protocol.Version,
			Type:      protocol.TypeResServerFilesList,
			RequestID: env.RequestID,
			TS:        ts,
			Payload:   resPayload,
		}, nil
	case protocol.TypeCmdServerFilesRead:
		var payload protocol.ServerFilesPathPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return nil, err
		}
		content, size, err := fs.ReadFile(payload.WorkDir, payload.Path)
		var resPayload []byte
		if err != nil {
			resPayload, _ = json.Marshal(map[string]string{"error": err.Error()})
		} else {
			resPayload, _ = json.Marshal(protocol.ServerFilesReadResult{
				Path:    payload.Path,
				Content: content,
				Size:    size,
			})
		}
		return &protocol.Envelope{
			V:         protocol.Version,
			Type:      protocol.TypeResServerFilesRead,
			RequestID: env.RequestID,
			TS:        ts,
			Payload:   resPayload,
		}, nil
	case protocol.TypeCmdServerFilesWrite:
		var payload protocol.ServerFilesWritePayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return nil, err
		}
		err := fs.WriteFile(payload.WorkDir, payload.Path, payload.Content)
		var resPayload []byte
		if err != nil {
			resPayload, _ = json.Marshal(map[string]string{"error": err.Error()})
		} else {
			resPayload, _ = json.Marshal(map[string]string{"status": "ok"})
		}
		return &protocol.Envelope{
			V:         protocol.Version,
			Type:      protocol.TypeResServerFilesWrite,
			RequestID: env.RequestID,
			TS:        ts,
			Payload:   resPayload,
		}, nil
	case protocol.TypeCmdServerModsList:
		var payload protocol.ServerModsListPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return nil, err
		}
		entries, err := fs.ListMods(payload.WorkDir, payload.ServerType)
		var resPayload []byte
		if err != nil {
			resPayload, _ = json.Marshal(map[string]string{"error": err.Error()})
		} else {
			resPayload, _ = json.Marshal(protocol.ServerModsListResult{Entries: entries})
		}
		return &protocol.Envelope{
			V:         protocol.Version,
			Type:      protocol.TypeResServerModsList,
			RequestID: env.RequestID,
			TS:        ts,
			Payload:   resPayload,
		}, nil
	case protocol.TypeCmdAgentPing:
		return &protocol.Envelope{
			V:         protocol.Version,
			Type:      protocol.TypeEvtAgentHeartbeat,
			RequestID: env.RequestID,
			TS:        ts,
		}, nil
	default:
		return nil, nil
	}
}

type ProcessRunner struct {
	DryRun             bool
	mu                 sync.Mutex
	cmd                *exec.Cmd
	dryPID             int
	stdin              io.WriteCloser
	pipeClosers        []io.Closer
	onOutput           func(stream, line string)
	onStatus           func(protocol.ServerStatusPayload)
	gameServerID       string
	logFollowCancel    context.CancelFunc
	stoppingGracefully bool
	managedWorkDir     string
}

func (r *ProcessRunner) ConsoleGameServerID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.gameServerID
}

func (r *ProcessRunner) emit(stream, line string) {
	r.mu.Lock()
	fn := r.onOutput
	r.mu.Unlock()
	if fn != nil {
		fn(stream, line)
	}
}

func (r *ProcessRunner) emitLocked(stream, line string) {
	if r.onOutput != nil {
		r.onOutput(stream, line)
	}
}

func (r *ProcessRunner) stopLogFollowLocked() {
	if r.logFollowCancel != nil {
		r.logFollowCancel()
		r.logFollowCancel = nil
	}
}

func (r *ProcessRunner) startLogFollowLocked(workDir string) {
	r.stopLogFollowLocked()
	if workDir == "" {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.logFollowCancel = cancel
	go followServerLog(ctx, workDir, func(line string) {
		r.emit("log", line)
	})
}

func (r *ProcessRunner) Start(payload protocol.ServerStartPayload) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cleanupDeadProcessLocked()
	if r.cmd != nil && r.cmd.Process != nil {
		if payload.WorkDir != "" {
			r.startLogFollowLocked(payload.WorkDir)
		}
		if id := strings.TrimSpace(payload.GameServerID); id != "" {
			r.gameServerID = id
		}
		return r.cmd.Process.Pid, nil
	}
	r.gameServerID = strings.TrimSpace(payload.GameServerID)
	start, err := ValidateStartPayload(payload)
	if err != nil {
		return 0, err
	}
	if r.DryRun {
		if r.dryPID == 0 {
			r.dryPID = os.Getpid()
		}
		target := start.JarPath
		if target == "" {
			target = start.Command
		}
		slog.Info("dry-run start server", "target", target, "work_dir", start.WorkDir, "pid", r.dryPID)
		r.emitLocked("stdout", "[QX Agent dry-run] Starting "+target)
		r.emitLocked("stdout", "Done ("+fmt.Sprintf("%d", r.dryPID)+"ms)")
		r.emitLocked("stdout", "For help, type \"help\"")
		return r.dryPID, nil
	}
	if start.Command != "" {
		return r.startCommandLocked(start)
	}
	if start.JarPath == "" {
		return 0, errors.New("jar_path required")
	}
	bin, err := ResolvedExecBin(start)
	if err != nil {
		return 0, err
	}
	jar, err := ResolvedJarPath(start)
	if err != nil {
		return 0, err
	}
	args := append([]string{}, start.JVMArgs...)
	args = append(args, "-jar", jar)
	args = append(args, start.ExtraArgs...)

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	cmd := exec.Command(bin, args...)
	cmd.Stdin = stdinR
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW
	if start.WorkDir != "" {
		cmd.Dir = start.WorkDir
	}
	applyJavaEnv(cmd, start.JavaBin)
	if err := cmd.Start(); err != nil {
		_ = stdinR.Close()
		_ = stdinW.Close()
		_ = stdoutR.Close()
		_ = stdoutW.Close()
		_ = stderrR.Close()
		_ = stderrW.Close()
		return 0, err
	}
	r.cmd = cmd
	r.stdin = stdinW
	r.pipeClosers = []io.Closer{stdinR, stdoutW, stderrW}
	r.stoppingGracefully = false
	r.managedWorkDir = start.WorkDir
	go streamLines("stdout", stdoutR, r.emit)
	go streamLines("stderr", stderrR, r.emit)
	r.startLogFollowLocked(start.WorkDir)
	go r.watchManagedProcess(cmd, start.WorkDir)
	return cmd.Process.Pid, nil
}

func (r *ProcessRunner) startCommandLocked(start ValidatedStart) (int, error) {
	cmdName, err := ResolvedExecCommand(start)
	if err != nil {
		return 0, err
	}
	args := append([]string{}, start.Args...)
	args = append(args, start.ExtraArgs...)
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	cmd := exec.Command(cmdName, args...)
	if start.WorkDir != "" {
		cmd.Dir = start.WorkDir
	}
	applyJavaEnv(cmd, start.JavaBin)
	cmd.Stdin = stdinR
	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW
	if err := cmd.Start(); err != nil {
		_ = stdinR.Close()
		_ = stdinW.Close()
		_ = stdoutR.Close()
		_ = stdoutW.Close()
		_ = stderrR.Close()
		_ = stderrW.Close()
		return 0, err
	}
	r.cmd = cmd
	r.stdin = stdinW
	r.pipeClosers = []io.Closer{stdinR, stdoutW, stderrW}
	r.stoppingGracefully = false
	r.managedWorkDir = start.WorkDir
	go streamLines("stdout", stdoutR, r.emit)
	go streamLines("stderr", stderrR, r.emit)
	r.startLogFollowLocked(start.WorkDir)
	go r.watchManagedProcess(cmd, start.WorkDir)
	return cmd.Process.Pid, nil
}

func (r *ProcessRunner) Stop(graceful bool, timeout time.Duration) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.DryRun {
		slog.Info("dry-run stop server")
		r.dryPID = 0
		r.gameServerID = ""
		r.stopLogFollowLocked()
		return 0, nil
	}
	if r.cmd == nil || r.cmd.Process == nil {
		r.closePipesLocked()
		r.gameServerID = ""
		r.stopLogFollowLocked()
		return 0, nil
	}
	cmd := r.cmd
	r.stoppingGracefully = true
	r.cmd = nil
	r.gameServerID = ""
	r.stopLogFollowLocked()
	r.closePipesLocked()

	if graceful {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case err := <-done:
			if err == nil {
				return cmd.ProcessState.ExitCode(), nil
			}
		case <-time.After(timeout):
			_ = cmd.Process.Kill()
		}
	} else {
		_ = cmd.Process.Kill()
	}
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return -1, err
	}
	return cmd.ProcessState.ExitCode(), nil
}

func javaBin(start ValidatedStart) string {
	if bin := strings.TrimSpace(start.JavaBin); bin != "" {
		return bin
	}
	return "java"
}

func applyJavaEnv(cmd *exec.Cmd, javaBinPath string) {
	javaBinPath = strings.TrimSpace(javaBinPath)
	if javaBinPath == "" || javaBinPath == "java" {
		return
	}
	binDir := filepath.Dir(javaBinPath)
	javaHome := filepath.Dir(binDir)
	cmd.Env = append(os.Environ(),
		"JAVA_HOME="+javaHome,
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
}

func DefaultHostname() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "qx-agent-" + uuid.NewString()[:8]
}
