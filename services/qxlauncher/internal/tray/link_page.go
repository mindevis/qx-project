package tray

import (
	"log/slog"

	"github.com/qxproject/qx/services/qxlauncher/internal/browser"
)

func OpenLinkPage(linkURL string) {
	if linkURL == "" {
		return
	}
	if err := browser.Open(linkURL); err != nil {
		slog.Warn("open link page failed", "url", linkURL, "err", err)
		return
	}
	slog.Info("opened link page in browser", "url", linkURL)
}
