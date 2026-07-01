package launcher

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"gorm.io/gorm"
)

const modInstallRequestTTL = 10 * time.Minute

type CreateModInstallRequestInput struct {
	InstanceID    string
	DeviceID      string
	Source        string
	ProjectID     string
	ProjectName   string
	VersionID     string
	VersionNumber string
	Filename      string
	DownloadURL   string
	ResourceType  string
}

type ModInstallRequestView struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	InstanceID    string `json:"instance_id"`
	Source        string `json:"source"`
	ProjectID     string `json:"project_id"`
	ProjectName   string `json:"project_name"`
	VersionID     string `json:"version_id"`
	VersionNumber string `json:"version_number,omitempty"`
	Filename      string `json:"filename"`
	DownloadURL   string `json:"download_url,omitempty"`
	ResourceType  string `json:"resource_type"`
	ErrorCode     *string `json:"error_code,omitempty"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type UpdateModInstallRequestInput struct {
	Status    string
	ErrorCode *string
}

func modInstallViewFromModel(req models.ModInstallRequest, includeURL bool) *ModInstallRequestView {
	view := &ModInstallRequestView{
		ID:            req.ID,
		Status:        req.Status,
		InstanceID:    req.InstanceID,
		Source:        req.Source,
		ProjectID:     req.ProjectID,
		ProjectName:   req.ProjectName,
		VersionID:     req.VersionID,
		VersionNumber: req.VersionNumber,
		Filename:      req.Filename,
		ResourceType:  req.ResourceType,
		ErrorCode:     req.ErrorCode,
		ExpiresAt:     req.ExpiresAt,
	}
	if includeURL {
		view.DownloadURL = req.DownloadURL
	}
	return view
}

func normalizeResourceType(resourceType string) string {
	switch strings.TrimSpace(resourceType) {
	case "modpack", "resourcepack", "shader":
		return strings.TrimSpace(resourceType)
	default:
		return "mod"
	}
}

func (s *Service) CreateModInstallRequest(ctx context.Context, owner Owner, in CreateModInstallRequestInput) (*ModInstallRequestView, error) {
	if in.InstanceID == "" || in.DeviceID == "" || in.Source == "" || in.ProjectID == "" ||
		in.ProjectName == "" || in.VersionID == "" || in.Filename == "" || in.DownloadURL == "" {
		return nil, ErrValidation
	}
	if err := s.ValidateDeviceForOwner(ctx, owner, in.DeviceID); err != nil {
		return nil, err
	}
	if _, err := s.GetInstance(ctx, owner, in.InstanceID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	req := models.ModInstallRequest{
		ID:            uuid.NewString(),
		DeviceID:      in.DeviceID,
		InstanceID:    in.InstanceID,
		Source:        in.Source,
		ProjectID:     in.ProjectID,
		ProjectName:   in.ProjectName,
		VersionID:     in.VersionID,
		VersionNumber: in.VersionNumber,
		Filename:      in.Filename,
		DownloadURL:   in.DownloadURL,
		ResourceType:  normalizeResourceType(in.ResourceType),
		Status:        models.ModInstallStatusQueued,
		ExpiresAt:     now.Add(modInstallRequestTTL),
		CreatedAt:     now,
	}
	if err := s.db.WithContext(ctx).Create(&req).Error; err != nil {
		return nil, err
	}
	return modInstallViewFromModel(req, false), nil
}

func (s *Service) GetModInstallRequest(ctx context.Context, owner Owner, requestID string) (*ModInstallRequestView, error) {
	req, err := s.getModInstallRequestForOwner(ctx, owner, requestID)
	if err != nil {
		return nil, err
	}
	return modInstallViewFromModel(*req, false), nil
}

func (s *Service) FetchPendingModInstall(ctx context.Context, deviceID string) (*ModInstallRequestView, error) {
	if deviceID == "" {
		return nil, ErrValidation
	}
	now := time.Now().UTC()
	s.expireStaleModInstalls(ctx, deviceID, now)

	var reqs []models.ModInstallRequest
	err := s.db.WithContext(ctx).
		Where("device_id = ? AND status = ?", deviceID, models.ModInstallStatusQueued).
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
	if err := s.db.WithContext(ctx).Model(&req).Updates(map[string]any{
		"status":        models.ModInstallStatusDispatched,
		"dispatched_at": dispatched,
	}).Error; err != nil {
		return nil, err
	}
	req.Status = models.ModInstallStatusDispatched
	req.DispatchedAt = &dispatched
	return modInstallViewFromModel(req, true), nil
}

func (s *Service) UpdateModInstallRequest(ctx context.Context, deviceID, requestID string, in UpdateModInstallRequestInput) (*ModInstallRequestView, error) {
	if deviceID == "" || requestID == "" || in.Status == "" {
		return nil, ErrValidation
	}
	var req models.ModInstallRequest
	if err := s.db.WithContext(ctx).Where("id = ? AND device_id = ?", requestID, deviceID).First(&req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	updates := map[string]any{"status": in.Status}
	if in.ErrorCode != nil {
		updates["error_code"] = in.ErrorCode
	}
	now := time.Now().UTC()
	switch in.Status {
	case models.ModInstallStatusDownloading:
		// keep dispatched_at
	case models.ModInstallStatusCompleted, models.ModInstallStatusFailed:
		updates["completed_at"] = now
	}

	if err := s.db.WithContext(ctx).Model(&req).Updates(updates).Error; err != nil {
		return nil, err
	}
	req.Status = in.Status
	if in.ErrorCode != nil {
		req.ErrorCode = in.ErrorCode
	}
	if in.Status == models.ModInstallStatusCompleted {
		if err := s.recordInstalledResource(ctx, req); err != nil {
			return nil, err
		}
	}
	return modInstallViewFromModel(req, false), nil
}

func (s *Service) recordInstalledResource(ctx context.Context, req models.ModInstallRequest) error {
	var inst models.LauncherInstance
	if err := s.db.WithContext(ctx).Where("id = ?", req.InstanceID).First(&inst).Error; err != nil {
		return err
	}
	entry := models.InstanceResourceEntry{
		Source:        req.Source,
		ProjectID:     req.ProjectID,
		ProjectName:   req.ProjectName,
		VersionID:     req.VersionID,
		VersionNumber: req.VersionNumber,
		Filename:      req.Filename,
		ResourceType:  req.ResourceType,
		InstalledAt:   resourceInstalledAt(),
	}
	appendInstanceResource(&inst, entry)
	return s.db.WithContext(ctx).Model(&inst).Updates(map[string]any{
		"mods":            inst.Mods,
		"resource_packs":  inst.ResourcePacks,
		"shaders":         inst.Shaders,
	}).Error
}

func (s *Service) getModInstallRequestForOwner(ctx context.Context, owner Owner, requestID string) (*models.ModInstallRequest, error) {
	var req models.ModInstallRequest
	if err := s.db.WithContext(ctx).Where("id = ?", requestID).First(&req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	inst, err := s.GetInstance(ctx, owner, req.InstanceID)
	if err != nil {
		return nil, err
	}
	_ = inst
	return &req, nil
}

func (s *Service) expireStaleModInstalls(ctx context.Context, deviceID string, now time.Time) {
	_ = s.db.WithContext(ctx).Model(&models.ModInstallRequest{}).
		Where("device_id = ? AND status = ? AND expires_at < ?", deviceID, models.ModInstallStatusQueued, now).
		Update("status", models.ModInstallStatusExpired).Error
}

func ResourceInstallFolder(resourceType string) string {
	switch normalizeResourceType(resourceType) {
	case "resourcepack":
		return "resourcepacks"
	case "shader":
		return "shaderpacks"
	default:
		return "mods"
	}
}
