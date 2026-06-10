package main

import (
	"log/slog"

	qxlog "github.com/qxproject/qx/pkg/log"
)

func run() {
	qxlog.SetupFromEnv()
	slog.Info("qx-agent: not implemented yet", "phase", 2, "doc", "docs/mvp.md")
}
