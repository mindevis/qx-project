//go:build windows

package notify

import (
	"log/slog"

	"git.sr.ht/~jackmordaunt/go-toast"
)

// toastAppID matches ProductName in versioninfo.json so Action Center groups
// notifications under a stable application identity.
const toastAppID = "QX Launcher"

func platformShow(title, message string) {
	err := (&toast.Notification{
		AppID: toastAppID,
		Title: title,
		Body:  message,
	}).Push()
	if err != nil {
		slog.Debug("toast notification failed", "err", err)
	}
}
