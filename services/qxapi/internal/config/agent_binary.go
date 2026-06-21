package config

import (
	"os"
	"path/filepath"
)

// agentBinaryCandidates are tried when QX_AGENT_BINARY_PATH is unset.
// API is usually started from services/qxapi (make api).
var agentBinaryCandidates = []string{
	"../../bin/qx-agent-linux",
	"bin/qx-agent-linux",
}

func resolveAgentBinaryPath() string {
	if v := os.Getenv("QX_AGENT_BINARY_PATH"); v != "" {
		if fileExists(v) {
			return v
		}
		// .env often uses repo-root relative path; API runs from services/qxapi.
		for _, candidate := range agentBinaryCandidates {
			if fileExists(candidate) {
				return candidate
			}
		}
		return v
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
