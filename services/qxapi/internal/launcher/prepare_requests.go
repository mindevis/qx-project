package launcher

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"gorm.io/gorm"
)

const prepareRequestTTL = 30 * time.Minute

type PrepareRequestView struct {
	ID         string              `json:"id"`
	Status     string              `json:"status"`
	InstanceID string              `json:"instance_id"`
	Instance   *LaunchInstanceView `json:"instance,omitempty"`
	ErrorCode  *string             `json:"error_code,omitempty"`
	ExpiresAt  time.Time           `json:"expires_at"`
}

type UpdatePrepareRequestInput struct {
	Status    string
	ErrorCode *string
}

func prepareViewFromModel(req models.PrepareRequest, inst *LaunchInstanceView, includeInstance bool) *PrepareRequestView {
	view := &PrepareRequestView{
		ID:         req.ID,
		Status:     req.Status,
		InstanceID: req.InstanceID,
		ErrorCode:  req.ErrorCode,
		ExpiresAt:  req.ExpiresAt,
	}
	if includeInstance && inst != nil {
		view.Instance = inst
	}
	return view
}

func (s *Service) CreatePrepareRequest(ctx context.Context, deviceID, instanceID string) (*PrepareRequestView, error) {
	if deviceID == "" || instanceID == "" {
		return nil, ErrValidation
	}
	var inst models.LauncherInstance
	if err := s.db.WithContext(ctx).Where("id = ?", instanceID).First(&inst).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	now := time.Now().UTC()
	req := models.PrepareRequest{
		ID:         uuid.NewString(),
		DeviceID:   deviceID,
		InstanceID: instanceID,
		Status:     models.PrepareStatusQueued,
		ExpiresAt:  now.Add(prepareRequestTTL),
		CreatedAt:  now,
	}
	if err := s.db.WithContext(ctx).Create(&req).Error; err != nil {
		return nil, err
	}
	return prepareViewFromModel(req, launchInstanceFromModel(inst), false), nil
}

func (s *Service) GetPrepareRequest(ctx context.Context, owner Owner, requestID string) (*PrepareRequestView, error) {
	req, err := s.getPrepareRequestForOwner(ctx, owner, requestID)
	if err != nil {
		return nil, err
	}
	return prepareViewFromModel(*req, nil, false), nil
}

func (s *Service) FetchPendingPrepare(ctx context.Context, deviceID string) (*PrepareRequestView, error) {
	if deviceID == "" {
		return nil, ErrValidation
	}
	now := time.Now().UTC()
	s.expireStalePrepareRequests(ctx, deviceID, now)

	var reqs []models.PrepareRequest
	err := s.db.WithContext(ctx).
		Where("device_id = ? AND status = ?", deviceID, models.PrepareStatusQueued).
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
		"status":        models.PrepareStatusPreparing,
		"dispatched_at": dispatched,
	}).Error; err != nil {
		return nil, err
	}
	req.Status = models.PrepareStatusPreparing
	req.DispatchedAt = &dispatched

	var inst models.LauncherInstance
	if err := s.db.WithContext(ctx).Where("id = ?", req.InstanceID).First(&inst).Error; err != nil {
		return nil, err
	}
	return prepareViewFromModel(req, launchInstanceFromModel(inst), true), nil
}

func (s *Service) UpdatePrepareRequest(ctx context.Context, deviceID, requestID string, in UpdatePrepareRequestInput) (*PrepareRequestView, error) {
	if deviceID == "" || requestID == "" || in.Status == "" {
		return nil, ErrValidation
	}
	var req models.PrepareRequest
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
	if in.Status == models.PrepareStatusCompleted || in.Status == models.PrepareStatusFailed {
		now := time.Now().UTC()
		updates["completed_at"] = now
	}

	if err := s.db.WithContext(ctx).Model(&req).Updates(updates).Error; err != nil {
		return nil, err
	}
	req.Status = in.Status
	if in.ErrorCode != nil {
		req.ErrorCode = in.ErrorCode
	}
	return prepareViewFromModel(req, nil, false), nil
}

func (s *Service) getPrepareRequestForOwner(ctx context.Context, owner Owner, requestID string) (*models.PrepareRequest, error) {
	var req models.PrepareRequest
	if err := s.db.WithContext(ctx).Where("id = ?", requestID).First(&req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if _, err := s.GetInstance(ctx, owner, req.InstanceID); err != nil {
		return nil, err
	}
	return &req, nil
}

func (s *Service) expireStalePrepareRequests(ctx context.Context, deviceID string, now time.Time) {
	_ = s.db.WithContext(ctx).Model(&models.PrepareRequest{}).
		Where("device_id = ? AND status = ? AND expires_at < ?", deviceID, models.PrepareStatusQueued, now).
		Update("status", models.PrepareStatusExpired).Error
}

func (s *Service) enqueuePrepareForInstance(ctx context.Context, owner Owner, instanceID string) (*string, error) {
	deviceID, err := s.FindLinkedDevice(ctx, owner)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	view, err := s.CreatePrepareRequest(ctx, deviceID, instanceID)
	if err != nil {
		return nil, err
	}
	return &view.ID, nil
}
