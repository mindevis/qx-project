package launcher

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/qxproject/qx/pkg/mcmanifest"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"gorm.io/gorm"
)

const launchRequestTTL = 5 * time.Minute

type CreateLaunchRequestInput struct {
	InstanceID       string
	OfflineProfileID string
	DeviceID         string
}

type LaunchRequestView struct {
	ID               string                           `json:"id"`
	Status           string                           `json:"status"`
	InstanceID       string                           `json:"instance_id"`
	OfflineProfileID *string                          `json:"offline_profile_id,omitempty"`
	ExpiresAt        time.Time                        `json:"expires_at"`
	Manifest         *mcmanifest.InstanceLaunchManifest `json:"manifest,omitempty"`
	Profile          *models.OfflineProfile           `json:"profile,omitempty"`
	PID              *int                             `json:"pid,omitempty"`
	ExitCode         *int                             `json:"exit_code,omitempty"`
	ErrorCode        *string                          `json:"error_code,omitempty"`
}

type UpdateLaunchRequestInput struct {
	Status    string
	PID       *int
	ExitCode  *int
	ErrorCode *string
}

func (s *Service) CreateLaunchRequest(ctx context.Context, owner Owner, in CreateLaunchRequestInput) (*LaunchRequestView, error) {
	if in.InstanceID == "" || in.DeviceID == "" {
		return nil, ErrValidation
	}
	if err := s.ValidateDeviceForOwner(ctx, owner, in.DeviceID); err != nil {
		return nil, err
	}
	inst, err := s.GetInstance(ctx, owner, in.InstanceID)
	if err != nil {
		return nil, err
	}
	if owner.IsGuest && inst.Loader != models.LoaderVanilla {
		return nil, ErrGuestLoaderOnly
	}

	var profileID *string
	var profile *models.OfflineProfile
	if in.OfflineProfileID != "" {
		profile, err = s.getProfile(ctx, owner, in.OfflineProfileID)
		if err != nil {
			return nil, err
		}
		profileID = &profile.ID
	}

	now := time.Now().UTC()
	req := models.LaunchRequest{
		ID:               uuid.NewString(),
		DeviceID:         in.DeviceID,
		InstanceID:       inst.ID,
		OfflineProfileID: profileID,
		Status:           models.LaunchStatusQueued,
		ExpiresAt:        now.Add(launchRequestTTL),
		CreatedAt:        now,
	}
	if err := s.db.WithContext(ctx).Create(&req).Error; err != nil {
		return nil, err
	}
	return launchViewFromModel(req, nil, profile), nil
}

func (s *Service) FetchPendingLaunch(ctx context.Context, deviceID string) (*LaunchRequestView, error) {
	deviceID = deviceID
	if deviceID == "" {
		return nil, ErrValidation
	}
	now := time.Now().UTC()
	s.expireStaleRequests(ctx, deviceID, now)

	var req models.LaunchRequest
	err := s.db.WithContext(ctx).
		Where("device_id = ? AND status = ?", deviceID, models.LaunchStatusQueued).
		Order("created_at asc").
		First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	dispatched := now
	if err := s.db.WithContext(ctx).Model(&req).Updates(map[string]any{
		"status":        models.LaunchStatusDispatched,
		"dispatched_at": dispatched,
	}).Error; err != nil {
		return nil, err
	}
	req.Status = models.LaunchStatusDispatched
	req.DispatchedAt = &dispatched

	return s.enrichLaunchView(ctx, req)
}

func (s *Service) UpdateLaunchRequest(ctx context.Context, deviceID, requestID string, in UpdateLaunchRequestInput) (*LaunchRequestView, error) {
	var req models.LaunchRequest
	if err := s.db.WithContext(ctx).Where("id = ? AND device_id = ?", requestID, deviceID).First(&req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	updates := map[string]any{"status": in.Status}
	if in.PID != nil {
		updates["pid"] = *in.PID
	}
	if in.ExitCode != nil {
		updates["exit_code"] = *in.ExitCode
	}
	if in.ErrorCode != nil {
		updates["error_code"] = *in.ErrorCode
	}
	if in.Status == models.LaunchStatusCompleted || in.Status == models.LaunchStatusFailed {
		now := time.Now().UTC()
		updates["completed_at"] = now
	}
	if err := s.db.WithContext(ctx).Model(&req).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Where("id = ?", requestID).First(&req).Error; err != nil {
		return nil, err
	}
	return s.enrichLaunchView(ctx, req)
}

func (s *Service) GetLaunchRequest(ctx context.Context, owner Owner, requestID string) (*LaunchRequestView, error) {
	var req models.LaunchRequest
	if err := s.db.WithContext(ctx).Where("id = ?", requestID).First(&req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if _, err := s.GetInstance(ctx, owner, req.InstanceID); err != nil {
		return nil, err
	}
	return s.enrichLaunchView(ctx, req)
}

func (s *Service) expireStaleRequests(ctx context.Context, deviceID string, now time.Time) {
	_ = s.db.WithContext(ctx).Model(&models.LaunchRequest{}).
		Where("device_id = ? AND status = ? AND expires_at < ?", deviceID, models.LaunchStatusQueued, now).
		Update("status", models.LaunchStatusExpired).Error
}

func (s *Service) enrichLaunchView(ctx context.Context, req models.LaunchRequest) (*LaunchRequestView, error) {
	var inst models.LauncherInstance
	if err := s.db.WithContext(ctx).Where("id = ?", req.InstanceID).First(&inst).Error; err != nil {
		return nil, err
	}
	manifest, err := s.manifestProvider().BuildInstanceManifest(ctx, inst.ID, inst.Name, inst.MCVersion, inst.Loader)
	if err != nil {
		return nil, err
	}
	var profile *models.OfflineProfile
	if req.OfflineProfileID != nil {
		var p models.OfflineProfile
		if err := s.db.WithContext(ctx).Where("id = ?", *req.OfflineProfileID).First(&p).Error; err == nil {
			profile = &p
		}
	}
	return launchViewFromModel(req, manifest, profile), nil
}

func launchViewFromModel(req models.LaunchRequest, manifest *mcmanifest.InstanceLaunchManifest, profile *models.OfflineProfile) *LaunchRequestView {
	return &LaunchRequestView{
		ID:               req.ID,
		Status:           req.Status,
		InstanceID:       req.InstanceID,
		OfflineProfileID: req.OfflineProfileID,
		ExpiresAt:        req.ExpiresAt,
		Manifest:         manifest,
		Profile:          profile,
		PID:              req.PID,
		ExitCode:         req.ExitCode,
		ErrorCode:        req.ErrorCode,
	}
}
