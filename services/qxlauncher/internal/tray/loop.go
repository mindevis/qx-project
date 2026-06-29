package tray

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qxproject/qx/services/qxlauncher/internal/apiclient"
	"github.com/qxproject/qx/services/qxlauncher/internal/cache"
	"github.com/qxproject/qx/services/qxlauncher/internal/config"
	"github.com/qxproject/qx/services/qxlauncher/internal/device"
	"github.com/qxproject/qx/services/qxlauncher/internal/minecraft"
	"github.com/qxproject/qx/services/qxlauncher/internal/notify"
)

type Config struct {
	APIBase          string
	DeviceToken      string
	TokenPath        string
	DeviceClient     *device.Client
	DataDir          string
	LaunchDryRun     bool
	JavaPath         string
	SkipJavaDownload bool
	LaunchPoll       time.Duration
	InstancePoll     time.Duration
}

var activeLaunches sync.Map

func RunLoop(ctx context.Context, cfg Config) {
	if cfg.LaunchPoll <= 0 {
		cfg.LaunchPoll = 2 * time.Second
	}
	if cfg.InstancePoll <= 0 {
		cfg.InstancePoll = 30 * time.Second
	}
	if cfg.DataDir == "" {
		home, _ := os.UserHomeDir()
		cfg.DataDir = config.UserDataDir(home)
	}

	api := apiclient.New(cfg.APIBase, cfg.DeviceToken)
	downloader := minecraft.NewDownloader(cfg.DataDir)
	downloader.JavaPath = cfg.JavaPath
	downloader.SkipJavaDownload = cfg.SkipJavaDownload

	launchTicker := time.NewTicker(cfg.LaunchPoll)
	defer launchTicker.Stop()
	syncTicker := time.NewTicker(cfg.InstancePoll)
	defer syncTicker.Stop()

	slog.Info("QXLauncher loop started", "launch_poll", cfg.LaunchPoll, "sync_poll", cfg.InstancePoll)

	for {
		select {
		case <-ctx.Done():
			return
		case <-syncTicker.C:
			if err := syncInstances(ctx, api, cfg.DataDir); err != nil {
				if apiclient.IsUnauthorized(err) && tryRefreshDeviceToken(ctx, api, cfg, err) {
					if err := syncInstances(ctx, api, cfg.DataDir); err != nil {
						logAPIFailure("instance sync failed", err)
					}
				} else {
					logAPIFailure("instance sync failed", err)
				}
			}
		case <-launchTicker.C:
			item, err := api.FetchPendingLaunch(ctx)
			if err != nil {
				if apiclient.IsUnauthorized(err) {
					tryRefreshDeviceToken(ctx, api, cfg, err)
					item, err = api.FetchPendingLaunch(ctx)
				}
				if err != nil {
					logAPIFailure("launch poll failed", err)
					continue
				}
			}
			if item == nil || item.Manifest == nil {
				continue
			}
			if _, loaded := activeLaunches.LoadOrStore(item.ID, true); loaded {
				continue
			}
			go func(launchID string, launchItem *apiclient.LaunchRequestItem) {
				defer activeLaunches.Delete(launchID)
				executeLaunch(context.Background(), api, downloader, cfg, launchItem)
			}(item.ID, item)
		}
	}
}

func executeLaunch(ctx context.Context, api *apiclient.Client, dl *minecraft.Downloader, cfg Config, item *apiclient.LaunchRequestItem) {
	username := "Player"
	offlineUUID := "00000000-0000-0000-0000-000000000000"
	skinModel := minecraft.ModelSteve
	var licensed *minecraft.LaunchAuth
	if item.Mojang != nil {
		licensed = &minecraft.LaunchAuth{
			Username:    item.Mojang.Username,
			UUID:        item.Mojang.UUID,
			AccessToken: item.Mojang.AccessToken,
		}
		username = item.Mojang.Username
		offlineUUID = item.Mojang.UUID
	} else if item.Profile != nil {
		username = item.Profile.Username
		offlineUUID = item.Profile.OfflineUUID
		skinModel = item.Profile.Model
	}

	var launchCosmetics *minecraft.LaunchCosmetics
	if item.Cosmetics != nil {
		launchCosmetics = &minecraft.LaunchCosmetics{
			SkinModel:      item.Cosmetics.SkinModel,
			SkinURL:        item.Cosmetics.SkinURL,
			UseSkinServer:  item.Cosmetics.UseSkinServer,
			SkinServerHost: item.Cosmetics.SkinServerHost,
			GameUUID:       item.Cosmetics.GameUUID,
		}
		if launchCosmetics.SkinModel == "" {
			launchCosmetics.SkinModel = skinModel
		}
	}

	dl.OnProgress = func(phase, message string) {
		fields := []any{"phase", phase, "message", message}
		fields = append(fields, minecraft.FormatLaunchLogFields(item.Manifest)...)
		slog.Info("launch prepare", fields...)
	}
	_ = api.UpdateLaunch(ctx, item.ID, map[string]any{"status": "running", "pid": 0})

	ready, err := dl.PrepareClientLaunch(ctx, minecraft.ClientLaunchInput{
		Manifest:    item.Manifest,
		Username:    username,
		OfflineUUID: offlineUUID,
		SkinModel:   skinModel,
		Licensed:    licensed,
		Cosmetics:   launchCosmetics,
	})
	dl.OnProgress = nil
	if err != nil {
		code := "PREPARE_FAILED"
		switch {
		case strings.Contains(err.Error(), "client jar"):
			code = "DOWNLOAD_FAILED"
		case strings.Contains(err.Error(), "libraries"):
			code = "LIBRARIES_FAILED"
		case strings.Contains(err.Error(), "assets"):
			code = "ASSETS_FAILED"
		case strings.Contains(err.Error(), "java"):
			code = "JAVA_FAILED"
		case strings.Contains(err.Error(), "loader install"):
			code = "LOADER_INSTALL_FAILED"
		}
		slog.Error("launch prepare failed", "err", err)
		_ = api.UpdateLaunch(ctx, item.ID, map[string]any{
			"status":     "failed",
			"error_code": code,
		})
		return
	}
	plan := ready.Plan

	label := minecraft.FormatLaunchLabel(item.Manifest, username)
	notify.Show("QXLauncher", "Запуск Minecraft: "+label)

	if cfg.LaunchDryRun {
		slog.Info("launch dry-run",
			append([]any{"main", plan.MainClass, "user", username, "game_dir", ready.GameDir}, minecraft.FormatLaunchLogFields(item.Manifest)...)...)
		_ = api.UpdateLaunch(ctx, item.ID, map[string]any{"status": "completed", "exit_code": 0})
		return
	}

	cmd, err := minecraft.StartClientProcess(context.Background(), plan, ready.LogPath)
	if err != nil {
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

func logAPIFailure(msg string, err error) {
	if apiclient.IsUnavailable(err) {
		slog.Debug(msg, "err", err)
		return
	}
	slog.Warn(msg, "err", err)
}

func tryRefreshDeviceToken(ctx context.Context, api *apiclient.Client, cfg Config, err error) bool {
	if !apiclient.IsUnauthorized(err) || cfg.DeviceClient == nil || cfg.TokenPath == "" {
		return false
	}
	token, refreshErr := device.RefreshDeviceToken(ctx, cfg.DeviceClient, cfg.TokenPath)
	if refreshErr != nil {
		slog.Warn("device token refresh failed", "err", refreshErr)
		return false
	}
	api.SetDeviceToken(token)
	slog.Info("device token refreshed")
	return true
}

func syncInstances(ctx context.Context, api *apiclient.Client, dataDir string) error {
	items, err := api.FetchDeviceInstances(ctx)
	if err != nil {
		return err
	}
	if err := cache.SyncInstances(dataDir, items); err != nil {
		return err
	}
	slog.Info("instances synced", "count", len(items))
	return nil
}
