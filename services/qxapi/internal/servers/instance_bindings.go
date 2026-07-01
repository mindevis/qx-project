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
	if _, _, err := s.getListedMonitoringServer(ctx, gameServerID); err != nil {
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

func bindingViewFromRow(row *bindingRow) InstanceBindingView {
	return InstanceBindingView{
		GameServerID:      row.GameServerID,
		InstanceID:        row.LauncherInstanceID,
		InstanceName:      row.InstanceName,
		InstanceMCVersion: row.InstanceMCVersion,
		InstanceLoader:    row.InstanceLoader,
	}
}
