package agent

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/pkg/safepath"
)

// ValidatedExecutable is an absolute executable path or the literal "java".
type ValidatedExecutable string

func (e ValidatedExecutable) String() string {
	return string(e)
}

// ExecCommandValidated starts a process with a validated executable path.
func ExecCommandValidated(executable ValidatedExecutable, args ...string) *exec.Cmd {
	return exec.Command(executable.String(), args...)
}

// ValidatedStart holds server start fields after path and command validation.
type ValidatedStart struct {
	WorkDir   string
	Command   string
	JarPath   string
	JavaBin   string
	JVMArgs   []string
	Args      []string
	ExtraArgs []string
}

// ValidateStartPayload validates remote start input before exec or file access.
func ValidateStartPayload(payload protocol.ServerStartPayload) (ValidatedStart, error) {
	if err := sanitizeStartPayload(&payload); err != nil {
		return ValidatedStart{}, err
	}
	return ValidatedStart{
		WorkDir:   payload.WorkDir,
		Command:   payload.Command,
		JarPath:   payload.JarPath,
		JavaBin:   payload.JavaBin,
		JVMArgs:   payload.JVMArgs,
		Args:      payload.Args,
		ExtraArgs: payload.ExtraArgs,
	}, nil
}

// ResolvedExecBin returns the validated java binary path for exec.Command.
func ResolvedExecBin(start ValidatedStart) (ValidatedExecutable, error) {
	path, err := resolveJavaBin(start.JavaBin, start.WorkDir)
	if err != nil {
		return "", err
	}
	return ValidatedExecutable(path), nil
}

// ResolvedJarPath returns the validated jar path for exec.Command args.
func ResolvedJarPath(start ValidatedStart) (string, error) {
	if strings.TrimSpace(start.JarPath) == "" {
		return "", fmt.Errorf("jar_path required")
	}
	return resolvePathRef(start.JarPath, start.WorkDir)
}

// ResolvedExecCommand returns the validated command for exec.Command.
func ResolvedExecCommand(start ValidatedStart) (ValidatedExecutable, error) {
	path, err := resolveCommand(start.Command, start.WorkDir)
	if err != nil {
		return "", err
	}
	return ValidatedExecutable(path), nil
}

func sanitizeStartPayload(payload *protocol.ServerStartPayload) error {
	var root string
	if wd := strings.TrimSpace(payload.WorkDir); wd != "" {
		resolved, err := safepath.ResolveRoot(wd)
		if err != nil {
			return fmt.Errorf("work dir: %w", err)
		}
		root = resolved
		payload.WorkDir = root
	}

	if cmd := strings.TrimSpace(payload.Command); cmd != "" {
		resolved, err := resolveCommand(cmd, root)
		if err != nil {
			return fmt.Errorf("command: %w", err)
		}
		payload.Command = resolved
	}

	if jar := strings.TrimSpace(payload.JarPath); jar != "" {
		resolved, err := resolvePathRef(jar, root)
		if err != nil {
			return fmt.Errorf("jar path: %w", err)
		}
		payload.JarPath = resolved
	}

	if jb := strings.TrimSpace(payload.JavaBin); jb != "" {
		resolved, err := resolveJavaBin(jb, root)
		if err != nil {
			return fmt.Errorf("java bin: %w", err)
		}
		payload.JavaBin = resolved
	}

	var err error
	if payload.JVMArgs, err = sanitizeArgs(payload.JVMArgs); err != nil {
		return fmt.Errorf("jvm args: %w", err)
	}
	if payload.Args, err = sanitizeArgs(payload.Args); err != nil {
		return fmt.Errorf("args: %w", err)
	}
	if payload.ExtraArgs, err = sanitizeArgs(payload.ExtraArgs); err != nil {
		return fmt.Errorf("extra args: %w", err)
	}
	return nil
}

func resolveCommand(command, root string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("empty command")
	}
	if command == "java" {
		return "java", nil
	}
	if command == "run.sh" {
		if root == "" {
			return "", fmt.Errorf("work dir required for run.sh")
		}
		return safepath.Join(root, "run.sh")
	}
	if filepath.IsAbs(command) {
		abs, err := filepath.Abs(command)
		if err != nil {
			return "", err
		}
		base := filepath.Base(abs)
		if base != "java" && base != "run.sh" {
			return "", fmt.Errorf("command not allowed: %q", command)
		}
		if base == "run.sh" {
			if root == "" {
				return "", fmt.Errorf("work dir required for run.sh")
			}
			return safepath.ResolveUnder(root, abs)
		}
		return abs, nil
	}
	return "", fmt.Errorf("command not allowed: %q", command)
}

func resolvePathRef(path, root string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	if root != "" {
		return safepath.ResolveUnder(root, path)
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("relative path requires work dir")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if strings.Contains(abs, "..") {
		return "", fmt.Errorf("invalid path")
	}
	return abs, nil
}

func resolveJavaBin(bin, root string) (string, error) {
	bin = strings.TrimSpace(bin)
	if bin == "" || bin == "java" {
		return "java", nil
	}
	if filepath.IsAbs(bin) {
		abs, err := filepath.Abs(bin)
		if err != nil {
			return "", err
		}
		if filepath.Base(abs) != "java" {
			return "", fmt.Errorf("java bin not allowed: %q", bin)
		}
		// Mojang Java lives outside per-instance work dirs (e.g. /opt/qxsystem/java).
		return abs, nil
	}
	if root == "" {
		return "", fmt.Errorf("relative java bin requires work dir")
	}
	return safepath.ResolveUnder(root, bin)
}

func sanitizeArgs(args []string) ([]string, error) {
	if len(args) == 0 {
		return args, nil
	}
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if !isSafeExecArg(arg) {
			return nil, fmt.Errorf("invalid exec arg")
		}
		out = append(out, arg)
	}
	return out, nil
}

func isSafeExecArg(arg string) bool {
	if arg == "" || len(arg) > 4096 {
		return false
	}
	if strings.ContainsAny(arg, "\x00\n\r;&|$`<>") {
		return false
	}
	return true
}
