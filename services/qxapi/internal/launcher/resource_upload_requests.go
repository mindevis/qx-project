package launcher

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"gorm.io/gorm"
)

const resourceUploadRequestTTL = 10 * time.Minute
const resourceUploadBridgeTimeout = 5 * time.Minute

type InstanceResourceUploadView struct {
	ID           string  `json:"id"`
	Status       string  `json:"status"`
	InstanceID   string  `json:"instance_id"`
	Filename     string  `json:"filename"`
	ResourceType string  `json:"resource_type"`
	FileSize     int64   `json:"file_size"`
	ErrorCode    *string `json:"error_code,omitempty"`
}

type UpdateResourceUploadRequestInput struct {
	Status    string
	ErrorCode *string
}

func resourceUploadObjectKey(id string) string {
	return "connect-mods/" + id
}

func (s *Service) CreateInstanceResourceUpload(ctx context.Context, owner Owner, instanceID, filename, resourceType string, data []byte) (*InstanceResourceUploadView, error) {
	return s.createInstanceResourceUpload(ctx, owner, instanceID, filename, resourceType, data, false)
}

func (s *Service) CreateInstanceResourceUploadForSync(ctx context.Context, owner Owner, instanceID, filename, resourceType string, data []byte) (*InstanceResourceUploadView, error) {
	return s.createInstanceResourceUpload(ctx, owner, instanceID, filename, resourceType, data, true)
}

func (s *Service) createInstanceResourceUpload(ctx context.Context, owner Owner, instanceID, filename, resourceType string, data []byte, allowManaged bool) (*InstanceResourceUploadView, error) {
	if err := ValidateResourceFilename(filename); err != nil {
		return nil, err
	}
	if err := ValidateResourceUploadSize(int64(len(data))); err != nil {
		return nil, err
	}
	if s.blobs == nil {
		return nil, ErrValidation
	}
	deviceID, err := s.requireLinkedDevice(ctx, owner)
	if err != nil {
		return nil, err
	}
	if _, err := s.GetInstance(ctx, owner, instanceID); err != nil {
		return nil, err
	}
	if !allowManaged {
		if err := s.AssertInstanceContentMutable(ctx, instanceID); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	req := models.InstanceResourceUploadRequest{
		ID:           uuid.NewString(),
		DeviceID:     deviceID,
		InstanceID:   instanceID,
		Filename:     filepath.Base(filename),
		ResourceType: normalizeResourceType(resourceType),
		FileSize:     int64(len(data)),
		Status:       models.ResourceUploadStatusQueued,
		ExpiresAt:    now.Add(resourceUploadRequestTTL),
		CreatedAt:    now,
	}
	req.ObjectKey = resourceUploadObjectKey(req.ID)
	if err := s.blobs.Put(ctx, req.ObjectKey, data); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Create(&req).Error; err != nil {
		_ = s.blobs.Delete(context.Background(), req.ObjectKey)
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
			s.deleteUploadBlob(req)
			return nil, res.err
		}
		s.deleteUploadBlob(req)
		return resourceUploadViewFromModel(req), nil
	case <-waitCtx.Done():
		s.deleteUploadBlob(req)
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
	deviceIDs, err := s.deliveryDeviceIDs(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	s.expireStaleResourceUploads(ctx, deviceIDs, now)

	var reqs []models.InstanceResourceUploadRequest
	err = s.db.WithContext(ctx).
		Where("device_id IN ? AND status = ?", deviceIDs, models.ResourceUploadStatusQueued).
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
		"device_id":     deviceID,
	}).Error; err != nil {
		return nil, err
	}
	req.Status = models.ResourceUploadStatusDispatched
	req.DeviceID = deviceID
	return resourceUploadViewFromModel(req), nil
}

func (s *Service) OpenResourceUpload(ctx context.Context, deviceID, requestID string) (io.ReadCloser, string, int64, error) {
	if deviceID == "" || requestID == "" || s.blobs == nil {
		return nil, "", 0, ErrValidation
	}
	var req models.InstanceResourceUploadRequest
	if err := s.db.WithContext(ctx).Where("id = ?", requestID).First(&req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", 0, ErrNotFound
		}
		return nil, "", 0, err
	}
	ids, err := s.deliveryDeviceIDs(ctx, deviceID)
	if err != nil {
		return nil, "", 0, err
	}
	allowed := false
	for _, id := range ids {
		if id == req.DeviceID {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, "", 0, ErrNotFound
	}
	if req.ObjectKey == "" {
		return nil, "", 0, ErrNotFound
	}
	rc, size, err := s.blobs.Open(ctx, req.ObjectKey)
	if err != nil {
		return nil, "", 0, err
	}
	return rc, req.Filename, size, nil
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
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if in.Status == models.ResourceUploadStatusCompleted {
			if err := recordUploadedResource(tx, req); err != nil {
				return err
			}
		}
		return tx.Model(&req).Updates(updates).Error
	}); err != nil {
		return nil, err
	}

	var bridgeErr error
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
	return resourceUploadViewFromModel(req), nil
}

func recordUploadedResource(tx *gorm.DB, req models.InstanceResourceUploadRequest) error {
	var inst models.LauncherInstance
	if err := lockInstanceForUpdate(tx).Where("id = ?", req.InstanceID).First(&inst).Error; err != nil {
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
	return tx.Model(&inst).Updates(map[string]any{
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

func resourceUploadViewFromModel(req models.InstanceResourceUploadRequest) *InstanceResourceUploadView {
	return &InstanceResourceUploadView{
		ID:           req.ID,
		Status:       req.Status,
		InstanceID:   req.InstanceID,
		Filename:     req.Filename,
		ResourceType: req.ResourceType,
		FileSize:     req.FileSize,
		ErrorCode:    req.ErrorCode,
	}
}

func (s *Service) deleteUploadBlob(req models.InstanceResourceUploadRequest) {
	if s.blobs == nil || req.ObjectKey == "" {
		return
	}
	_ = s.blobs.Delete(context.Background(), req.ObjectKey)
}

func (s *Service) expireStaleResourceUploads(ctx context.Context, deviceIDs []string, now time.Time) {
	if len(deviceIDs) == 0 {
		return
	}
	var expired []models.InstanceResourceUploadRequest
	_ = s.db.WithContext(ctx).
		Where("device_id IN ? AND status = ? AND expires_at < ?", deviceIDs, models.ResourceUploadStatusQueued, now).
		Find(&expired).Error
	for _, req := range expired {
		s.deleteUploadBlob(req)
	}
	_ = s.db.WithContext(ctx).Model(&models.InstanceResourceUploadRequest{}).
		Where("device_id IN ? AND status = ? AND expires_at < ?", deviceIDs, models.ResourceUploadStatusQueued, now).
		Update("status", models.ResourceUploadStatusExpired).Error
}
