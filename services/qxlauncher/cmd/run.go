package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	qxlog "github.com/qxproject/qx/pkg/log"
	"github.com/qxproject/qx/services/qxlauncher/internal/apiclient"
	"github.com/qxproject/qx/services/qxlauncher/internal/auth"
	"github.com/qxproject/qx/services/qxlauncher/internal/config"
	"github.com/qxproject/qx/services/qxlauncher/internal/device"
	"github.com/qxproject/qx/services/qxlauncher/internal/notify"
	"github.com/qxproject/qx/services/qxlauncher/internal/tray"
)

func run() {
	cfg := config.Load()
	qxlog.Setup(qxlog.Options{Level: cfg.LogLevel, Format: cfg.LogFormat})

	apiBase := cfg.APIBaseURL
	webBase := cfg.WebBaseURL
	tokenPath := cfg.DeviceTokenPath
	dataDir := filepath.Dir(tokenPath)
	authPath := filepath.Join(dataDir, "user_auth.json")
	maxPolls := cfg.LinkMaxPolls

	resolvedID := cfg.DeviceID
	if resolvedID == "" {
		resolvedID = device.ResolveDeviceID(dataDir)
	}
	client := device.NewClient(apiBase, resolvedID)

	ctx := context.Background()

	userToken := resolveUserToken(ctx, apiBase, authPath, cfg)

	deviceToken, err := device.EnsureDeviceToken(ctx, client, tokenPath)
	if err != nil {
		slog.Warn("device token check failed", "err", err)
	}
	if deviceToken != "" {
		_ = device.SaveDeviceID(dataDir, client.DeviceID)
	}
	var linkURL string
	var pendingReg *device.RegisterResult

	if deviceToken == "" {
		reg, err := client.Register(ctx)
		if err != nil {
			if apiclient.IsUnavailable(err) {
				slog.Warn("backend unavailable, device registration deferred", "err", err)
				notify.Show("QXLauncher", "Сервер недоступен. Лаунчер работает в автономном режиме.")
			} else {
				slog.Error("device register failed", "err", err)
			}
		} else {
			if err := device.SaveDeviceID(dataDir, reg.DeviceID); err != nil {
				slog.Warn("save device id failed", "err", err)
			}
			client.DeviceID = reg.DeviceID
			pendingReg = reg
			linkURL = reg.LinkURL
			tray.OpenLinkPage(reg.LinkURL)
			notify.Show("QXLauncher", "Подтвердите привязку в браузере")
			slog.Info("device registered",
				"device_id", reg.DeviceID,
				"link_url", reg.LinkURL,
			)
		}

		if cfg.SkipTray && pendingReg != nil {
			if maxPolls <= 0 {
				return
			}
			token, err := tray.CompleteDeviceLink(ctx, client, tray.DeviceLinkConfig{
				TokenPath:    tokenPath,
				MaxLinkPolls: maxPolls,
				UserToken:    userToken,
			}, pendingReg)
			if err != nil {
				slog.Warn("device link pending", "err", err)
				return
			}
			if token == "" {
				slog.Warn("device linked without token")
				return
			}
			deviceToken = token
			notify.Show("QXLauncher", "Лаунчер связан с сайтом")
			slog.Info("device linked", "token_path", tokenPath)
		}
	}

	if cfg.SkipTray {
		if deviceToken != "" {
			tray.RunLoop(ctx, tray.Config{
				APIBase:          apiBase,
				DeviceToken:      deviceToken,
				TokenPath:        tokenPath,
				DeviceClient:     client,
				DataDir:          dataDir,
				LaunchDryRun:     cfg.LaunchDryRun,
				JavaPath:         cfg.JavaPath,
				SkipJavaDownload: cfg.SkipJavaDownload,
			})
		}
		return
	}

	tray.RunSystrayApp(tray.SystrayConfig{
		WebBaseURL:       webBase,
		LinkURL:          linkURL,
		DeviceToken:      deviceToken,
		TokenPath:        tokenPath,
		APIBase:          apiBase,
		DataDir:          dataDir,
		LaunchDryRun:     cfg.LaunchDryRun,
		JavaPath:         cfg.JavaPath,
		SkipJavaDownload: cfg.SkipJavaDownload,
		DeviceClient:     client,
		UserToken:       userToken,
		MaxLinkPolls:    maxPolls,
		PendingRegister: pendingReg,
	})
}

func resolveUserToken(ctx context.Context, apiBase, authPath string, cfg config.Config) string {
	if cfg.Email != "" {
		if cfg.Password == "" {
			slog.Warn("email set without password in launcher config")
			return ""
		}
		session, err := auth.Login(ctx, apiBase, cfg.Email, cfg.Password)
		if err != nil {
			slog.Warn("qx login failed", "err", err)
			return ""
		}
		if err := auth.SaveSession(authPath, session); err != nil {
			slog.Warn("save user session failed", "err", err)
		} else {
			slog.Info("qx user logged in", "path", authPath)
		}
		return session.AccessToken
	}
	token, err := auth.EnsureFreshAccessToken(ctx, apiBase, authPath)
	if err != nil && !os.IsNotExist(err) && !apiclient.IsUnavailable(err) {
		slog.Warn("session refresh failed", "err", err)
	}
	return token
}
