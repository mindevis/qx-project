package launcher

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func instanceManagedByID(inst *models.LauncherInstance) string {
	if inst == nil || inst.ManagedByGameServerID == nil {
		return ""
	}
	return strings.TrimSpace(*inst.ManagedByGameServerID)
}

func (s *Service) InstanceIsServerManaged(ctx context.Context, instanceID string) (bool, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return false, ErrValidation
	}
	var inst models.LauncherInstance
	if err := s.db.WithContext(ctx).
		Select("id", "managed_by_game_server_id").
		Where("id = ?", instanceID).
		First(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrNotFound
		}
		return false, err
	}
	if instanceManagedByID(&inst) != "" {
		return true, nil
	}
	var n int64
	if err := s.db.WithContext(ctx).
		Model(&models.GameServerInstanceBinding{}).
		Where("launcher_instance_id = ?", instanceID).
		Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Service) AssertInstanceContentMutable(ctx context.Context, instanceID string) error {
	managed, err := s.InstanceIsServerManaged(ctx, instanceID)
	if err != nil {
		return err
	}
	if managed {
		return ErrInstanceManaged
	}
	return nil
}

func (s *Service) AnnotateInstancesManaged(ctx context.Context, items []models.LauncherInstance) []models.LauncherInstance {
	if len(items) == 0 {
		return items
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if instanceManagedByID(&item) == "" {
			ids = append(ids, item.ID)
		}
	}
	if len(ids) == 0 {
		return items
	}
	var bindings []models.GameServerInstanceBinding
	if err := s.db.WithContext(ctx).
		Where("launcher_instance_id IN ?", ids).
		Order("updated_at desc").
		Find(&bindings).Error; err != nil {
		return items
	}
	byInst := make(map[string]string, len(bindings))
	for _, binding := range bindings {
		if _, ok := byInst[binding.LauncherInstanceID]; ok {
			continue
		}
		byInst[binding.LauncherInstanceID] = binding.GameServerID
	}
	for i := range items {
		if instanceManagedByID(&items[i]) != "" {
			continue
		}
		gsID, ok := byInst[items[i].ID]
		if !ok || strings.TrimSpace(gsID) == "" {
			continue
		}
		id := gsID
		items[i].ManagedByGameServerID = &id
		_ = s.db.WithContext(ctx).
			Model(&models.LauncherInstance{}).
			Where("id = ? AND (managed_by_game_server_id IS NULL OR managed_by_game_server_id = '')", items[i].ID).
			Update("managed_by_game_server_id", id).Error
	}
	return items
}
