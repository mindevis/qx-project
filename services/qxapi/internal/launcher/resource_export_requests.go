package launcher

import (
	"context"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"gorm.io/gorm"
)

const resourceExportRequestTTL = 10 * time.Minute
const resourceExportBridgeTimeout = 60 * time.Second

type exportRPCResult struct {
	data []byte
	err  error
}

type InstanceResourceExportView struct {
	ID           string  `json:"id"`
	Status       string  `json:"status"`
	InstanceID   string  `json:"instance_id"`
	Filename     string  `json:"filename"`
	ResourceType string  `json:"resource_type"`
	ErrorCode    *string `json:"error_code,omitempty"`
}

type UpdateResourceExportRequestInput struct {
	Status     string
	ContentB64 string
	ErrorCode  *string
}

func (s *Service) ExportInstanceResource(
	ctx context.Context,
	owner Owner,
	instanceID, filename, resourceType string,
) ([]byte, error) {
	if err := ValidateResourceFilename(filename); err != nil {
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
	req := models.InstanceResourceExportRequest{
		ID:           uuid.NewString(),
		DeviceID:     deviceID,
		InstanceID:   instanceID,
		Filename:     filename,
		ResourceType: normalizeResourceType(resourceType),
		Status:       models.ResourceExportStatusQueued,
		ExpiresAt:    now.Add(resourceExportRequestTTL),
		CreatedAt:    now,
	}
	if err := s.db.WithContext(ctx).Create(&req).Error; err != nil {
		return nil, err
	}

	ch := make(chan exportRPCResult, 1)
	s.pendingExportRPC.Store(req.ID, ch)
	defer s.pendingExportRPC.Delete(req.ID)

	waitCtx, cancel := context.WithTimeout(ctx, resourceExportBridgeTimeout)
	defer cancel()

	select {
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		return res.data, nil
	case <-waitCtx.Done():
		_ = s.db.WithContext(context.Background()).Model(&req).Update("status", models.ResourceExportStatusExpired).Error
		if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
			return nil, ErrBridgeTimeout
		}
		return nil, waitCtx.Err()
	}
}

func (s *Service) FetchPendingResourceExport(ctx context.Context, deviceID string) (*InstanceResourceExportView, error) {
	if deviceID == "" {
		return nil, ErrValidation
	}
	now := time.Now().UTC()
	deviceIDs, err := s.deliveryDeviceIDs(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	s.expireStaleResourceExports(ctx, deviceIDs, now)

	var reqs []models.InstanceResourceExportRequest
	err = s.db.WithContext(ctx).
		Where("device_id IN ? AND status = ?", deviceIDs, models.ResourceExportStatusQueued).
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
		"status":        models.ResourceExportStatusDispatched,
		"dispatched_at": dispatched,
		"device_id":     deviceID,
	}).Error; err != nil {
		return nil, err
	}
	req.Status = models.ResourceExportStatusDispatched
	req.DeviceID = deviceID
	return resourceExportViewFromModel(req), nil
}

func (s *Service) UpdateResourceExportRequest(
	ctx context.Context,
	deviceID, requestID string,
	in UpdateResourceExportRequestInput,
) (*InstanceResourceExportView, error) {
	if deviceID == "" || requestID == "" || in.Status == "" {
		return nil, ErrValidation
	}
	var req models.InstanceResourceExportRequest
	if err := s.db.WithContext(ctx).Where("id = ? AND device_id = ?", requestID, deviceID).First(&req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	updates := map[string]any{"status": in.Status}
	if in.ContentB64 != "" {
		updates["content_b64"] = in.ContentB64
	}
	if in.ErrorCode != nil {
		updates["error_code"] = in.ErrorCode
	}
	now := time.Now().UTC()
	if in.Status == models.ResourceExportStatusCompleted || in.Status == models.ResourceExportStatusFailed {
		updates["completed_at"] = now
	}
	if err := s.db.WithContext(ctx).Model(&req).Updates(updates).Error; err != nil {
		return nil, err
	}

	var bridgeRes exportRPCResult
	if in.Status == models.ResourceExportStatusCompleted {
		if in.ContentB64 == "" {
			bridgeRes.err = errors.New("EXPORT_EMPTY")
		} else {
			data, decErr := base64.StdEncoding.DecodeString(in.ContentB64)
			if decErr != nil {
				bridgeRes.err = errors.New("EXPORT_DECODE_FAILED")
			} else if err := ValidateResourceUploadSize(int64(len(data))); err != nil {
				bridgeRes.err = err
			} else {
				bridgeRes.data = data
			}
		}
	}
	if in.Status == models.ResourceExportStatusFailed {
		code := "EXPORT_FAILED"
		if in.ErrorCode != nil && *in.ErrorCode != "" {
			code = *in.ErrorCode
		}
		bridgeRes.err = errors.New(code)
	}
	s.deliverExportRPC(requestID, bridgeRes)

	req.Status = in.Status
	if in.ErrorCode != nil {
		req.ErrorCode = in.ErrorCode
	}
	return resourceExportViewFromModel(req), nil
}

func (s *Service) deliverExportRPC(requestID string, res exportRPCResult) {
	if requestID == "" {
		return
	}
	raw, ok := s.pendingExportRPC.Load(requestID)
	if !ok {
		return
	}
	ch := raw.(chan exportRPCResult)
	select {
	case ch <- res:
	default:
	}
}

func resourceExportViewFromModel(req models.InstanceResourceExportRequest) *InstanceResourceExportView {
	return &InstanceResourceExportView{
		ID:           req.ID,
		Status:       req.Status,
		InstanceID:   req.InstanceID,
		Filename:     req.Filename,
		ResourceType: req.ResourceType,
		ErrorCode:    req.ErrorCode,
	}
}

func (s *Service) expireStaleResourceExports(ctx context.Context, deviceIDs []string, now time.Time) {
	if len(deviceIDs) == 0 {
		return
	}
	_ = s.db.WithContext(ctx).Model(&models.InstanceResourceExportRequest{}).
		Where("device_id IN ? AND status = ? AND expires_at < ?", deviceIDs, models.ResourceExportStatusQueued, now).
		Update("status", models.ResourceExportStatusExpired).Error
}

func FindUploadedInstanceResource(inst *models.LauncherInstance, filename, resourceType string) *models.InstanceResourceEntry {
	resourceType = normalizeResourceType(resourceType)
	for _, entry := range append(append(append(inst.Mods, inst.ResourcePacks...), inst.Shaders...), inst.Datapacks...) {
		e := entry
		if e.Source != "upload" || e.Filename != filename {
			continue
		}
		if normalizeResourceType(e.ResourceType) == resourceType {
			return &e
		}
	}
	return nil
}
