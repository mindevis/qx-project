package tray

import "strings"

// Tray icons are rasterized from web/qxweb/public/brand-icon.svg (launcher orbit + desktop icon).
// Windows uses ICO (required by LoadImage); other platforms use PNG.
var iconPendingPNG = trayIconData
var iconLinkedPNG = trayIconData

func LauncherPageURL(webBase string) string {
	base := strings.TrimRight(webBase, "/")
	if base == "" {
		base = "http://localhost:5173"
	}
	return base + "/launcher"
}
