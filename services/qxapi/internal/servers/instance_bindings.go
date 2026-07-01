package servers

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

type InstanceBindingView struct {
	GameServerID       string `json:"game_server_id"`
	InstanceID         string `json:"instance_id"`
	InstanceName       string `json:"instance_name,omitempty"`
	InstanceMCVersion  string `json:"instance_mc_version,omitempty"`
	InstanceLoader     string `json:"instance_loader,omitempty"`
}

type bindingRow struct {
	models.GameServerInstanceBinding
	InstanceName      string `gorm:"column:instance_name"`
	InstanceMCVersion string `gorm:"column:instance_mc_version"`
	InstanceLoader    string `gorm:"column:instance_loader"`
}

func (s *Service) ListInstanceBindings(ctx context.Context, userID string) ([]InstanceBindingView, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrValidation
	}
	var rows []bindingRow
	err := s.db.WithContext(ctx).
		Table("game_server_instance_bindings").
		Select(`game_server_instance_bindings.*,
			launcher_instances.name AS instance_name,
			launcher_instances.mc_version AS instance_mc_version,
			launcher_instances.loader AS instance_loader`).
		Joins("JOIN launcher_instances ON launcher_instances.id = game_server_instance_bindings.launcher_instance_id").
		Where("game_server_instance_bindings.user_id = ?", userID).
		Order("game_server_instance_bindings.updated_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]InstanceBindingView, 0, len(rows))
	for _, row := range rows {
		out = append(out, bindingViewFromRow(&row))
	}
	return out, nil
}

func (s *Service) SetInstanceBinding(ctx context.Context, userID, gameServerID, instanceID string) (*InstanceBindingView, error) {
	userID = strings.TrimSpace(userID)
	gameServerID = strings.TrimSpace(gameServerID)
	instanceID = strings.TrimSpace(instanceID)
	if userID == "" || gameServerID == "" || instanceID == "" {
		return nil, ErrValidation
	}
	if err := s.assertBindableGameServer(ctx, userID, gameServerID); err != nil {
		return nil, err
	}
	var inst models.LauncherInstance
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", instanceID, userID).
		First(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var existing models.GameServerInstanceBinding
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND game_server_id = ?", userID, gameServerID).
		First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		existing = models.GameServerInstanceBinding{
			ID:                 uuid.NewString(),
			UserID:             userID,
			GameServerID:       gameServerID,
			LauncherInstanceID: inst.ID,
		}
		if err := s.db.WithContext(ctx).Create(&existing).Error; err != nil {
			return nil, err
		}
	} else {
		existing.LauncherInstanceID = inst.ID
		if err := s.db.WithContext(ctx).Save(&existing).Error; err != nil {
			return nil, err
		}
	}

	view := bindingViewFromRow(&bindingRow{
		GameServerInstanceBinding: existing,
		InstanceName:              inst.Name,
		InstanceMCVersion:         inst.MCVersion,
		InstanceLoader:            inst.Loader,
	})
	return &view, nil
}

func (s *Service) ClearInstanceBinding(ctx context.Context, userID, gameServerID string) error {
	userID = strings.TrimSpace(userID)
	gameServerID = strings.TrimSpace(gameServerID)
	if userID == "" || gameServerID == "" {
		return ErrValidation
	}
	res := s.db.WithContext(ctx).
		Where("user_id = ? AND game_server_id = ?", userID, gameServerID).
		Delete(&models.GameServerInstanceBinding{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ListBindableServers returns game servers the user may bind a launcher instance to:
// the user's own servers with a public address, plus public monitoring listings.
func (s *Service) ListBindableServers(ctx context.Context, userID string, in ListMonitoringInput) ([]MonitoringServerView, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrValidation
	}
	seen := make(map[string]struct{})
	out := make([]MonitoringServerView, 0)

	ownQuery := s.db.WithContext(ctx).
		Table("game_servers").
		Select("game_servers.*, users.tier AS owner_tier").
		Joins(monitoringJoinServers(s.db)).
		Joins(monitoringJoinUsers(s.db)).
		Where("servers.owner_id = ?", userID).
		Where("game_servers.address IS NOT NULL AND game_servers.address <> ''")

	mcVersion := strings.TrimSpace(in.MCVersion)
	if mcVersion != "" {
		ownQuery = ownQuery.Where("game_servers.mc_version = ?", mcVersion)
	}
	loader := strings.TrimSpace(in.Loader)
	if loader != "" {
		ownQuery = ownQuery.Where("game_servers.server_type = ?", loader)
	}

	var ownRows []monitoringRow
	if err := ownQuery.Order("game_servers.name ASC").Find(&ownRows).Error; err != nil {
		return nil, err
	}
	for _, row := range ownRows {
		view := monitoringViewFromRow(&row)
		out = append(out, view)
		seen[view.ID] = struct{}{}
	}

	public, err := s.ListMonitoringServers(ctx, in)
	if err != nil {
		return nil, err
	}
	for _, view := range public {
		if _, ok := seen[view.ID]; ok {
			continue
		}
		out = append(out, view)
	}
	return out, nil
}

func (s *Service) assertBindableGameServer(ctx context.Context, userID, gameServerID string) error {
	if _, _, err := s.getListedMonitoringServer(ctx, gameServerID); err == nil {
		return nil
	}
	_, err := s.getOwnedGameServerWithAddress(ctx, userID, gameServerID)
	return err
}

func (s *Service) getOwnedGameServerWithAddress(ctx context.Context, ownerID, gameServerID string) (*models.GameServer, error) {
	var row models.GameServer
	err := s.db.WithContext(ctx).
		Table("game_servers").
		Select("game_servers.*").
		Joins(monitoringJoinServers(s.db)).
		Where("game_servers.id = ?", gameServerID).
		Where("servers.owner_id = ?", ownerID).
		Where("game_servers.address IS NOT NULL AND game_servers.address <> ''").
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}

func bindingViewFromRow(row *bindingRow) InstanceBindingView {
	return InstanceBindingView{
		GameServerID:      row.GameServerID,
		InstanceID:        row.LauncherInstanceID,
		InstanceName:      row.InstanceName,
		InstanceMCVersion: row.InstanceMCVersion,
		InstanceLoader:    row.InstanceLoader,
	}
}
