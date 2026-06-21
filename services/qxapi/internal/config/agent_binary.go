package config

import (
	"os"
	"path/filepath"
)

// agentBinaryCandidates are tried when agent_binary_path is unset in qxapi.toml.
// API is usually started from services/qxapi (make api).
var agentBinaryCandidates = []string{
	"../../bin/qx-agent-linux",
	"bin/qx-agent-linux",
}

func resolveAgentBinaryPath(configured string) string {
	if configured != "" {
		if fileExists(configured) {
			return configured
		}
		for _, candidate := range agentBinaryCandidates {
			if fileExists(candidate) {
				return candidate
			}
		}
		return configured
	}
	for _, candidate := range agentBinaryCandidates {
		if fileExists(candidate) {
			return candidate
		}
	}
	return agentBinaryCandidates[0]
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// AgentBinaryAbs returns an absolute path when possible (for logging).
func AgentBinaryAbs(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
