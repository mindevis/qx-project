module github.com/qxproject/qx/services/qxlauncher

go 1.22

require (
	fyne.io/systray v1.12.2
	github.com/google/uuid v1.6.0
	github.com/qxproject/qx/pkg/log v0.0.0
	github.com/qxproject/qx/pkg/mcmanifest v0.0.0
)

require (
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
)

replace github.com/qxproject/qx/pkg/log => ../../pkg/log

replace github.com/qxproject/qx/pkg/mcmanifest => ../../pkg/mcmanifest
