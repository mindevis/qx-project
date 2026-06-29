package servers

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

type MonitoringServerView struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	ServerType    string   `json:"server_type"`
	MCVersion     string   `json:"mc_version"`
	LoaderVersion *string  `json:"loader_version,omitempty"`
	Address       string   `json:"address"`
	Port          int      `json:"port"`
	Status        string   `json:"status"`
	IsOnline      bool     `json:"is_online"`
	IsPremium     bool     `json:"is_premium"`
	Description   string   `json:"description,omitempty"`
	BannerURL     string   `json:"banner_url,omitempty"`
	Tags          []string `json:"tags"`
	Mods          []string `json:"mods"`
	Plugins       []string `json:"plugins"`
	LikesCount    int      `json:"likes_count"`
	RatingAvg     float64  `json:"rating_avg"`
	RatingCount   int      `json:"rating_count"`
}

type ListMonitoringInput struct {
	MCVersion  string
	Loader     string
	Mod        string
	Plugin     string
	Query      string
}

type monitoringRow struct {
	models.GameServer
	OwnerTier string `gorm:"column:owner_tier"`
}

func (s *Service) ListMonitoringServers(ctx context.Context, in ListMonitoringInput) ([]MonitoringServerView, error) {
	query := s.db.WithContext(ctx).
		Table("game_servers").
		Select("game_servers.*, users.tier AS owner_tier").
		Joins("JOIN servers ON servers.id = game_servers.server_id").
		Joins("JOIN users ON users.id = servers.owner_id").
		Where("game_servers.show_in_monitoring = ?", true).
		Where("game_servers.address IS NOT NULL AND game_servers.address <> ''")

	mcVersion := strings.TrimSpace(in.MCVersion)
	if mcVersion != "" {
		query = query.Where("game_servers.mc_version = ?", mcVersion)
	}
	loader := strings.TrimSpace(in.Loader)
	if loader != "" {
		query = query.Where("game_servers.server_type = ?", loader)
	}
	mod := strings.ToLower(strings.TrimSpace(in.Mod))
	if mod != "" {
		query = query.Where("LOWER(game_servers.monitoring_mods_json) LIKE ?", "%"+mod+"%")
	}
	plugin := strings.ToLower(strings.TrimSpace(in.Plugin))
	if plugin != "" {
		query = query.Where("LOWER(game_servers.monitoring_plugins_json) LIKE ?", "%"+plugin+"%")
	}
	q := strings.ToLower(strings.TrimSpace(in.Query))
	if q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"LOWER(game_servers.name) LIKE ? OR LOWER(game_servers.monitoring_description) LIKE ? OR LOWER(game_servers.monitoring_tags_json) LIKE ?",
			like, like, like,
		)
	}

	var rows []monitoringRow
	if err := query.
		Order("CASE WHEN users.tier = 'premium' THEN 0 ELSE 1 END").
		Order("game_servers.likes_count DESC").
		Order("game_servers.created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]MonitoringServerView, 0, len(rows))
	for _, row := range rows {
		out = append(out, monitoringViewFromRow(&row))
	}
	return out, nil
}

func (s *Service) LikeMonitoringServer(ctx context.Context, userID, gameServerID string) (*MonitoringServerView, error) {
	userID = strings.TrimSpace(userID)
	gameServerID = strings.TrimSpace(gameServerID)
	if userID == "" || gameServerID == "" {
		return nil, ErrValidation
	}
	item, ownerTier, err := s.getListedMonitoringServer(ctx, gameServerID)
	if err != nil {
		return nil, err
	}

	var feedback models.GameServerMonitoringFeedback
	err = s.db.WithContext(ctx).
		Where("user_id = ? AND game_server_id = ?", userID, gameServerID).
		First(&feedback).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		feedback = models.GameServerMonitoringFeedback{
			ID:           uuid.NewString(),
			UserID:       userID,
			GameServerID: gameServerID,
			Liked:        true,
		}
		if err := s.db.WithContext(ctx).Create(&feedback).Error; err != nil {
			return nil, err
		}
		if err := s.db.WithContext(ctx).Model(item).UpdateColumn("likes_count", gorm.Expr("likes_count + 1")).Error; err != nil {
			return nil, err
		}
		item.LikesCount++
	} else if !feedback.Liked {
		feedback.Liked = true
		if err := s.db.WithContext(ctx).Save(&feedback).Error; err != nil {
			return nil, err
		}
		if err := s.db.WithContext(ctx).Model(item).UpdateColumn("likes_count", gorm.Expr("likes_count + 1")).Error; err != nil {
			return nil, err
		}
		item.LikesCount++
	}

	row := monitoringRow{GameServer: *item, OwnerTier: ownerTier}
	view := monitoringViewFromRow(&row)
	return &view, nil
}

func (s *Service) RateMonitoringServer(ctx context.Context, userID, gameServerID string, rating int) (*MonitoringServerView, error) {
	userID = strings.TrimSpace(userID)
	gameServerID = strings.TrimSpace(gameServerID)
	if userID == "" || gameServerID == "" || rating < 1 || rating > 5 {
		return nil, ErrValidation
	}
	item, ownerTier, err := s.getListedMonitoringServer(ctx, gameServerID)
	if err != nil {
		return nil, err
	}

	var feedback models.GameServerMonitoringFeedback
	err = s.db.WithContext(ctx).
		Where("user_id = ? AND game_server_id = ?", userID, gameServerID).
		First(&feedback).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	updates := map[string]any{}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		feedback = models.GameServerMonitoringFeedback{
			ID:           uuid.NewString(),
			UserID:       userID,
			GameServerID: gameServerID,
			Rating:       &rating,
		}
		if err := s.db.WithContext(ctx).Create(&feedback).Error; err != nil {
			return nil, err
		}
		updates["rating_sum"] = gorm.Expr("rating_sum + ?", rating)
		updates["rating_count"] = gorm.Expr("rating_count + 1")
		item.RatingSum += rating
		item.RatingCount++
	} else {
		old := 0
		if feedback.Rating != nil {
			old = *feedback.Rating
		}
		feedback.Rating = &rating
		if err := s.db.WithContext(ctx).Save(&feedback).Error; err != nil {
			return nil, err
		}
		if old == 0 {
			updates["rating_sum"] = gorm.Expr("rating_sum + ?", rating)
			updates["rating_count"] = gorm.Expr("rating_count + 1")
			item.RatingSum += rating
			item.RatingCount++
		} else {
			updates["rating_sum"] = gorm.Expr("rating_sum + ? - ?", rating, old)
			item.RatingSum += rating - old
		}
	}
	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(item).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	row := monitoringRow{GameServer: *item, OwnerTier: ownerTier}
	view := monitoringViewFromRow(&row)
	return &view, nil
}

func (s *Service) refreshMonitoringSnapshots(ctx context.Context, ownerID, vpsID string, item *models.GameServer) {
	if item == nil || !item.ShowInMonitoring || !gameServerIsInstalled(item) {
		return
	}
	if s.hub == nil || !s.hub.IsOnline(vpsID) {
		return
	}
	entries, err := s.ListGameServerMods(ctx, ownerID, vpsID, item.ID)
	if err != nil {
		return
	}
	mods, plugins := splitModPluginNames(item.ServerType, entries)
	modsJSON, _ := json.Marshal(mods)
	pluginsJSON, _ := json.Marshal(plugins)
	_ = s.db.WithContext(ctx).Model(item).Updates(map[string]any{
		"monitoring_mods_json":     string(modsJSON),
		"monitoring_plugins_json":  string(pluginsJSON),
		"updated_at":               item.UpdatedAt,
	}).Error
	item.MonitoringModsJSON = string(modsJSON)
	item.MonitoringPluginsJSON = string(pluginsJSON)
}

func (s *Service) getListedMonitoringServer(ctx context.Context, gameServerID string) (*models.GameServer, string, error) {
	var row monitoringRow
	err := s.db.WithContext(ctx).
		Table("game_servers").
		Select("game_servers.*, users.tier AS owner_tier").
		Joins("JOIN servers ON servers.id = game_servers.server_id").
		Joins("JOIN users ON users.id = servers.owner_id").
		Where("game_servers.id = ? AND game_servers.show_in_monitoring = ?", gameServerID, true).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}
	if row.Address == nil || strings.TrimSpace(*row.Address) == "" {
		return nil, "", ErrNotFound
	}
	item := row.GameServer
	return &item, row.OwnerTier, nil
}

func monitoringViewFromRow(row *monitoringRow) MonitoringServerView {
	item := &row.GameServer
	address := ""
	if item.Address != nil {
		address = strings.TrimSpace(*item.Address)
	}
	ratingAvg := 0.0
	if item.RatingCount > 0 {
		ratingAvg = math.Round((float64(item.RatingSum)/float64(item.RatingCount))*10) / 10
	}
	return MonitoringServerView{
		ID:            item.ID,
		Name:          item.Name,
		ServerType:    item.ServerType,
		MCVersion:     item.MCVersion,
		LoaderVersion: item.LoaderVersion,
		Address:       address,
		Port:          item.Port,
		Status:        item.Status,
		IsOnline:      item.Status == models.GameServerStatusRunning,
		IsPremium:     strings.EqualFold(row.OwnerTier, "premium"),
		Description:   strings.TrimSpace(item.MonitoringDescription),
		BannerURL:     strings.TrimSpace(item.BannerURL),
		Tags:          decodeStringListJSON(item.MonitoringTagsJSON),
		Mods:          decodeStringListJSON(item.MonitoringModsJSON),
		Plugins:       decodeStringListJSON(item.MonitoringPluginsJSON),
		LikesCount:    item.LikesCount,
		RatingAvg:     ratingAvg,
		RatingCount:   item.RatingCount,
	}
}

func splitModPluginNames(serverType string, entries []protocol.FileEntry) ([]string, []string) {
	mods := []string{}
	plugins := []string{}
	usesMods := gameServerTypeUsesMods(serverType)
	usesPlugins := gameServerTypeUsesPlugins(serverType)
	for _, e := range entries {
		if e.Dir {
			continue
		}
		name := strings.TrimSuffix(strings.TrimSpace(e.Name), ".jar")
		if name == "" {
			continue
		}
		switch {
		case usesMods && !usesPlugins:
			mods = append(mods, name)
		case usesPlugins && !usesMods:
			plugins = append(plugins, name)
		case usesMods:
			mods = append(mods, name)
		default:
			plugins = append(plugins, name)
		}
	}
	return mods, plugins
}

func gameServerTypeUsesMods(serverType string) bool {
	switch strings.ToLower(strings.TrimSpace(serverType)) {
	case "forge", "neoforge", "fabric", "quilt", "mohist", "magma", "arclight":
		return true
	default:
		return false
	}
}

func gameServerTypeUsesPlugins(serverType string) bool {
	switch strings.ToLower(strings.TrimSpace(serverType)) {
	case "paper", "spigot", "purpur", "mohist", "magma", "arclight":
		return true
	default:
		return false
	}
}

func decodeStringListJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	return out
}

func encodeStringListJSON(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	out, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(out)
}
