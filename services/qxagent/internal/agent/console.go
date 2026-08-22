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

func (r *ProcessRunner) SetOutputHandler(fn func(stream, line, gameServerID string)) {
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
		r.emit("stderr", "invalid work dir: "+err.Error(), strings.TrimSpace(payload.GameServerID))
		return
	}
	workDir = resolved
	gsID := strings.TrimSpace(payload.GameServerID)
	r.mu.Lock()
	inst := r.getOrCreateInstanceLocked(workDir)
	if gsID != "" {
		inst.gameServerID = gsID
	} else {
		gsID = inst.gameServerID
	}
	r.mu.Unlock()

	logPath, err := safepath.Join(workDir, "logs", "latest.log")
	if err != nil {
		r.emit("stderr", "invalid log path: "+err.Error(), gsID)
		return
	}
	if lines, err := readRecentLogLines(logPath, 500); err == nil {
		for _, line := range lines {
			r.emit("log", line, gsID)
		}
	} else {
		r.emit("stderr", "log file not ready: "+logPath, gsID)
	}

	r.mu.Lock()
	r.startLogFollowLocked(inst)
	r.mu.Unlock()
}

func (r *ProcessRunner) WriteInput(line, gameServerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	gameServerID = strings.TrimSpace(gameServerID)
	if r.DryRun {
		r.emitLocked("stdin", "> "+line, gameServerID)
		r.emitLocked("stdout", "[dry-run] command accepted", gameServerID)
		return nil
	}
	inst := r.inputInstanceLocked(gameServerID)
	if inst == nil || inst.stdin == nil {
		err := fmt.Errorf("server process not running")
		r.emitLocked("stderr", err.Error(), gameServerID)
		return err
	}
	_, err := fmt.Fprintln(inst.stdin, line)
	return err
}

func (r *ProcessRunner) inputInstanceLocked(gameServerID string) *managedInstance {
	if gameServerID != "" {
		for _, inst := range r.instances {
			if inst != nil && inst.gameServerID == gameServerID {
				return inst
			}
		}
		return nil
	}
	var found *managedInstance
	n := 0
	for _, inst := range r.instances {
		if inst != nil && inst.stdin != nil {
			found = inst
			n++
		}
	}
	if n == 1 {
		return found
	}
	return nil
}

func (inst *managedInstance) closePipesLocked() {
	if inst == nil {
		return
	}
	if inst.stdin != nil {
		_ = inst.stdin.Close()
		inst.stdin = nil
	}
	for _, c := range inst.pipeClosers {
		_ = c.Close()
	}
	inst.pipeClosers = nil
}
