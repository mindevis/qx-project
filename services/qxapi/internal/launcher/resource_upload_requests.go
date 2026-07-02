package launcher

import (
	"context"
	"encoding/base64"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"gorm.io/gorm"
)

const resourceUploadRequestTTL = 10 * time.Minute
const resourceUploadBridgeTimeout = 60 * time.Second

type InstanceResourceUploadView struct {
	ID           string  `json:"id"`
	Status       string  `json:"status"`
	InstanceID   string  `json:"instance_id"`
	Filename     string  `json:"filename"`
	ResourceType string  `json:"resource_type"`
	FileSize     int64   `json:"file_size"`
	ContentB64   string  `json:"content_b64,omitempty"`
	ErrorCode    *string `json:"error_code,omitempty"`
}

type UpdateResourceUploadRequestInput struct {
	Status    string
	ErrorCode *string
}

func (s *Service) CreateInstanceResourceUpload(ctx context.Context, owner Owner, instanceID, filename, resourceType string, data []byte) (*InstanceResourceUploadView, error) {
	if err := ValidateResourceFilename(filename); err != nil {
		return nil, err
	}
	if err := ValidateResourceUploadSize(int64(len(data))); err != nil {
		return nil, err
	}
	deviceID, err := s.requireLinkedDevice(ctx, owner)
	if err != nil {
		return nil, err
	}
	if _, err := s.GetInstance(ctx, owner, instanceID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	req := models.InstanceResourceUploadRequest{
		ID:           uuid.NewString(),
		DeviceID:     deviceID,
		InstanceID:   instanceID,
		Filename:     filepath.Base(filename),
		ResourceType: normalizeResourceType(resourceType),
		ContentB64:   base64.StdEncoding.EncodeToString(data),
		FileSize:     int64(len(data)),
		Status:       models.ResourceUploadStatusQueued,
		ExpiresAt:    now.Add(resourceUploadRequestTTL),
		CreatedAt:    now,
	}
	if err := s.db.WithContext(ctx).Create(&req).Error; err != nil {
		return nil, err
	}

	ch := make(chan fileRPCResult, 1)
	s.pendingUploadRPC.Store(req.ID, ch)
	defer s.pendingUploadRPC.Delete(req.ID)

	waitCtx, cancel := context.WithTimeout(ctx, resourceUploadBridgeTimeout)
	defer cancel()

	select {
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		return resourceUploadViewFromModel(req, false), nil
	case <-waitCtx.Done():
		_ = s.db.WithContext(context.Background()).Model(&req).Update("status", models.ResourceUploadStatusExpired).Error
		if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
			return nil, ErrBridgeTimeout
		}
		return nil, waitCtx.Err()
	}
}

func (s *Service) FetchPendingResourceUpload(ctx context.Context, deviceID string) (*InstanceResourceUploadView, error) {
	if deviceID == "" {
		return nil, ErrValidation
	}
	now := time.Now().UTC()
	s.expireStaleResourceUploads(ctx, deviceID, now)

	var reqs []models.InstanceResourceUploadRequest
	err := s.db.WithContext(ctx).
		Where("device_id = ? AND status = ?", deviceID, models.ResourceUploadStatusQueued).
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
		"status":        models.ResourceUploadStatusDispatched,
		"dispatched_at": dispatched,
	}).Error; err != nil {
		return nil, err
	}
	req.Status = models.ResourceUploadStatusDispatched
	return resourceUploadViewFromModel(req, true), nil
}

func (s *Service) UpdateResourceUploadRequest(ctx context.Context, deviceID, requestID string, in UpdateResourceUploadRequestInput) (*InstanceResourceUploadView, error) {
	if deviceID == "" || requestID == "" || in.Status == "" {
		return nil, ErrValidation
	}
	var req models.InstanceResourceUploadRequest
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
	if in.Status == models.ResourceUploadStatusCompleted || in.Status == models.ResourceUploadStatusFailed {
		updates["completed_at"] = now
	}
	if err := s.db.WithContext(ctx).Model(&req).Updates(updates).Error; err != nil {
		return nil, err
	}

	var bridgeErr error
	if in.Status == models.ResourceUploadStatusCompleted {
		if err := s.recordUploadedResource(ctx, req); err != nil {
			return nil, err
		}
		bridgeErr = nil
	}
	if in.Status == models.ResourceUploadStatusFailed {
		code := "UPLOAD_FAILED"
		if in.ErrorCode != nil && *in.ErrorCode != "" {
			code = *in.ErrorCode
		}
		bridgeErr = errors.New(code)
	}
	s.deliverUploadRPC(requestID, bridgeErr)

	req.Status = in.Status
	if in.ErrorCode != nil {
		req.ErrorCode = in.ErrorCode
	}
	return resourceUploadViewFromModel(req, false), nil
}

func (s *Service) recordUploadedResource(ctx context.Context, req models.InstanceResourceUploadRequest) error {
	var inst models.LauncherInstance
	if err := s.db.WithContext(ctx).Where("id = ?", req.InstanceID).First(&inst).Error; err != nil {
		return err
	}
	name := strings.TrimSuffix(req.Filename, filepath.Ext(req.Filename))
	entry := models.InstanceResourceEntry{
		Source:       "upload",
		ProjectName:  name,
		Filename:     req.Filename,
		ResourceType: req.ResourceType,
		FileSize:     req.FileSize,
		InstalledAt:  resourceInstalledAt(),
	}
	appendInstanceResource(&inst, entry)
	return s.db.WithContext(ctx).Model(&inst).Updates(map[string]any{
		"mods":           inst.Mods,
		"resource_packs": inst.ResourcePacks,
		"shaders":        inst.Shaders,
		"datapacks":      inst.Datapacks,
	}).Error
}

func (s *Service) deliverUploadRPC(requestID string, err error) {
	if requestID == "" {
		return
	}
	raw, ok := s.pendingUploadRPC.Load(requestID)
	if !ok {
		return
	}
	ch := raw.(chan fileRPCResult)
	select {
	case ch <- fileRPCResult{err: err}:
	default:
	}
}

func resourceUploadViewFromModel(req models.InstanceResourceUploadRequest, includeContent bool) *InstanceResourceUploadView {
	view := &InstanceResourceUploadView{
		ID:           req.ID,
		Status:       req.Status,
		InstanceID:   req.InstanceID,
		Filename:     req.Filename,
		ResourceType: req.ResourceType,
		FileSize:     req.FileSize,
		ErrorCode:    req.ErrorCode,
	}
	if includeContent {
		view.ContentB64 = req.ContentB64
	}
	return view
}

func (s *Service) expireStaleResourceUploads(ctx context.Context, deviceID string, now time.Time) {
	_ = s.db.WithContext(ctx).Model(&models.InstanceResourceUploadRequest{}).
		Where("device_id = ? AND status = ? AND expires_at < ?", deviceID, models.ResourceUploadStatusQueued, now).
		Update("status", models.ResourceUploadStatusExpired).Error
}
