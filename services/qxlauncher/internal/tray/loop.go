package tray

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxlauncher/internal/apiclient"
	"github.com/qxproject/qx/services/qxlauncher/internal/cache"
	"github.com/qxproject/qx/services/qxlauncher/internal/config"
	"github.com/qxproject/qx/services/qxlauncher/internal/device"
	"github.com/qxproject/qx/services/qxlauncher/internal/minecraft"
	"github.com/qxproject/qx/services/qxlauncher/internal/notify"
	"github.com/qxproject/qx/services/qxlauncher/internal/updater"
	"github.com/qxproject/qx/services/qxlauncher/internal/version"
)

type Config struct {
	APIBase           string
	DeviceToken       string
	TokenPath         string
	DeviceClient      *device.Client
	DataDir           string
	LaunchDryRun      bool
	JavaPath          string
	SkipJavaDownload  bool
	LaunchPoll        time.Duration
	InstancePoll      time.Duration
	ManifestBuilder   minecraft.ManifestBuilder
	OnInstancesSynced func([]apiclient.InstanceItem)
}

var activeLaunches sync.Map
var activeUpdates sync.Map
var activeModInstalls sync.Map
var activePrepares sync.Map
var activeInstanceFiles sync.Map
var activeModUninstalls sync.Map
var activeResourceUploads sync.Map
var activeResourceExports sync.Map

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
	downloader.ManifestBuilder = cfg.ManifestBuilder

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
			items, err := syncInstances(ctx, api, cfg.DataDir)
			if err != nil {
				if apiclient.IsUnauthorized(err) && tryRefreshDeviceToken(ctx, api, cfg, err) {
					items, err = syncInstances(ctx, api, cfg.DataDir)
					if err != nil {
						logAPIFailure("instance sync failed", err)
					}
				} else {
					logAPIFailure("instance sync failed", err)
				}
			}
			if err == nil && cfg.OnInstancesSynced != nil {
				cfg.OnInstancesSynced(items)
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
				}
			}
			if err == nil && item != nil && item.Instance != nil {
				if _, loaded := activeLaunches.LoadOrStore(item.ID, true); !loaded {
					go func(launchID string, launchItem *apiclient.LaunchRequestItem) {
						defer activeLaunches.Delete(launchID)
						executeLaunch(context.Background(), api, downloader, cfg, launchItem)
					}(item.ID, item)
				}
			}

			updateItem, updateErr := api.FetchPendingUpdate(ctx)
			if updateErr != nil {
				if apiclient.IsUnauthorized(updateErr) {
					tryRefreshDeviceToken(ctx, api, cfg, updateErr)
					updateItem, updateErr = api.FetchPendingUpdate(ctx)
				}
				if updateErr != nil {
					logAPIFailure("update poll failed", updateErr)
				}
			}
			if updateErr == nil && updateItem != nil {
				if _, loaded := activeUpdates.LoadOrStore(updateItem.ID, true); !loaded {
					go func(item *apiclient.UpdateRequestItem) {
						defer activeUpdates.Delete(item.ID)
						executeUpdate(context.Background(), api, item)
					}(updateItem)
				}
			}

			modItem, modErr := api.FetchPendingModInstall(ctx)
			if modErr != nil {
				if apiclient.IsUnauthorized(modErr) {
					tryRefreshDeviceToken(ctx, api, cfg, modErr)
					modItem, modErr = api.FetchPendingModInstall(ctx)
				}
				if modErr != nil {
					logAPIFailure("mod install poll failed", modErr)
				}
			}
			if modErr == nil && modItem != nil {
				if _, loaded := activeModInstalls.LoadOrStore(modItem.ID, true); !loaded {
					go func(item *apiclient.ModInstallRequestItem) {
						defer activeModInstalls.Delete(item.ID)
						executeModInstall(context.Background(), api, downloader, item)
					}(modItem)
				}
			}

			prepareItem, prepareErr := api.FetchPendingPrepare(ctx)
			if prepareErr != nil {
				if apiclient.IsUnauthorized(prepareErr) {
					tryRefreshDeviceToken(ctx, api, cfg, prepareErr)
					prepareItem, prepareErr = api.FetchPendingPrepare(ctx)
				}
				if prepareErr != nil {
					logAPIFailure("prepare poll failed", prepareErr)
				}
			}
			if prepareErr == nil && prepareItem != nil {
				if _, loaded := activePrepares.LoadOrStore(prepareItem.ID, true); !loaded {
					go func(item *apiclient.PrepareRequestItem) {
						defer activePrepares.Delete(item.ID)
						executePrepare(context.Background(), api, downloader, item)
					}(prepareItem)
				}
			}

			fileItem, fileErr := api.FetchPendingInstanceFile(ctx)
			if fileErr != nil {
				if apiclient.IsUnauthorized(fileErr) {
					tryRefreshDeviceToken(ctx, api, cfg, fileErr)
					fileItem, fileErr = api.FetchPendingInstanceFile(ctx)
				}
				if fileErr != nil {
					logAPIFailure("instance file poll failed", fileErr)
				}
			}
			if fileErr == nil && fileItem != nil {
				if _, loaded := activeInstanceFiles.LoadOrStore(fileItem.ID, true); !loaded {
					go func(item *apiclient.InstanceFileRequestItem) {
						defer activeInstanceFiles.Delete(item.ID)
						executeInstanceFile(context.Background(), api, downloader, item)
					}(fileItem)
				}
			}

			uninstallItem, uninstallErr := api.FetchPendingModUninstall(ctx)
			if uninstallErr != nil {
				if apiclient.IsUnauthorized(uninstallErr) {
					tryRefreshDeviceToken(ctx, api, cfg, uninstallErr)
					uninstallItem, uninstallErr = api.FetchPendingModUninstall(ctx)
				}
				if uninstallErr != nil {
					logAPIFailure("mod uninstall poll failed", uninstallErr)
				}
			}
			if uninstallErr == nil && uninstallItem != nil {
				if _, loaded := activeModUninstalls.LoadOrStore(uninstallItem.ID, true); !loaded {
					go func(item *apiclient.ModUninstallRequestItem) {
						defer activeModUninstalls.Delete(item.ID)
						executeModUninstall(context.Background(), api, downloader, item)
					}(uninstallItem)
				}
			}

			uploadItem, uploadErr := api.FetchPendingResourceUpload(ctx)
			if uploadErr != nil {
				if apiclient.IsUnauthorized(uploadErr) {
					tryRefreshDeviceToken(ctx, api, cfg, uploadErr)
					uploadItem, uploadErr = api.FetchPendingResourceUpload(ctx)
				}
				if uploadErr != nil {
					logAPIFailure("resource upload poll failed", uploadErr)
				}
			}
			if uploadErr == nil && uploadItem != nil {
				if _, loaded := activeResourceUploads.LoadOrStore(uploadItem.ID, true); !loaded {
					go func(item *apiclient.ResourceUploadRequestItem) {
						defer activeResourceUploads.Delete(item.ID)
						executeResourceUpload(context.Background(), api, downloader, item)
					}(uploadItem)
				}
			}

			exportItem, exportErr := api.FetchPendingResourceExport(ctx)
			if exportErr != nil {
				if apiclient.IsUnauthorized(exportErr) {
					tryRefreshDeviceToken(ctx, api, cfg, exportErr)
					exportItem, exportErr = api.FetchPendingResourceExport(ctx)
				}
				if exportErr != nil {
					logAPIFailure("resource export poll failed", exportErr)
				}
			}
			if exportErr == nil && exportItem != nil {
				if _, loaded := activeResourceExports.LoadOrStore(exportItem.ID, true); !loaded {
					go func(item *apiclient.ResourceExportRequestItem) {
						defer activeResourceExports.Delete(item.ID)
						executeResourceExport(context.Background(), api, downloader, item)
					}(exportItem)
				}
			}
		}
	}
}

func executeUpdate(ctx context.Context, api *apiclient.Client, item *apiclient.UpdateRequestItem) {
	notify.Show("QXLauncher", "Загрузка обновления…")
	if err := updater.Apply(ctx, item.DownloadURL, item.Filename, nil); err != nil {
		slog.Error("launcher update failed", "err", err)
		_ = api.CompleteUpdate(ctx, item.ID, "failed", "", "UPDATE_FAILED")
		return
	}
	_ = api.CompleteUpdate(ctx, item.ID, "completed", version.Version, "")
}

func modInstallFolder(resourceType string) string {
	switch resourceType {
	case "resourcepack":
		return "resourcepacks"
	case "shader":
		return "shaderpacks"
	case "datapack":
		return filepath.Join("saves", "world", "datapacks")
	default:
		return "mods"
	}
}

func executePrepare(ctx context.Context, api *apiclient.Client, dl *minecraft.Downloader, item *apiclient.PrepareRequestItem) {
	label := item.InstanceID
	if item.Instance != nil && item.Instance.Name != "" {
		label = item.Instance.Name
	}
	notify.Show("QXLauncher", "Установка "+label+"…")

	if item.Instance == nil {
		slog.Error("prepare missing instance metadata")
		_ = api.UpdatePrepareRequest(ctx, item.ID, "failed", "INSTANCE_MISSING")
		return
	}

	loaderVersion := ""
	if item.Instance.LoaderVersion != "" {
		loaderVersion = item.Instance.LoaderVersion
	}

	manifest, err := dl.BuildLaunchManifest(ctx, minecraft.LaunchInstance{
		ID:            item.Instance.ID,
		Name:          item.Instance.Name,
		MCVersion:     item.Instance.MCVersion,
		Loader:        item.Instance.Loader,
		LoaderVersion: loaderVersion,
		MaxMemoryMB:   optionalInt(item.Instance.MaxMemoryMB),
		MinMemoryMB:   optionalInt(item.Instance.MinMemoryMB),
		ExtraJVMArgs:  item.Instance.ExtraJVMArgs,
		WindowWidth:   item.Instance.WindowWidth,
		WindowHeight:  item.Instance.WindowHeight,
	})
	if err != nil {
		slog.Error("prepare manifest build failed", "err", err)
		_ = api.UpdatePrepareRequest(ctx, item.ID, "failed", "MANIFEST_UNAVAILABLE")
		return
	}

	dl.OnProgress = func(phase, message string) {
		fields := []any{"phase", phase, "message", message}
		fields = append(fields, minecraft.FormatLaunchLogFields(manifest)...)
		slog.Info("prepare", fields...)

		switch phase {
		case "loader-install", "prepare":
			switch message {
			case "java runtime":
				_ = api.UpdatePrepareRequest(ctx, item.ID, "preparing", "")
			case "client jar", "libraries", "natives", "assets":
				_ = api.UpdatePrepareRequest(ctx, item.ID, "downloading", "")
			default:
				if strings.Contains(message, "installer") {
					_ = api.UpdatePrepareRequest(ctx, item.ID, "preparing", "")
				}
			}
		}
	}

	if _, _, _, _, _, _, err := dl.PrepareInstanceGameFiles(ctx, manifest); err != nil {
		code := "PREPARE_FAILED"
		switch {
		case strings.Contains(err.Error(), "client jar"):
			code = "DOWNLOAD_FAILED"
		case strings.Contains(err.Error(), "natives:"):
			code = "NATIVES_FAILED"
		case strings.Contains(err.Error(), "libraries"):
			code = "LIBRARIES_FAILED"
		case strings.Contains(err.Error(), "assets"):
			code = "ASSETS_FAILED"
		case strings.Contains(err.Error(), "java"):
			code = "JAVA_FAILED"
		case strings.Contains(err.Error(), "loader install"):
			code = "LOADER_INSTALL_FAILED"
		}
		slog.Error("prepare failed", "err", err, "error_code", code)
		dl.OnProgress = nil
		_ = api.UpdatePrepareRequest(ctx, item.ID, "failed", code)
		return
	}
	dl.OnProgress = nil
	_ = api.UpdatePrepareRequest(ctx, item.ID, "completed", "")
}

func executeModInstall(ctx context.Context, api *apiclient.Client, dl *minecraft.Downloader, item *apiclient.ModInstallRequestItem) {
	label := item.ProjectName
	if label == "" {
		label = item.Filename
	}
	notify.Show("QXLauncher", "Установка "+label+"…")
	_ = api.CompleteModInstall(ctx, item.ID, "downloading", "")
	folder := modInstallFolder(item.ResourceType)
	if err := dl.InstallResource(ctx, item.InstanceID, folder, item.Filename, item.DownloadURL); err != nil {
		slog.Error("mod install failed", "project", item.ProjectID, "err", err)
		_ = api.CompleteModInstall(ctx, item.ID, "failed", "INSTALL_FAILED")
		return
	}
	// The server records the installed resource as part of marking the request
	// completed, so a failure here means the mod is on disk but not registered.
	// Retry a few times and log loudly instead of swallowing the error.
	if err := completeModInstallWithRetry(ctx, api, item.ID); err != nil {
		slog.Error("mod install completion report failed", "project", item.ProjectID, "file", item.Filename, "err", err)
	}
}

func completeModInstallWithRetry(ctx context.Context, api *apiclient.Client, requestID string) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if err = api.CompleteModInstall(ctx, requestID, "completed", ""); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}
	return err
}

func executeInstanceFile(ctx context.Context, api *apiclient.Client, dl *minecraft.Downloader, item *apiclient.InstanceFileRequestItem) {
	switch item.Operation {
	case "list":
		entries, err := dl.ListInstanceDir(item.InstanceID, item.Path)
		if err != nil {
			slog.Error("instance file list failed", "instance", item.InstanceID, "err", err)
			_ = api.CompleteInstanceFile(ctx, item.ID, "failed", "", "FILE_LIST_FAILED")
			return
		}
		raw, _ := json.Marshal(protocol.InstanceFilesListResult{Entries: entries})
		_ = api.CompleteInstanceFile(ctx, item.ID, "completed", string(raw), "")
	case "read":
		content, size, err := dl.ReadInstanceFile(item.InstanceID, item.Path)
		if err != nil {
			slog.Error("instance file read failed", "instance", item.InstanceID, "path", item.Path, "err", err)
			_ = api.CompleteInstanceFile(ctx, item.ID, "failed", "", "FILE_READ_FAILED")
			return
		}
		raw, _ := json.Marshal(protocol.InstanceFilesReadResult{
			Path:    item.Path,
			Content: content,
			Size:    size,
		})
		_ = api.CompleteInstanceFile(ctx, item.ID, "completed", string(raw), "")
	case "write":
		if err := dl.WriteInstanceFile(item.InstanceID, item.Path, item.WriteContent); err != nil {
			slog.Error("instance file write failed", "instance", item.InstanceID, "path", item.Path, "err", err)
			_ = api.CompleteInstanceFile(ctx, item.ID, "failed", "", "FILE_WRITE_FAILED")
			return
		}
		_ = api.CompleteInstanceFile(ctx, item.ID, "completed", `{"status":"ok"}`, "")
	default:
		_ = api.CompleteInstanceFile(ctx, item.ID, "failed", "", "UNKNOWN_OPERATION")
	}
}

func executeModUninstall(ctx context.Context, api *apiclient.Client, dl *minecraft.Downloader, item *apiclient.ModUninstallRequestItem) {
	folder := modInstallFolder(item.ResourceType)
	if err := dl.RemoveInstanceResourceFile(item.InstanceID, folder, item.Filename); err != nil {
		slog.Error("mod uninstall failed", "instance", item.InstanceID, "file", item.Filename, "err", err)
		_ = api.CompleteModUninstall(ctx, item.ID, "failed", "UNINSTALL_FAILED")
		return
	}
	_ = api.CompleteModUninstall(ctx, item.ID, "completed", "")
}

func executeResourceUpload(ctx context.Context, api *apiclient.Client, dl *minecraft.Downloader, item *apiclient.ResourceUploadRequestItem) {
	data, err := base64.StdEncoding.DecodeString(item.ContentB64)
	if err != nil {
		slog.Error("resource upload decode failed", "err", err)
		_ = api.CompleteResourceUpload(ctx, item.ID, "failed", "UPLOAD_DECODE_FAILED")
		return
	}
	folder := modInstallFolder(item.ResourceType)
	if err := dl.WriteInstanceResourceFile(item.InstanceID, folder, item.Filename, data); err != nil {
		slog.Error("resource upload write failed", "instance", item.InstanceID, "file", item.Filename, "err", err)
		_ = api.CompleteResourceUpload(ctx, item.ID, "failed", "UPLOAD_FAILED")
		return
	}
	_ = api.CompleteResourceUpload(ctx, item.ID, "completed", "")
}

func executeResourceExport(ctx context.Context, api *apiclient.Client, dl *minecraft.Downloader, item *apiclient.ResourceExportRequestItem) {
	folder := modInstallFolder(item.ResourceType)
	data, err := dl.ReadInstanceResourceFile(item.InstanceID, folder, item.Filename)
	if err != nil {
		slog.Error("resource export read failed", "instance", item.InstanceID, "file", item.Filename, "err", err)
		_ = api.CompleteResourceExport(ctx, item.ID, "failed", "", "EXPORT_READ_FAILED")
		return
	}
	contentB64 := base64.StdEncoding.EncodeToString(data)
	if err := api.CompleteResourceExport(ctx, item.ID, "completed", contentB64, ""); err != nil {
		slog.Error("resource export complete failed", "id", item.ID, "err", err)
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

	reportLaunchStatus := func(status string, extra map[string]any) {
		payload := map[string]any{"status": status}
		for k, v := range extra {
			payload[k] = v
		}
		_ = api.UpdateLaunch(ctx, item.ID, payload)
	}

	notifyLaunchStep := func(title string) {
		notify.Show("QXLauncher", title)
	}

	if item.Instance == nil {
		slog.Error("launch missing instance metadata")
		reportLaunchStatus("failed", map[string]any{"error_code": "INSTANCE_MISSING"})
		notifyLaunchStep("Не удалось подготовить запуск")
		return
	}

	reportLaunchStatus("preparing", nil)
	notifyLaunchStep("Подготовка к запуску…")

	manifest, err := dl.BuildLaunchManifest(ctx, minecraft.LaunchInstance{
		ID:            item.Instance.ID,
		Name:          item.Instance.Name,
		MCVersion:     item.Instance.MCVersion,
		Loader:        item.Instance.Loader,
		LoaderVersion: item.Instance.LoaderVersion,
		MaxMemoryMB:   optionalInt(item.Instance.MaxMemoryMB),
		MinMemoryMB:   optionalInt(item.Instance.MinMemoryMB),
		ExtraJVMArgs:  item.Instance.ExtraJVMArgs,
		WindowWidth:   item.Instance.WindowWidth,
		WindowHeight:  item.Instance.WindowHeight,
	})
	if err != nil {
		slog.Error("launch manifest build failed", "err", err)
		reportLaunchStatus("failed", map[string]any{"error_code": "MANIFEST_UNAVAILABLE"})
		notifyLaunchStep("Не удалось подготовить запуск")
		return
	}

	dl.OnProgress = func(phase, message string) {
		fields := []any{"phase", phase, "message", message}
		fields = append(fields, minecraft.FormatLaunchLogFields(manifest)...)
		slog.Info("launch prepare", fields...)

		switch phase {
		case "loader-install", "prepare":
			switch message {
			case "java runtime":
				reportLaunchStatus("preparing", nil)
			case "client jar", "libraries", "natives", "assets":
				reportLaunchStatus("downloading", nil)
			default:
				if strings.Contains(message, "installer") {
					reportLaunchStatus("preparing", nil)
				}
			}
		}
	}

	joinAddr := ""
	joinPort := 0
	if item.JoinServerAddress != nil {
		joinAddr = strings.TrimSpace(*item.JoinServerAddress)
	}
	if item.JoinServerPort != nil {
		joinPort = *item.JoinServerPort
	}
	if joinAddr != "" {
		slog.Info("launch join server", "address", joinAddr, "port", joinPort)
	}

	ready, err := dl.PrepareClientLaunch(ctx, minecraft.ClientLaunchInput{
		Manifest:          manifest,
		Username:          username,
		OfflineUUID:       offlineUUID,
		SkinModel:         skinModel,
		Licensed:          licensed,
		Cosmetics:         launchCosmetics,
		JoinServerAddress: joinAddr,
		JoinServerPort:    joinPort,
	})
	dl.OnProgress = nil
	if err != nil {
		code := "PREPARE_FAILED"
		switch {
		case strings.Contains(err.Error(), "client jar"):
			code = "DOWNLOAD_FAILED"
		case strings.Contains(err.Error(), "natives:"):
			code = "NATIVES_FAILED"
		case strings.Contains(err.Error(), "libraries"):
			code = "LIBRARIES_FAILED"
		case strings.Contains(err.Error(), "assets"):
			code = "ASSETS_FAILED"
		case strings.Contains(err.Error(), "java"):
			code = "JAVA_FAILED"
		case strings.Contains(err.Error(), "loader install"):
			code = "LOADER_INSTALL_FAILED"
		}
		slog.Error("launch prepare failed", "err", err, "error_code", code)
		reportLaunchStatus("failed", map[string]any{"error_code": code})
		msg := "Не удалось подготовить запуск"
		if code == "NATIVES_FAILED" {
			msg = "Не удалось подготовить LWJGL natives"
		}
		notifyLaunchStep(msg)
		return
	}
	plan := ready.Plan

	label := minecraft.FormatLaunchLabel(manifest, username)
	reportLaunchStatus("launching", nil)
	notifyLaunchStep("Запуск Minecraft: " + label)

	if cfg.LaunchDryRun {
		slog.Info("launch dry-run",
			append([]any{
				"main", plan.MainClass,
				"user", username,
				"game_dir", ready.GameDir,
				"natives_dir", filepath.Join(ready.GameDir, "natives"),
				"log_path", ready.LogPath,
			}, minecraft.FormatLaunchLogFields(manifest)...)...)
		reportLaunchStatus("completed", map[string]any{"exit_code": 0})
		return
	}

	slog.Info("starting minecraft process",
		append([]any{
			"java", plan.JavaBin,
			"main", plan.MainClass,
			"game_dir", ready.GameDir,
			"log_path", ready.LogPath,
			"cmd_file", filepath.Join(ready.GameDir, "launch.cmd.txt"),
		}, minecraft.FormatLaunchLogFields(manifest)...)...)

	cmd, err := minecraft.StartClientProcess(context.Background(), plan, ready.LogPath)
	if err != nil {
		slog.Error("java start failed", "err", err)
		reportLaunchStatus("failed", map[string]any{"error_code": "JAVA_START_FAILED"})
		notifyLaunchStep("Не удалось запустить Java")
		return
	}
	pid := cmd.Process.Pid
	reportLaunchStatus("running", map[string]any{"pid": pid})

	err = cmd.Wait()
	exitCode := 0
	if err != nil {
		exitCode = 1
	}
	status := "completed"
	errorCode := ""
	if exitCode != 0 {
		status = "failed"
		errorCode = "GAME_EXIT_" + strconv.Itoa(exitCode)
		if hint := minecraft.LaunchFailureHint(ready.LogPath, ready.GameDir); hint != "" {
			slog.Error("launch failed", "exit_code", exitCode, "hint", hint, "log_path", ready.LogPath)
			notifyLaunchStep("Minecraft завершился с ошибкой: " + hint)
			if strings.Contains(strings.ToLower(hint), "lwjgl") || strings.Contains(strings.ToLower(hint), "native") {
				errorCode = "NATIVES_LOAD_FAILED"
			}
		}
	}
	payload := map[string]any{"exit_code": exitCode}
	if errorCode != "" {
		payload["error_code"] = errorCode
	}
	reportLaunchStatus(status, payload)
	slog.Info("launch finished", "pid", strconv.Itoa(pid), "exit", exitCode)
}

func optionalInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
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

func syncInstances(ctx context.Context, api *apiclient.Client, dataDir string) ([]apiclient.InstanceItem, error) {
	items, err := api.FetchDeviceInstances(ctx)
	if err != nil {
		return nil, err
	}
	if err := cache.SyncInstances(dataDir, items); err != nil {
		return nil, err
	}
	slog.Info("instances synced", "count", len(items))
	return items, nil
}
