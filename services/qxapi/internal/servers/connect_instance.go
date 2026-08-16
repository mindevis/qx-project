package servers

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/launcher"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

func (s *Service) EnsureConnectBinding(
	ctx context.Context,
	userID, gameServerID string,
	launcherSvc *launcher.Service,
) (*InstanceBindingView, error) {
	userID = strings.TrimSpace(userID)
	gameServerID = strings.TrimSpace(gameServerID)
	if userID == "" || gameServerID == "" || launcherSvc == nil {
		return nil, ErrValidation
	}
	if err := s.assertBindableGameServer(ctx, userID, gameServerID); err != nil {
		return nil, err
	}

	if view, err := s.existingBindingView(ctx, userID, gameServerID); err == nil {
		prepareID, prepErr := launcherSvc.EnsureInstancePrepared(ctx, launcher.Owner{UserID: userID}, view.InstanceID)
		if prepErr != nil {
			return nil, prepErr
		}
		view.PrepareRequestID = prepareID
		return view, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	if leftover, err := s.managedInstanceForServer(ctx, userID, gameServerID); err == nil {
		view, bindErr := s.bindManagedInstance(ctx, userID, gameServerID, leftover)
		if bindErr != nil {
			return nil, bindErr
		}
		prepareID, prepErr := launcherSvc.EnsureInstancePrepared(ctx, launcher.Owner{UserID: userID}, leftover.ID)
		if prepErr != nil {
			return nil, prepErr
		}
		view.PrepareRequestID = prepareID
		return view, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	var gs models.GameServer
	if err := s.db.WithContext(ctx).Where("id = ?", gameServerID).First(&gs).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	name, mcVersion, loader, loaderVersion, ok := connectInstanceSpec(&gs)
	if !ok {
		return nil, ErrValidation
	}

	created, err := launcherSvc.CreateInstance(ctx, launcher.Owner{UserID: userID}, launcher.CreateInstanceInput{
		Name:                  name,
		MCVersion:             mcVersion,
		Loader:                loader,
		LoaderVersion:         loaderVersion,
		ManagedByGameServerID: gameServerID,
	})
	if err != nil {
		return nil, err
	}

	view, err := s.bindManagedInstance(ctx, userID, gameServerID, created.Instance)
	if err != nil {
		return nil, err
	}
	view.PrepareRequestID = created.PrepareRequestID
	return view, nil
}

func (s *Service) existingBindingView(ctx context.Context, userID, gameServerID string) (*InstanceBindingView, error) {
	var row bindingRow
	err := s.db.WithContext(ctx).
		Table("game_server_instance_bindings").
		Select(`game_server_instance_bindings.*,
			launcher_instances.name AS instance_name,
			launcher_instances.mc_version AS instance_mc_version,
			launcher_instances.loader AS instance_loader`).
		Joins("JOIN launcher_instances ON launcher_instances.id = game_server_instance_bindings.launcher_instance_id").
		Where("game_server_instance_bindings.user_id = ? AND game_server_instance_bindings.game_server_id = ?", userID, gameServerID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	view := bindingViewFromRow(&row)
	return &view, nil
}

func (s *Service) managedInstanceForServer(ctx context.Context, userID, gameServerID string) (*models.LauncherInstance, error) {
	var inst models.LauncherInstance
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND managed_by_game_server_id = ?", userID, gameServerID).
		Order("created_at desc").
		First(&inst).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &inst, nil
}

func (s *Service) bindManagedInstance(ctx context.Context, userID, gameServerID string, inst *models.LauncherInstance) (*InstanceBindingView, error) {
	if inst.ManagedByGameServerID == nil || strings.TrimSpace(*inst.ManagedByGameServerID) != gameServerID {
		id := gameServerID
		inst.ManagedByGameServerID = &id
		_ = s.db.WithContext(ctx).
			Model(&models.LauncherInstance{}).
			Where("id = ?", inst.ID).
			Update("managed_by_game_server_id", gameServerID).Error
	}

	existing := models.GameServerInstanceBinding{
		ID:                 uuid.NewString(),
		UserID:             userID,
		GameServerID:       gameServerID,
		LauncherInstanceID: inst.ID,
	}
	if err := s.db.WithContext(ctx).Create(&existing).Error; err != nil {
		return nil, err
	}
	view := bindingViewFromInstance(&existing, inst)
	return &view, nil
}

func connectInstanceSpec(gs *models.GameServer) (name, mcVersion, loader, loaderVersion string, ok bool) {
	mcVersion = strings.TrimSpace(gs.MCVersion)
	if mcVersion == "" {
		return "", "", "", "", false
	}
	name = strings.TrimSpace(gs.Name)
	if name == "" {
		name = "Minecraft"
	}
	if len(name) > 64 {
		name = name[:64]
	}
	loader = strings.TrimSpace(gs.ServerType)
	switch loader {
	case models.LoaderVanilla, models.LoaderForge, models.LoaderNeoForge, models.LoaderFabric, models.LoaderQuilt:
	default:
		loader = models.LoaderVanilla
	}
	if gs.LoaderVersion != nil {
		loaderVersion = strings.TrimSpace(*gs.LoaderVersion)
	}
	if loader != models.LoaderVanilla && loaderVersion == "" {
		return "", "", "", "", false
	}
	return name, mcVersion, loader, loaderVersion, true
}
