package agent

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/pkg/safepath"
)

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

	payload.JVMArgs = sanitizeArgs(payload.JVMArgs)
	payload.Args = sanitizeArgs(payload.Args)
	payload.ExtraArgs = sanitizeArgs(payload.ExtraArgs)
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
	if root == "" {
		if !filepath.IsAbs(bin) {
			return "", fmt.Errorf("relative java bin requires work dir")
		}
		return safepath.ResolveRoot(bin)
	}
	return safepath.ResolveUnder(root, bin)
}

func sanitizeArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	out := make([]string, len(args))
	copy(out, args)
	return out
}
