package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"

	qxlog "github.com/qxproject/qx/pkg/log"
)

// Setup configures slog with stdout and an append-only file at {dataDir}/logs/qxlauncher.log.
// Returns the log file path, or "" when file logging could not be enabled.
func Setup(dataDir string, opts qxlog.Options) string {
	logPath := filepath.Join(dataDir, "logs", "qxlauncher.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		qxlog.Setup(opts)
		return ""
	}
	stdout := io.Writer(os.Stdout)
	if opts.Output != nil {
		stdout = opts.Output
	}
	opts.Output = io.MultiWriter(stdout, &appendFile{path: logPath})
	qxlog.Setup(opts)
	slog.Info("qxlauncher log file enabled", "path", logPath)
	return logPath
}

type appendFile struct {
	path string
}

func (a *appendFile) Write(p []byte) (int, error) {
	f, err := os.OpenFile(a.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return f.Write(p)
}
