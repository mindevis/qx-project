package launcher

import (
	"context"
	"strings"
	"time"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

type UpdateRequestView struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	DownloadURL string `json:"download_url"`
	Filename    string `json:"filename"`
}

type UpdateRequestResult struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type CompleteUpdateInput struct {
	Status          string
	LauncherVersion string
	ErrorCode       string
}

func (s *Service) RequestLauncherUpdate(ctx context.Context, userID string) (*UpdateRequestResult, error) {
	deviceID, err := s.FindLinkedDevice(ctx, Owner{UserID: userID})
	if err != nil {
		return nil, err
	}
	device, err := s.getDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if device.Status != models.DeviceStatusLinked {
		return nil, ErrDeviceNotLinked
	}
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(device).Update("update_requested_at", now).Error; err != nil {
		return nil, err
	}
	return &UpdateRequestResult{ID: deviceID, Status: "queued"}, nil
}

func (s *Service) FetchPendingUpdate(ctx context.Context, deviceID string) (*UpdateRequestView, error) {
	if deviceID == "" {
		return nil, ErrValidation
	}
	device, err := s.getDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if device.UpdateRequestedAt == nil {
		return nil, nil
	}
	release := s.releaseInfo()
	return &UpdateRequestView{
		ID:          device.DeviceID,
		Version:     release.Version,
		DownloadURL: release.DownloadURL,
		Filename:    release.Filename,
	}, nil
}

func (s *Service) CompleteLauncherUpdate(ctx context.Context, deviceID string, in CompleteUpdateInput) error {
	if deviceID == "" || in.Status == "" {
		return ErrValidation
	}
	device, err := s.getDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	if device.UpdateRequestedAt == nil {
		return ErrNotFound
	}
	updates := map[string]any{"update_requested_at": nil}
	if in.Status == "completed" {
		if ver := strings.TrimSpace(in.LauncherVersion); ver != "" {
			updates["launcher_version"] = ver
		}
	}
	return s.db.WithContext(ctx).Model(device).Updates(updates).Error
}
