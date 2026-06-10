package main

import (
	"log/slog"

	qxlog "github.com/qxproject/qx/pkg/log"
)

func run() {
	qxlog.SetupFromEnv()
	slog.Info("qx-launcher: not implemented yet", "phase", 1, "doc", "docs/mvp.md")
}
