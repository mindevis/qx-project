package tray

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/qxproject/qx/services/qxlauncher/internal/apiclient"
	"github.com/qxproject/qx/services/qxlauncher/internal/device"
	"github.com/qxproject/qx/services/qxlauncher/internal/notify"
)

var deviceLinkPollActive atomic.Bool

type DeviceLinkConfig struct {
	TokenPath    string
	MaxLinkPolls int
	UserToken    string
}

func CompleteDeviceLink(
	ctx context.Context,
	client *device.Client,
	cfg DeviceLinkConfig,
	reg *device.RegisterResult,
) (string, error) {
	return completeDeviceLink(ctx, client, cfg, reg)
}

func completeDeviceLink(
	ctx context.Context,
	client *device.Client,
	cfg DeviceLinkConfig,
	reg *device.RegisterResult,
) (string, error) {
	status, err := client.PollUntilLinked(ctx, cfg.MaxLinkPolls, reg.PollIntervalSec, time.Sleep, cfg.UserToken)
	if err != nil {
		return "", err
	}
	if status.DeviceToken == nil {
		return "", nil
	}
	token := *status.DeviceToken
	if err := client.SaveDeviceToken(cfg.TokenPath, token); err != nil {
		return "", err
	}
	_ = device.SaveDeviceID(filepath.Dir(cfg.TokenPath), reg.DeviceID)
	return token, nil
}

func pollDeviceLink(
	ctx context.Context,
	cfg SystrayConfig,
	mLink menuDisabler,
	linkURL *string,
	startLoop func(string),
	onLinked func(),
) {
	if !deviceLinkPollActive.CompareAndSwap(false, true) {
		slog.Debug("device link poll already in progress")
		return
	}
	defer deviceLinkPollActive.Store(false)

	reg := cfg.PendingRegister
	if reg == nil {
		var err error
		reg, err = cfg.DeviceClient.Register(ctx)
		if err != nil {
			if apiclient.IsUnavailable(err) {
				slog.Warn("backend unavailable, device link deferred", "err", err)
				notify.Show("QXLauncher", "Сервер недоступен. Повторите привязку позже.")
			} else {
				slog.Error("device register failed", "err", err)
			}
			return
		}
	}
	*linkURL = reg.LinkURL
	OpenLinkPage(reg.LinkURL)
	notify.Show("QXLauncher", "Подтвердите привязку в браузере")
	slog.Info("device registered", "device_id", reg.DeviceID, "link_url", reg.LinkURL)

	token, err := completeDeviceLink(ctx, cfg.DeviceClient, DeviceLinkConfig{
		TokenPath:    cfg.TokenPath,
		MaxLinkPolls: cfg.MaxLinkPolls,
		UserToken:    cfg.UserToken,
	}, reg)
	if err != nil {
		slog.Warn("device link pending", "err", err)
		return
	}
	if token == "" {
		return
	}
	mLink.disable()
	if onLinked != nil {
		onLinked()
	}
	notify.Show("QXLauncher", "Лаунчер связан с сайтом")
	slog.Info("device linked", "token_path", cfg.TokenPath)
	startLoop(token)
}

type menuDisabler interface {
	disable()
}
