package launcher

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/qxproject/qx/services/qxapi/internal/cosmetics"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"github.com/qxproject/qx/services/qxapi/internal/mojang"
	"gorm.io/gorm"
)

const launchRequestTTL = 5 * time.Minute

type CreateLaunchRequestInput struct {
	InstanceID        string
	OfflineProfileID  string
	DeviceID          string
	UseMojangAccount  bool
	JoinServerAddress string
	JoinServerPort    int
}

type MojangSessionView struct {
	Username    string `json:"username"`
	UUID        string `json:"uuid"`
	AccessToken string `json:"access_token"`
}

type LaunchInstanceView struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	MCVersion     string   `json:"mc_version"`
	Loader        string   `json:"loader"`
	LoaderVersion *string  `json:"loader_version,omitempty"`
	MaxMemoryMB   *int     `json:"max_memory_mb,omitempty"`
	MinMemoryMB   *int     `json:"min_memory_mb,omitempty"`
	ExtraJVMArgs  []string `json:"extra_jvm_args,omitempty"`
	WindowWidth   *int     `json:"window_width,omitempty"`
	WindowHeight  *int     `json:"window_height,omitempty"`
}

type LaunchRequestView struct {
	ID                string                 `json:"id"`
	Status            string                 `json:"status"`
	InstanceID        string                 `json:"instance_id"`
	OfflineProfileID  *string                `json:"offline_profile_id,omitempty"`
	UseMojangAccount  bool                   `json:"use_mojang_account,omitempty"`
	JoinServerAddress *string                `json:"join_server_address,omitempty"`
	JoinServerPort    *int                   `json:"join_server_port,omitempty"`
	ExpiresAt         time.Time              `json:"expires_at"`
	Instance          *LaunchInstanceView    `json:"instance,omitempty"`
	Profile           *models.OfflineProfile `json:"profile,omitempty"`
	MojangSession     *MojangSessionView     `json:"mojang_session,omitempty"`
	Cosmetics         *cosmetics.LaunchView  `json:"cosmetics,omitempty"`
	PID               *int                   `json:"pid,omitempty"`
	ExitCode          *int                   `json:"exit_code,omitempty"`
	ErrorCode         *string                `json:"error_code,omitempty"`
	ProgressMessage   string                 `json:"progress_message,omitempty"`
}

type UpdateLaunchRequestInput struct {
	Status          string
	PID             *int
	ExitCode        *int
	ErrorCode       *string
	ProgressMessage string
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

	if in.OfflineProfileID != "" && in.UseMojangAccount {
		return nil, ErrValidation
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

	if in.UseMojangAccount {
		if owner.UserID == "" || s.mojang == nil {
			return nil, ErrValidation
		}
		status, err := s.mojang.GetStatus(ctx, owner.UserID)
		if err != nil {
			return nil, err
		}
		if !status.Linked {
			return nil, ErrValidation
		}
	}

	now := time.Now().UTC()
	req := models.LaunchRequest{
		ID:               uuid.NewString(),
		DeviceID:         in.DeviceID,
		InstanceID:       inst.ID,
		OfflineProfileID: profileID,
		UseMojangAccount: in.UseMojangAccount,
		Status:           models.LaunchStatusQueued,
		ExpiresAt:        now.Add(launchRequestTTL),
		CreatedAt:        now,
	}
	if addr := strings.TrimSpace(in.JoinServerAddress); addr != "" {
		req.JoinServerAddress = &addr
		port := in.JoinServerPort
		if port <= 0 {
			port = 25565
		}
		req.JoinServerPort = &port
	}
	if err := s.db.WithContext(ctx).Create(&req).Error; err != nil {
		return nil, err
	}
	return launchViewFromModel(req, nil, profile, nil, nil), nil
}

func (s *Service) FetchPendingLaunch(ctx context.Context, deviceID string) (*LaunchRequestView, error) {
	if deviceID == "" {
		return nil, ErrValidation
	}
	now := time.Now().UTC()
	deviceIDs, err := s.deliveryDeviceIDs(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	s.expireStaleRequests(ctx, deviceIDs, now)

	var reqs []models.LaunchRequest
	err = s.db.WithContext(ctx).
		Where("device_id IN ? AND status = ?", deviceIDs, models.LaunchStatusQueued).
		Order("created_at asc").
		Limit(1).
		Find(&reqs).Error
	if err != nil {
		return nil, err
	}
	if len(reqs) == 0 {
		return nil, nil
	}
	req := reqs[0]

	dispatched := now
	// Claim for the polling device so any online launcher of the owner handles it.
	if err := s.db.WithContext(ctx).Model(&req).Updates(map[string]any{
		"status":        models.LaunchStatusDispatched,
		"dispatched_at": dispatched,
		"device_id":     deviceID,
	}).Error; err != nil {
		return nil, err
	}
	req.Status = models.LaunchStatusDispatched
	req.DeviceID = deviceID
	req.DispatchedAt = &dispatched

	view, err := s.enrichLaunchView(ctx, req)
	if err != nil {
		if errors.Is(err, ErrMojangUnavailable) {
			_ = s.db.WithContext(ctx).Model(&req).Updates(map[string]any{
				"status":        models.LaunchStatusQueued,
				"dispatched_at": nil,
			}).Error
			return nil, err
		}
		if failed, failErr := s.failLaunchRequestOnEnrichError(ctx, req.ID, err); failErr == nil && failed != nil {
			return nil, nil
		}
		return nil, err
	}
	return view, nil
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
	if msg := strings.TrimSpace(in.ProgressMessage); msg != "" {
		if len(msg) > 256 {
			msg = msg[:256]
		}
		updates["progress_message"] = msg
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
	return launchViewFromModel(req, nil, nil, nil, nil), nil
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
	// Web status polling must not enrich — manifest/Mojang enrichment runs when the launcher fetches a pending job.
	return launchViewFromModel(req, nil, nil, nil, nil), nil
}

func (s *Service) failLaunchRequestOnEnrichError(ctx context.Context, requestID string, err error) (*models.LaunchRequest, error) {
	errorCode := ""
	switch {
	case errors.Is(err, ErrMojangSession), errors.Is(err, ErrValidation):
		errorCode = "MOJANG_SESSION"
	default:
		return nil, err
	}
	now := time.Now().UTC()
	updates := map[string]any{
		"status":       models.LaunchStatusFailed,
		"error_code":   errorCode,
		"completed_at": now,
	}
	if err := s.db.WithContext(ctx).Model(&models.LaunchRequest{}).Where("id = ?", requestID).Updates(updates).Error; err != nil {
		return nil, err
	}
	var req models.LaunchRequest
	if err := s.db.WithContext(ctx).Where("id = ?", requestID).First(&req).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (s *Service) expireStaleRequests(ctx context.Context, deviceIDs []string, now time.Time) {
	if len(deviceIDs) == 0 {
		return
	}
	_ = s.db.WithContext(ctx).Model(&models.LaunchRequest{}).
		Where("device_id IN ? AND status = ? AND expires_at < ?", deviceIDs, models.LaunchStatusQueued, now).
		Update("status", models.LaunchStatusExpired).Error
}

func launchNeedsMojangSession(status string) bool {
	return status == models.LaunchStatusQueued || status == models.LaunchStatusDispatched
}

func launchInstanceFromModel(inst models.LauncherInstance) *LaunchInstanceView {
	return &LaunchInstanceView{
		ID:            inst.ID,
		Name:          inst.Name,
		MCVersion:     inst.MCVersion,
		Loader:        inst.Loader,
		LoaderVersion: inst.LoaderVersion,
		MaxMemoryMB:   inst.MaxMemoryMB,
		MinMemoryMB:   inst.MinMemoryMB,
		ExtraJVMArgs:  []string(inst.ExtraJVMArgs),
		WindowWidth:   inst.WindowWidth,
		WindowHeight:  inst.WindowHeight,
	}
}

func (s *Service) enrichLaunchView(ctx context.Context, req models.LaunchRequest) (*LaunchRequestView, error) {
	var inst models.LauncherInstance
	if err := s.db.WithContext(ctx).Where("id = ?", req.InstanceID).First(&inst).Error; err != nil {
		return nil, err
	}
	instanceView := launchInstanceFromModel(inst)
	var profile *models.OfflineProfile
	if req.OfflineProfileID != nil {
		var p models.OfflineProfile
		if err := s.db.WithContext(ctx).Where("id = ?", *req.OfflineProfileID).First(&p).Error; err == nil {
			profile = &p
		}
	}
	var mojangSession *MojangSessionView
	if req.UseMojangAccount && s.mojang != nil && inst.UserID != nil && launchNeedsMojangSession(req.Status) {
		session, err := s.mojang.SessionForLaunch(ctx, *inst.UserID)
		if err != nil {
			switch {
			case errors.Is(err, mojang.ErrNotLinked):
				return nil, ErrValidation
			case errors.Is(err, mojang.ErrSessionRevoked):
				return nil, fmt.Errorf("%w: %v", ErrMojangSession, err)
			case errors.Is(err, mojang.ErrNotConfigured), errors.Is(err, mojang.ErrSessionUnavailable):
				return nil, fmt.Errorf("%w: %v", ErrMojangUnavailable, err)
			default:
				return nil, err
			}
		}
		mojangSession = &MojangSessionView{
			Username:    session.Username,
			UUID:        session.UUID,
			AccessToken: session.AccessToken,
		}
	}
	var cosmeticsView *cosmetics.LaunchView
	if s.cosmetics != nil && inst.UserID != nil {
		gameUUID := ""
		if mojangSession != nil {
			gameUUID = mojangSession.UUID
		} else if profile != nil {
			gameUUID = profile.OfflineUUID
		}
		view, err := s.cosmetics.LaunchViewForGame(ctx, *inst.UserID, gameUUID)
		if err == nil {
			cosmeticsView = view
		}
	}
	return launchViewFromModel(req, instanceView, profile, mojangSession, cosmeticsView), nil
}

func launchViewFromModel(req models.LaunchRequest, instance *LaunchInstanceView, profile *models.OfflineProfile, mojangSession *MojangSessionView, cosmeticsView *cosmetics.LaunchView) *LaunchRequestView {
	return &LaunchRequestView{
		ID:                req.ID,
		Status:            req.Status,
		InstanceID:        req.InstanceID,
		OfflineProfileID:  req.OfflineProfileID,
		UseMojangAccount:  req.UseMojangAccount,
		JoinServerAddress: req.JoinServerAddress,
		JoinServerPort:    req.JoinServerPort,
		ExpiresAt:         req.ExpiresAt,
		Instance:          instance,
		Profile:           profile,
		MojangSession:     mojangSession,
		Cosmetics:         cosmeticsView,
		PID:               req.PID,
		ExitCode:          req.ExitCode,
		ErrorCode:         req.ErrorCode,
		ProgressMessage:   req.ProgressMessage,
	}
}
