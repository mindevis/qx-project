package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	qxlog "github.com/qxproject/qx/pkg/log"
)

const (
	logFileName        = "qxlauncher.log"
	defaultRotateBytes = 5 * 1024 * 1024
	defaultRotateKeep  = 3
)

var (
	rotateBytes int64 = defaultRotateBytes
	rotateKeep        = defaultRotateKeep
	activeSink  *rotatingFile
)

// Close releases the current log file. Safe to call when logging is not set up.
func Close() {
	if activeSink == nil {
		return
	}
	_ = activeSink.Close()
	activeSink = nil
}

// Dir is {dataDir}/logs.
func Dir(dataDir string) string {
	return filepath.Join(dataDir, "logs")
}

// FilePath is {dataDir}/logs/qxlauncher.log.
func FilePath(dataDir string) string {
	return filepath.Join(Dir(dataDir), logFileName)
}

// Setup configures slog to write to a rotating file and, when possible, stdout.
// Windows GUI builds have no console: stdout errors are ignored so the file
// still receives every line. Returns the log file path, or "" on failure.
func Setup(dataDir string, opts qxlog.Options) string {
	logPath := FilePath(dataDir)
	sink := &rotatingFile{path: logPath, max: rotateBytes, keep: rotateKeep}
	if err := sink.open(); err != nil {
		qxlog.Setup(opts)
		return ""
	}
	Close()
	activeSink = sink

	console := opts.Output
	if console == nil {
		console = os.Stdout
	}
	opts.Output = io.MultiWriter(sink, bestEffortWriter{console})
	if qxlog.ParseLevel(opts.Level) <= slog.LevelDebug {
		opts.AddSource = true
	}
	qxlog.Setup(opts)
	slog.Info("qxlauncher log file enabled", "path", logPath)
	return logPath
}

type bestEffortWriter struct{ w io.Writer }

func (b bestEffortWriter) Write(p []byte) (int, error) {
	if b.w != nil {
		_, _ = b.w.Write(p)
	}
	return len(p), nil
}

type rotatingFile struct {
	mu     sync.Mutex
	path   string
	f      *os.File
	size   int64
	max    int64
	keep   int
	closed bool
}

func (r *rotatingFile) open() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.openLocked()
}

func (r *rotatingFile) openLocked() error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	r.f = f
	r.size = info.Size()
	return nil
}

func (r *rotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return len(p), nil
	}
	if r.f == nil {
		if err := r.openLocked(); err != nil {
			return 0, err
		}
	}
	n, err := r.f.Write(p)
	r.size += int64(n)
	if err == nil && r.max > 0 && r.size >= r.max {
		_ = r.rotateLocked()
	}
	return n, err
}

func (r *rotatingFile) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	if r.f == nil {
		return nil
	}
	err := r.f.Close()
	r.f = nil
	return err
}

func (r *rotatingFile) rotateLocked() error {
	if r.f != nil {
		_ = r.f.Close()
		r.f = nil
	}
	if r.keep < 1 {
		r.keep = 1
	}
	oldest := r.path + "." + strconv.Itoa(r.keep)
	_ = os.Remove(oldest)
	for i := r.keep - 1; i >= 1; i-- {
		from := r.path + "." + strconv.Itoa(i)
		to := r.path + "." + strconv.Itoa(i+1)
		_ = os.Rename(from, to)
	}
	if err := os.Rename(r.path, r.path+".1"); err != nil {
		return r.openLocked()
	}
	return r.openLocked()
}
