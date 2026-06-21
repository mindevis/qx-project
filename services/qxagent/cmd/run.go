package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	qxlog "github.com/qxproject/qx/pkg/log"
	"github.com/qxproject/qx/services/qxagent/internal/agent"
	"github.com/qxproject/qx/services/qxagent/internal/config"
)

var configPath = flag.String("config", "", "path to agent.toml (default: repo agent.toml or /etc/qx-agent/agent.toml)")

func run() {
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("config load failed", "error", err)
		return
	}
	qxlog.Setup(qxlog.Options{Level: cfg.LogLevel, Format: cfg.LogFormat})

	wsURL := cfg.WSURL
	if wsURL == "" {
		apiBase := cfg.APIBaseURL
		if apiBase == "" {
			apiBase = "http://localhost:3000/api/v1"
		}
		wsURL = agent.WSURLFromAPI(apiBase)
	}

	hostname := cfg.Hostname
	if hostname == "" {
		hostname = agent.DefaultHostname()
	}

	client := agent.NewClient(agent.Config{
		WSURL:    wsURL,
		Token:    cfg.AgentToken,
		Hostname: hostname,
		Version:  "0.1.0",
		DryRun:   cfg.DryRun,
	})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	slog.Info("qx-agent starting",
		"ws_url", wsURL,
		"hostname", hostname,
		"config", cfg.ConfigPath,
		"server_id", cfg.ServerID,
		"dry_run", cfg.DryRun,
	)
	if err := client.Run(ctx); err != nil && err != context.Canceled {
		slog.Error("qx-agent stopped", "err", err)
	}
}
