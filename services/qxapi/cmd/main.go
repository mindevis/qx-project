package main

import (
	"log/slog"
	"os"
)

var exit = os.Exit

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		exit(1)
	}
}
