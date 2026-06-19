package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	qxlog "github.com/qxproject/qx/pkg/log"
	"github.com/qxproject/qx/services/qxagent/internal/agent"
)

func run() {
	qxlog.SetupFromEnv()

	token := os.Getenv("QX_AGENT_TOKEN")
	if token == "" {
		slog.Error("QX_AGENT_TOKEN is required")
		return
	}

	wsURL := os.Getenv("QX_AGENT_WS_URL")
	if wsURL == "" {
		apiBase := os.Getenv("QX_API_BASE_URL")
		if apiBase == "" {
			apiBase = "http://localhost:3000/api/v1"
		}
		wsURL = agent.WSURLFromAPI(apiBase)
	}

	hostname := os.Getenv("QX_AGENT_HOSTNAME")
	if hostname == "" {
		hostname = agent.DefaultHostname()
	}

	client := agent.NewClient(agent.Config{
		WSURL:    wsURL,
		Token:    token,
		Hostname: hostname,
		Version:  "0.1.0",
		DryRun:   os.Getenv("QX_AGENT_DRY_RUN") == "1",
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	slog.Info("qx-agent starting", "ws_url", wsURL, "hostname", hostname, "dry_run", os.Getenv("QX_AGENT_DRY_RUN") == "1")
	if err := client.Run(ctx); err != nil && err != context.Canceled {
		slog.Error("qx-agent stopped", "err", err)
	}
}
