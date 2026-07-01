package notify

import (
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/qxproject/qx/services/qxlauncher/internal/proc"
)

const dedupeWindow = 60 * time.Second

var (
	notifyMu   sync.Mutex
	notifyLast = make(map[string]time.Time)
)

func Show(title, message string) {
	key := title + "\x00" + message
	notifyMu.Lock()
	if last, ok := notifyLast[key]; ok && time.Since(last) < dedupeWindow {
		notifyMu.Unlock()
		slog.Debug("notification suppressed (duplicate)", "title", title)
		return
	}
	notifyLast[key] = time.Now()
	notifyMu.Unlock()

	slog.Info("notification", "title", title, "message", message)
	switch runtime.GOOS {
	case "windows":
		go platformShow(title, message)
	case "darwin":
		_ = proc.Command("osascript", "-e", `display notification "`+escapeAppleScript(message)+`" with title "`+escapeAppleScript(title)+`"`).Start()
	}
}

func escapeAppleScript(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '"' || s[i] == '\\' {
			out = append(out, '\\')
		}
		out = append(out, s[i])
	}
	return string(out)
}
