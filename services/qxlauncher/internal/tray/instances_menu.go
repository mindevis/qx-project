package tray

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"fyne.io/systray"

	"github.com/qxproject/qx/services/qxlauncher/internal/apiclient"
	"github.com/qxproject/qx/services/qxlauncher/internal/auth"
	"github.com/qxproject/qx/services/qxlauncher/internal/browser"
	"github.com/qxproject/qx/services/qxlauncher/internal/cache"
	"github.com/qxproject/qx/services/qxlauncher/internal/notify"
)

type InstancesMenuConfig struct {
	APIBase     string
	DeviceToken string
	UserToken   string
	AuthPath    string
	DataDir     string
}

type InstancesMenu struct {
	mu           sync.Mutex
	cfg          InstancesMenuConfig
	root         *systray.MenuItem
	entries      []instanceMenuEntry
	cancelClicks context.CancelFunc
}

type instanceMenuEntry struct {
	root       *systray.MenuItem
	launch     *systray.MenuItem
	openFolder *systray.MenuItem
	delete     *systray.MenuItem
}

func NewInstancesMenu(cfg InstancesMenuConfig) *InstancesMenu {
	return &InstancesMenu{cfg: cfg}
}

func (m *InstancesMenu) SetDeviceToken(token string) {
	m.mu.Lock()
	m.cfg.DeviceToken = token
	m.mu.Unlock()
}

func (m *InstancesMenu) Attach() {
	m.root = systray.AddMenuItem("Инстансы", "Minecraft-инстансы на этом устройстве")
	m.root.Hide()
}

func (m *InstancesMenu) Clear() {
	m.Refresh(nil)
}

func (m *InstancesMenu) Refresh(items []apiclient.InstanceItem) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancelClicks != nil {
		m.cancelClicks()
		m.cancelClicks = nil
	}
	for _, entry := range m.entries {
		entry.root.Remove()
	}
	m.entries = nil

	if m.root == nil {
		return
	}
	if len(items) == 0 {
		m.root.Hide()
		return
	}
	m.root.Show()

	clickCtx, cancel := context.WithCancel(context.Background())
	m.cancelClicks = cancel

	for _, item := range items {
		entry := m.addEntry(item)
		m.entries = append(m.entries, entry)
		go m.watchEntry(clickCtx, item, entry)
	}
}

func (m *InstancesMenu) addEntry(item apiclient.InstanceItem) instanceMenuEntry {
	title := instanceMenuTitle(item)
	root := m.root.AddSubMenuItem(title, item.MCVersion+" · "+item.Loader)
	entry := instanceMenuEntry{root: root}
	entry.launch = root.AddSubMenuItem("Запуск", "Запустить Minecraft")
	entry.openFolder = root.AddSubMenuItem("Открыть папку", "Открыть каталог инстанса")
	entry.delete = root.AddSubMenuItem("Удалить", "Удалить инстанс с сайта")
	return entry
}

func (m *InstancesMenu) watchEntry(ctx context.Context, item apiclient.InstanceItem, entry instanceMenuEntry) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-entry.launch.ClickedCh:
			m.handleLaunch(item)
		case <-entry.openFolder.ClickedCh:
			m.handleOpenFolder(item.ID)
		case <-entry.delete.ClickedCh:
			m.handleDelete(item)
		}
	}
}

func (m *InstancesMenu) handleLaunch(item apiclient.InstanceItem) {
	userToken := m.resolveUserToken(context.Background())
	if userToken == "" {
		notify.Show("QXLauncher", "Войдите на сайте, чтобы запускать из трея")
		return
	}
	m.mu.Lock()
	deviceToken := m.cfg.DeviceToken
	m.mu.Unlock()
	api := apiclient.New(m.cfg.APIBase, deviceToken)
	profileID := m.defaultProfileID(context.Background(), api, userToken)
	_, err := api.CreateLaunchRequest(context.Background(), userToken, item.ID, profileID, false)
	if err != nil {
		slog.Warn("tray launch request failed", "instance", item.ID, "err", err)
		notify.Show("QXLauncher", "Не удалось отправить запуск: "+item.Name)
		return
	}
	notify.Show("QXLauncher", "Запуск: "+item.Name)
}

func (m *InstancesMenu) handleOpenFolder(instanceID string) {
	m.mu.Lock()
	dataDir := m.cfg.DataDir
	m.mu.Unlock()
	dir := instanceDir(dataDir, instanceID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		slog.Warn("create instance dir failed", "dir", dir, "err", err)
		notify.Show("QXLauncher", "Не удалось открыть папку")
		return
	}
	if err := browser.OpenFolder(dir); err != nil {
		slog.Warn("open instance folder failed", "dir", dir, "err", err)
		notify.Show("QXLauncher", "Не удалось открыть папку")
	}
}

func (m *InstancesMenu) handleDelete(item apiclient.InstanceItem) {
	userToken := m.resolveUserToken(context.Background())
	if userToken == "" {
		notify.Show("QXLauncher", "Войдите на сайте, чтобы удалять инстансы")
		return
	}
	m.mu.Lock()
	deviceToken := m.cfg.DeviceToken
	dataDir := m.cfg.DataDir
	m.mu.Unlock()
	api := apiclient.New(m.cfg.APIBase, deviceToken)
	if err := api.DeleteInstance(context.Background(), userToken, item.ID); err != nil {
		slog.Warn("tray delete instance failed", "instance", item.ID, "err", err)
		notify.Show("QXLauncher", "Не удалось удалить: "+item.Name)
		return
	}
	if snap, err := cache.LoadInstances(dataDir); err == nil {
		filtered := make([]apiclient.InstanceItem, 0, len(snap.Instances))
		for _, inst := range snap.Instances {
			if inst.ID != item.ID {
				filtered = append(filtered, inst)
			}
		}
		_ = cache.SyncInstances(dataDir, filtered)
		m.Refresh(filtered)
	}
	notify.Show("QXLauncher", "Инстанс удалён: "+item.Name)
}

func (m *InstancesMenu) defaultProfileID(ctx context.Context, api *apiclient.Client, userToken string) string {
	profiles, err := api.ListProfiles(ctx, userToken)
	if err != nil || len(profiles) == 0 {
		return ""
	}
	return profiles[0].ID
}

func (m *InstancesMenu) resolveUserToken(ctx context.Context) string {
	if m.cfg.AuthPath != "" {
		if token, err := auth.EnsureFreshAccessToken(ctx, m.cfg.APIBase, m.cfg.AuthPath); err == nil && token != "" {
			return token
		}
	}
	return m.cfg.UserToken
}

func instanceMenuTitle(item apiclient.InstanceItem) string {
	name := item.Name
	if name == "" {
		name = item.ID
	}
	if item.MCVersion != "" {
		return name + " (" + item.MCVersion + ")"
	}
	return name
}

func instanceDir(dataDir, instanceID string) string {
	return filepath.Join(cache.InstanceDataRoot(dataDir), instanceID)
}
