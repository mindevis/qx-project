package main

import (
	"log/slog"
	"runtime/debug"
)

func main() {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("qxlauncher panic", "panic", rec, "stack", string(debug.Stack()))
		}
	}()
	run()
}
