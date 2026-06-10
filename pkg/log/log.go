package log

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// Options configures the default slog logger.
type Options struct {
	Level  string
	Format string
	Output io.Writer
}

// ParseLevel maps LOG_LEVEL strings to slog levels.
func ParseLevel(raw string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO", "":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Setup creates a slog logger, sets it as the default, and returns it.
func Setup(opts Options) *slog.Logger {
	out := opts.Output
	if out == nil {
		out = os.Stdout
	}

	level := ParseLevel(opts.Level)
	handlerOpts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if strings.EqualFold(strings.TrimSpace(opts.Format), "json") {
		handler = slog.NewJSONHandler(out, handlerOpts)
	} else {
		handler = slog.NewTextHandler(out, handlerOpts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// SetupFromEnv configures logging from LOG_LEVEL and LOG_FORMAT.
func SetupFromEnv() *slog.Logger {
	return Setup(Options{
		Level:  os.Getenv("LOG_LEVEL"),
		Format: os.Getenv("LOG_FORMAT"),
	})
}
