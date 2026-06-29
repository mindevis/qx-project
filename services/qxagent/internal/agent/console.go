package agent

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/pkg/safepath"
)

func streamLines(stream string, r io.Reader, onOutput func(string, string)) {
	if onOutput == nil {
		return
	}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		onOutput(stream, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		onOutput("stderr", stream+" read error: "+err.Error())
	}
}

func (r *ProcessRunner) SetOutputHandler(fn func(stream, line string)) {
	r.mu.Lock()
	r.onOutput = fn
	r.mu.Unlock()
}

func (r *ProcessRunner) AttachConsole(payload protocol.ConsoleAttachPayload) {
	workDir := strings.TrimSpace(payload.WorkDir)
	if workDir == "" {
		return
	}
	resolved, err := safepath.ResolveRoot(workDir)
	if err != nil {
		r.emit("stderr", "invalid work dir: "+err.Error())
		return
	}
	workDir = resolved
	r.mu.Lock()
	if id := strings.TrimSpace(payload.GameServerID); id != "" {
		r.gameServerID = id
	}
	r.mu.Unlock()

	logPath, err := safepath.Join(workDir, "logs", "latest.log")
	if err != nil {
		r.emit("stderr", "invalid log path: "+err.Error())
		return
	}
	if lines, err := readRecentLogLines(logPath, 500); err == nil {
		for _, line := range lines {
			r.emit("log", line)
		}
	} else {
		r.emit("stderr", "log file not ready: "+logPath)
	}

	r.mu.Lock()
	r.startLogFollowLocked(workDir)
	r.mu.Unlock()
}

func (r *ProcessRunner) WriteInput(line string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.DryRun {
		if r.onOutput != nil {
			r.onOutput("stdin", "> "+line)
			r.onOutput("stdout", "[dry-run] command accepted")
		}
		return nil
	}
	if r.stdin == nil {
		err := fmt.Errorf("server process not running")
		if r.onOutput != nil {
			r.onOutput("stderr", err.Error())
		}
		return err
	}
	_, err := fmt.Fprintln(r.stdin, line)
	return err
}

func (r *ProcessRunner) closePipesLocked() {
	if r.stdin != nil {
		_ = r.stdin.Close()
		r.stdin = nil
	}
	for _, c := range r.pipeClosers {
		_ = c.Close()
	}
	r.pipeClosers = nil
}
