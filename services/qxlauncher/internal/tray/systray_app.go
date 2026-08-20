package tray

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync/atomic"

	"fyne.io/systray"

	"github.com/qxproject/qx/pkg/safepath"
	"github.com/qxproject/qx/services/qxlauncher/internal/browser"
	"github.com/qxproject/qx/services/qxlauncher/internal/cache"
	"github.com/qxproject/qx/services/qxlauncher/internal/device"
	"github.com/qxproject/qx/services/qxlauncher/internal/logging"
	"github.com/qxproject/qx/services/qxlauncher/internal/notify"
)

type SystrayConfig struct {
	WebBaseURL       string
	LinkURL          string
	DeviceToken      string
	TokenPath        string
	APIBase          string
	DataDir          string
	LaunchDryRun     bool
	JavaPath         string
	SkipJavaDownload bool
	DeviceClient     *device.Client
	UserToken        string
	MaxLinkPolls     int
	PendingRegister  *device.RegisterResult
}

type systrayMenuItem struct{ item *systray.MenuItem }

func (m systrayMenuItem) disable() { m.item.Disable() }

func RunSystrayApp(cfg SystrayConfig) {
	if cfg.WebBaseURL == "" {
		cfg.WebBaseURL = "http://localhost:5173"
	}
	if cfg.MaxLinkPolls <= 0 {
		cfg.MaxLinkPolls = 60
	}
	launcherURL := LauncherPageURL(cfg.WebBaseURL)
	linkURL := cfg.LinkURL
	linked := cfg.DeviceToken != ""
	var currentToken = cfg.DeviceToken
	var loopStarted atomic.Bool

	systray.Run(func() {
		if linked {
			systray.SetIcon(iconLinkedPNG)
			systray.SetTooltip("QXLauncher — связан")
		} else {
			systray.SetIcon(iconPendingPNG)
			systray.SetTooltip("QXLauncher — ожидает привязки")
		}
		systray.SetTitle("QXLauncher")

		mLink := systray.AddMenuItem("Связать QXLauncher", "Открыть страницу привязки")
		mOpen := systray.AddMenuItem("Открыть сайт", launcherURL)
		mUnlink := systray.AddMenuItem("Отвязать", "Снять привязку с сайтом")
		systray.AddSeparator()
		instancesMenu := NewInstancesMenu(InstancesMenuConfig{
			APIBase:     cfg.APIBase,
			DeviceToken: cfg.DeviceToken,
			UserToken:   cfg.UserToken,
			AuthPath:    filepath.Join(cfg.DataDir, "user_auth.json"),
			DataDir:     cfg.DataDir,
		})
		instancesMenu.Attach()
		if linked {
			if snap, err := cache.LoadInstances(cfg.DataDir); err == nil && len(snap.Instances) > 0 {
				instancesMenu.Refresh(snap.Instances)
			}
		}
		systray.AddSeparator()
		mLogs := systray.AddMenuItem("Открыть логи", "Папка с qxlauncher.log")
		mQuit := systray.AddMenuItem("Выход", "")

		if linked {
			mLink.Disable()
		} else {
			mUnlink.Disable()
		}

		ctx, cancel := context.WithCancel(context.Background())
		var restartLoop func(string)
		restartLoop = func(token string) {
			if token == "" || !loopStarted.CompareAndSwap(false, true) {
				return
			}
			currentToken = token
			instancesMenu.SetDeviceToken(token)
			go RunLoop(ctx, Config{
				APIBase:           cfg.APIBase,
				DeviceToken:       token,
				TokenPath:         cfg.TokenPath,
				DeviceClient:      cfg.DeviceClient,
				DataDir:           cfg.DataDir,
				LaunchDryRun:      cfg.LaunchDryRun,
				JavaPath:          cfg.JavaPath,
				SkipJavaDownload:  cfg.SkipJavaDownload,
				OnInstancesSynced: instancesMenu.Refresh,
			})
		}

		if cfg.DeviceToken != "" {
			restartLoop(cfg.DeviceToken)
		} else if cfg.PendingRegister != nil {
			go pollDeviceLink(ctx, cfg, systrayMenuItem{mLink}, &linkURL, restartLoop, func() {
				systray.SetIcon(iconLinkedPNG)
				systray.SetTooltip("QXLauncher — связан")
				mLink.Disable()
				mUnlink.Enable()
			})
		}

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-mLink.ClickedCh:
					if currentToken == "" && cfg.DeviceClient != nil {
						go pollDeviceLink(ctx, cfg, systrayMenuItem{mLink}, &linkURL, restartLoop, func() {
							systray.SetIcon(iconLinkedPNG)
							systray.SetTooltip("QXLauncher — связан")
							mLink.Disable()
							mUnlink.Enable()
						})
						continue
					}
					target := linkURL
					if target == "" {
						target = launcherURL
					}
					if err := browser.Open(target); err != nil {
						slog.Warn("open link url failed", "err", err)
					}
				case <-mOpen.ClickedCh:
					if err := browser.Open(launcherURL); err != nil {
						slog.Warn("open launcher url failed", "err", err)
					}
				case <-mUnlink.ClickedCh:
					if currentToken == "" || cfg.DeviceClient == nil {
						continue
					}
					unlinkCtx := context.Background()
					if err := cfg.DeviceClient.UnlinkWithRefresh(unlinkCtx, cfg.TokenPath); err != nil {
						slog.Warn("device unlink failed", "err", err)
						notify.Show("QXLauncher", "Не удалось отвязать устройство")
						continue
					}
					if err := cfg.DeviceClient.ClearDeviceToken(cfg.TokenPath); err != nil {
						slog.Warn("clear device token failed", "err", err)
					}
					currentToken = ""
					loopStarted.Store(false)
					cancel()
					ctx, cancel = context.WithCancel(context.Background())
					mUnlink.Disable()
					mLink.Enable()
					systray.SetIcon(iconPendingPNG)
					systray.SetTooltip("QXLauncher — ожидает привязки")
					instancesMenu.Clear()
					notify.Show("QXLauncher", "Устройство отвязано")
				case <-mLogs.ClickedCh:
					logDir := logging.Dir(cfg.DataDir)
					if err := safepath.EnsureDir(logDir); err != nil {
						slog.Warn("create logs folder failed", "dir", logDir, "err", err)
						notify.Show("QXLauncher", "Не удалось открыть логи")
						continue
					}
					if err := browser.OpenFolder(logDir); err != nil {
						slog.Warn("open logs folder failed", "dir", logDir, "err", err)
						notify.Show("QXLauncher", "Не удалось открыть логи")
					}
				case <-mQuit.ClickedCh:
					cancel()
					systray.Quit()
					return
				}
			}
		}()
	}, func() {})
}
