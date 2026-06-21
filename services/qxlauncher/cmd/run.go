package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	qxlog "github.com/qxproject/qx/pkg/log"
	"github.com/qxproject/qx/services/qxlauncher/internal/auth"
	"github.com/qxproject/qx/services/qxlauncher/internal/device"
	"github.com/qxproject/qx/services/qxlauncher/internal/notify"
	"github.com/qxproject/qx/services/qxlauncher/internal/tray"
)

func run() {
	qxlog.SetupFromEnv()

	apiBase := os.Getenv("QX_API_BASE_URL")
	if apiBase == "" {
		apiBase = "http://localhost:3000/api/v1"
	}
	webBase := os.Getenv("QX_WEB_BASE_URL")
	if webBase == "" {
		webBase = "http://localhost:5173"
	}
	tokenPath := os.Getenv("QX_DEVICE_TOKEN_PATH")
	if tokenPath == "" {
		home, _ := os.UserHomeDir()
		tokenPath = filepath.Join(home, ".qx", "device_token")
	}
	dataDir := filepath.Dir(tokenPath)
	authPath := filepath.Join(dataDir, "user_auth.json")
	maxPolls := 60
	if v := os.Getenv("QX_LINK_MAX_POLLS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxPolls = n
		}
	}

	resolvedID := device.ResolveDeviceID(dataDir)
	client := device.NewClient(apiBase, resolvedID)

	ctx := context.Background()

	userToken := resolveUserToken(ctx, apiBase, authPath)

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
			slog.Error("device register failed", "err", err)
			return
		}
		if err := device.SaveDeviceID(dataDir, reg.DeviceID); err != nil {
			slog.Warn("save device id failed", "err", err)
		}
		client.DeviceID = reg.DeviceID
		pendingReg = reg
		linkURL = reg.LinkURL
		notify.Show("QXLauncher", "Свяжите лаунчер с сайтом: "+reg.UserCode)
		slog.Info("device registered",
			"device_id", reg.DeviceID,
			"user_code", reg.UserCode,
			"link_url", reg.LinkURL,
		)

		if os.Getenv("QX_SKIP_TRAY") == "1" {
			if maxPolls <= 0 {
				return
			}
			token, err := tray.CompleteDeviceLink(ctx, client, tray.DeviceLinkConfig{
				TokenPath:    tokenPath,
				MaxLinkPolls: maxPolls,
				UserToken:    userToken,
			}, reg)
			if err != nil {
				slog.Warn("device link pending", "err", err, "hint", "open link_url in browser")
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

	if os.Getenv("QX_SKIP_TRAY") == "1" {
		if deviceToken != "" {
			dryRun := os.Getenv("QX_LAUNCH_DRY_RUN") == "1"
			tray.RunLoop(ctx, tray.Config{
				APIBase:      apiBase,
				DeviceToken:  deviceToken,
				TokenPath:    tokenPath,
				DeviceClient: client,
				DataDir:      dataDir,
				LaunchDryRun: dryRun,
			})
		}
		return
	}

	dryRun := os.Getenv("QX_LAUNCH_DRY_RUN") == "1"
	tray.RunSystrayApp(tray.SystrayConfig{
		WebBaseURL:      webBase,
		LinkURL:         linkURL,
		DeviceToken:     deviceToken,
		TokenPath:       tokenPath,
		APIBase:         apiBase,
		DataDir:         dataDir,
		LaunchDryRun:    dryRun,
		DeviceClient:    client,
		UserToken:       userToken,
		MaxLinkPolls:    maxPolls,
		PendingRegister: pendingReg,
	})
}

func resolveUserToken(ctx context.Context, apiBase, authPath string) string {
	if email := os.Getenv("QX_EMAIL"); email != "" {
		password := os.Getenv("QX_PASSWORD")
		if password == "" {
			slog.Warn("QX_EMAIL set without QX_PASSWORD")
			return ""
		}
		session, err := auth.Login(ctx, apiBase, email, password)
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
	if err != nil {
		slog.Warn("session refresh failed", "err", err)
	}
	return token
}
