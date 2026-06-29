module github.com/qxproject/qx/services/qxlauncher

go 1.25.0

require (
	fyne.io/systray v1.12.2
	github.com/google/uuid v1.6.0
	github.com/pelletier/go-toml/v2 v2.4.2
	github.com/qxproject/qx/pkg/log v0.0.0
	github.com/qxproject/qx/pkg/mcmanifest v0.0.0
	github.com/qxproject/qx/pkg/mojangjava v0.0.0
	github.com/qxproject/qx/pkg/reporoot v0.0.0
	github.com/qxproject/qx/pkg/safepath v0.0.0
	golang.org/x/sys v0.46.0
)

require github.com/godbus/dbus/v5 v5.1.0 // indirect

replace github.com/qxproject/qx/pkg/reporoot => ../../pkg/reporoot

replace github.com/qxproject/qx/pkg/log => ../../pkg/log

replace github.com/qxproject/qx/pkg/mcmanifest => ../../pkg/mcmanifest

replace github.com/qxproject/qx/pkg/mojangjava => ../../pkg/mojangjava

replace github.com/qxproject/qx/pkg/safepath => ../../pkg/safepath
