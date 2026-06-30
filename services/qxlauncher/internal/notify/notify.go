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
		script := fmtToastScript(title, message)
		_ = proc.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Start()
	case "darwin":
		_ = proc.Command("osascript", "-e", `display notification "`+escapeAppleScript(message)+`" with title "`+escapeAppleScript(title)+`"`).Start()
	}
}

func fmtToastScript(title, message string) string {
	return `[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null; ` +
		`[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null; ` +
		`$template = '<toast><visual><binding template="ToastText02"><text id="1">' + ` +
		`'` + escapePS(title) + `' + '</text><text id="2">' + '` + escapePS(message) + `' + '</text></binding></visual></toast>'; ` +
		`$xml = New-Object Windows.Data.Xml.Dom.XmlDocument; $xml.LoadXml($template); ` +
		`$toast = [Windows.UI.Notifications.ToastNotification]::new($xml); ` +
		`[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('QXLauncher').Show($toast)`
}

func escapePS(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			out = append(out, "''"...)
			continue
		}
		out = append(out, s[i])
	}
	return string(out)
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
