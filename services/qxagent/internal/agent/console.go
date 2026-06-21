package agent

import (
	"bufio"
	"fmt"
	"io"
)

func streamLines(stream string, r io.Reader, onOutput func(string, string)) {
	if onOutput == nil {
		return
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		onOutput(stream, scanner.Text())
	}
}

func (r *ProcessRunner) SetOutputHandler(fn func(stream, line string)) {
	r.mu.Lock()
	r.onOutput = fn
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
