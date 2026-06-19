package tray

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/qxproject/qx/services/qxlauncher/internal/apiclient"
	"github.com/qxproject/qx/services/qxlauncher/internal/cache"
	"github.com/qxproject/qx/services/qxlauncher/internal/minecraft"
	"github.com/qxproject/qx/services/qxlauncher/internal/notify"
)

type Config struct {
	APIBase      string
	DeviceToken  string
	DataDir      string
	LaunchDryRun bool
	LaunchPoll   time.Duration
	InstancePoll time.Duration
}

func RunLoop(ctx context.Context, cfg Config) {
	if cfg.LaunchPoll <= 0 {
		cfg.LaunchPoll = 2 * time.Second
	}
	if cfg.InstancePoll <= 0 {
		cfg.InstancePoll = 30 * time.Second
	}
	if cfg.DataDir == "" {
		home, _ := os.UserHomeDir()
		cfg.DataDir = filepath.Join(home, ".qx")
	}

	api := apiclient.New(cfg.APIBase, cfg.DeviceToken)
	downloader := minecraft.NewDownloader(cfg.DataDir)

	launchTicker := time.NewTicker(cfg.LaunchPoll)
	defer launchTicker.Stop()
	syncTicker := time.NewTicker(cfg.InstancePoll)
	defer syncTicker.Stop()

	slog.Info("tray loop started", "launch_poll", cfg.LaunchPoll, "sync_poll", cfg.InstancePoll)

	for {
		select {
		case <-ctx.Done():
			return
		case <-syncTicker.C:
			if err := syncInstances(ctx, api, cfg.DataDir); err != nil {
				slog.Warn("instance sync failed", "err", err)
			}
		case <-launchTicker.C:
			item, err := api.FetchPendingLaunch(ctx)
			if err != nil {
				slog.Warn("launch poll failed", "err", err)
				continue
			}
			if item == nil || item.Manifest == nil {
				continue
			}
			go executeLaunch(ctx, api, downloader, cfg, item)
		}
	}
}

func executeLaunch(ctx context.Context, api *apiclient.Client, dl *minecraft.Downloader, cfg Config, item *apiclient.LaunchRequestItem) {
	username := "Player"
	offlineUUID := "00000000-0000-0000-0000-000000000000"
	mcVersion := ""
	if item.Manifest != nil {
		mcVersion = item.Manifest.MCVersion
	}
	if item.Profile != nil {
		username = item.Profile.Username
		offlineUUID = item.Profile.OfflineUUID
	}

	jar, err := dl.EnsureClientJar(ctx, item.Manifest)
	if err != nil {
		slog.Error("download failed", "err", err)
		_ = api.UpdateLaunch(ctx, item.ID, map[string]any{
			"status":     "failed",
			"error_code": "DOWNLOAD_FAILED",
		})
		return
	}

	libPaths, err := dl.EnsureLibraries(ctx, item.Manifest)
	if err != nil {
		slog.Error("libraries failed", "err", err)
		_ = api.UpdateLaunch(ctx, item.ID, map[string]any{
			"status":     "failed",
			"error_code": "LIBRARIES_FAILED",
		})
		return
	}

	nativesDir, err := dl.EnsureNatives(ctx, item.Manifest)
	if err != nil {
		slog.Warn("natives optional failed", "err", err)
		nativesDir = ""
	}

	assetsDir, err := dl.EnsureAssets(ctx, item.Manifest)
	if err != nil {
		slog.Error("assets failed", "err", err)
		_ = api.UpdateLaunch(ctx, item.ID, map[string]any{
			"status":     "failed",
			"error_code": "ASSETS_FAILED",
		})
		return
	}

	javaBin, err := dl.EnsureJava(ctx, item.Manifest)
	if err != nil {
		slog.Error("java runtime failed", "err", err)
		_ = api.UpdateLaunch(ctx, item.ID, map[string]any{
			"status":     "failed",
			"error_code": "JAVA_FAILED",
		})
		return
	}

	gameDir := dl.InstanceGameDir(item.Manifest.InstanceID)
	plan := minecraft.BuildLaunchPlan(item.Manifest, jar, libPaths, nativesDir, assetsDir, gameDir, username, offlineUUID, javaBin)
	_ = api.UpdateLaunch(ctx, item.ID, map[string]any{"status": "running", "pid": 0})

	label := username
	if mcVersion != "" {
		label = username + " · " + mcVersion
	}
	notify.Show("QXLauncher", "Запуск Minecraft: "+label)

	if cfg.LaunchDryRun {
		slog.Info("launch dry-run", "jar", jar, "main", plan.MainClass, "user", username)
		_ = api.UpdateLaunch(ctx, item.ID, map[string]any{"status": "completed", "exit_code": 0})
		return
	}

	cmd := exec.CommandContext(ctx, plan.JavaBin, plan.Args...)
	cmd.Dir = plan.WorkingDir
	if err := cmd.Start(); err != nil {
		slog.Error("java start failed", "err", err)
		_ = api.UpdateLaunch(ctx, item.ID, map[string]any{
			"status":     "failed",
			"error_code": "JAVA_START_FAILED",
		})
		return
	}
	pid := cmd.Process.Pid
	_ = api.UpdateLaunch(ctx, item.ID, map[string]any{"status": "running", "pid": pid})

	err = cmd.Wait()
	exitCode := 0
	if err != nil {
		exitCode = 1
	}
	status := "completed"
	if exitCode != 0 {
		status = "failed"
	}
	_ = api.UpdateLaunch(ctx, item.ID, map[string]any{
		"status":    status,
		"exit_code": exitCode,
	})
	slog.Info("launch finished", "pid", strconv.Itoa(pid), "exit", exitCode)
}

func syncInstances(ctx context.Context, api *apiclient.Client, dataDir string) error {
	items, err := api.FetchDeviceInstances(ctx)
	if err != nil {
		return err
	}
	if err := cache.SaveInstances(dataDir, items); err != nil {
		return err
	}
	slog.Info("instances synced", "count", len(items))
	return nil
}
