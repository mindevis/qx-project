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
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/qxproject/qx/pkg/protocol"
)

type Config struct {
	WSURL    string
	Token    string
	Hostname string
	Version  string
	DryRun   bool
}

type Client struct {
	cfg    Config
	runner *ProcessRunner
	dialer *websocket.Dialer
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
		payload, _ := json.Marshal(protocol.ConsoleOutputPayload{Stream: stream, Line: line})
		_ = conn.WriteJSON(protocol.Envelope{
			V:       protocol.Version,
			Type:    protocol.TypeEvtConsoleOutput,
			TS:      time.Now().UTC().Format(time.RFC3339),
			Payload: payload,
		})
	})

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
			_ = conn.WriteJSON(protocol.Envelope{
				V:    protocol.Version,
				Type: protocol.TypeEvtAgentHeartbeat,
				TS:   time.Now().UTC().Format(time.RFC3339),
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
		if env.RequestID != "" {
			if cached, ok := cache.Get(env.RequestID); ok {
				if err := conn.WriteJSON(cached); err != nil {
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
		if err := conn.WriteJSON(res); err != nil {
			return err
		}
	}
}

func (c *Client) dispatchCommand(env protocol.Envelope) (*protocol.Envelope, error) {
	ts := time.Now().UTC().Format(time.RFC3339)
	switch env.Type {
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
	case protocol.TypeCmdConsoleInput:
		var payload protocol.ConsoleInputPayload
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return nil, err
		}
		if err := c.runner.WriteInput(payload.Line); err != nil {
			return nil, err
		}
		return nil, nil
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
	DryRun      bool
	mu          sync.Mutex
	cmd         *exec.Cmd
	dryPID      int
	stdin       io.WriteCloser
	pipeClosers []io.Closer
	onOutput    func(stream, line string)
}

func (r *ProcessRunner) Start(payload protocol.ServerStartPayload) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd != nil && r.cmd.Process != nil {
		return r.cmd.Process.Pid, nil
	}
	if r.DryRun {
		if r.dryPID == 0 {
			r.dryPID = os.Getpid()
		}
		slog.Info("dry-run start server", "jar", payload.JarPath, "pid", r.dryPID)
		if r.onOutput != nil {
			r.onOutput("stdout", "[QX Agent dry-run] Starting "+payload.JarPath)
			r.onOutput("stdout", "Done ("+fmt.Sprintf("%d", r.dryPID)+"ms)")
			r.onOutput("stdout", "For help, type \"help\"")
		}
		return r.dryPID, nil
	}
	if payload.JarPath == "" {
		return 0, errors.New("jar_path required")
	}
	args := append([]string{}, payload.JVMArgs...)
	args = append(args, "-jar", payload.JarPath)
	args = append(args, payload.ExtraArgs...)

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	cmd := exec.Command("java", args...)
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
	onOutput := r.onOutput
	go streamLines("stdout", stdoutR, onOutput)
	go streamLines("stderr", stderrR, onOutput)
	return cmd.Process.Pid, nil
}

func (r *ProcessRunner) Stop(graceful bool, timeout time.Duration) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.DryRun {
		slog.Info("dry-run stop server")
		r.dryPID = 0
		return 0, nil
	}
	if r.cmd == nil || r.cmd.Process == nil {
		r.closePipesLocked()
		return 0, nil
	}
	cmd := r.cmd
	r.cmd = nil
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

func DefaultHostname() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "qx-agent-" + uuid.NewString()[:8]
}
