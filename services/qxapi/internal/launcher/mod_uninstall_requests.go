package launcher

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"gorm.io/gorm"
)

const modUninstallRequestTTL = 10 * time.Minute
const modUninstallBridgeTimeout = 25 * time.Second

type ModUninstallRequestView struct {
	ID           string  `json:"id"`
	Status       string  `json:"status"`
	InstanceID   string  `json:"instance_id"`
	Source       string  `json:"source"`
	ProjectID    string  `json:"project_id,omitempty"`
	Filename     string  `json:"filename"`
	ResourceType string  `json:"resource_type"`
	ErrorCode    *string `json:"error_code,omitempty"`
}

type UpdateModUninstallRequestInput struct {
	Status    string
	ErrorCode *string
}

func (s *Service) DeleteInstanceResourceWithBridge(ctx context.Context, owner Owner, instanceID string, in DeleteInstanceResourceInput) error {
	if in.Source == "" || (in.ProjectID == "" && in.Filename == "") {
		return ErrValidation
	}
	inst, err := s.GetInstance(ctx, owner, instanceID)
	if err != nil {
		return err
	}
	var target *models.InstanceResourceEntry
	for _, entry := range append(append(append(inst.Mods, inst.ResourcePacks...), inst.Shaders...), inst.Datapacks...) {
		e := entry
		if resourceEntryMatches(e, in) {
			target = &e
			break
		}
	}
	if target == nil {
		return ErrNotFound
	}
	deviceID, err := s.requireLinkedDevice(ctx, owner)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	req := models.ModUninstallRequest{
		ID:           uuid.NewString(),
		DeviceID:     deviceID,
		InstanceID:   instanceID,
		Source:       target.Source,
		ProjectID:    target.ProjectID,
		Filename:     target.Filename,
		ResourceType: normalizeResourceType(target.ResourceType),
		Status:       models.ModUninstallStatusQueued,
		ExpiresAt:    now.Add(modUninstallRequestTTL),
		CreatedAt:    now,
	}
	if err := s.db.WithContext(ctx).Create(&req).Error; err != nil {
		return err
	}

	ch := make(chan fileRPCResult, 1)
	s.pendingUninstallRPC.Store(req.ID, ch)
	defer s.pendingUninstallRPC.Delete(req.ID)

	waitCtx, cancel := context.WithTimeout(ctx, modUninstallBridgeTimeout)
	defer cancel()

	select {
	case res := <-ch:
		if res.err != nil {
			return res.err
		}
		return s.removeInstanceResourceFromDB(ctx, instanceID, in)
	case <-waitCtx.Done():
		_ = s.db.WithContext(context.Background()).Model(&req).Update("status", models.ModUninstallStatusExpired).Error
		if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
			return ErrBridgeTimeout
		}
		return waitCtx.Err()
	}
}

func (s *Service) removeInstanceResourceFromDB(ctx context.Context, instanceID string, in DeleteInstanceResourceInput) error {
	var inst models.LauncherInstance
	if err := s.db.WithContext(ctx).Where("id = ?", instanceID).First(&inst).Error; err != nil {
		return err
	}
	if !removeInstanceResource(&inst, in) {
		return ErrNotFound
	}
	return s.db.WithContext(ctx).Model(&inst).Updates(map[string]any{
		"mods":           inst.Mods,
		"resource_packs": inst.ResourcePacks,
		"shaders":        inst.Shaders,
		"datapacks":      inst.Datapacks,
	}).Error
}

func (s *Service) FetchPendingModUninstall(ctx context.Context, deviceID string) (*ModUninstallRequestView, error) {
	if deviceID == "" {
		return nil, ErrValidation
	}
	now := time.Now().UTC()
	deviceIDs, err := s.deliveryDeviceIDs(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	s.expireStaleModUninstalls(ctx, deviceIDs, now)

	var reqs []models.ModUninstallRequest
	err = s.db.WithContext(ctx).
		Where("device_id IN ? AND status = ?", deviceIDs, models.ModUninstallStatusQueued).
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
		"status":        models.ModUninstallStatusDispatched,
		"dispatched_at": dispatched,
		"device_id":     deviceID,
	}).Error; err != nil {
		return nil, err
	}
	req.Status = models.ModUninstallStatusDispatched
	req.DeviceID = deviceID
	return modUninstallViewFromModel(req), nil
}

func (s *Service) UpdateModUninstallRequest(ctx context.Context, deviceID, requestID string, in UpdateModUninstallRequestInput) (*ModUninstallRequestView, error) {
	if deviceID == "" || requestID == "" || in.Status == "" {
		return nil, ErrValidation
	}
	var req models.ModUninstallRequest
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
	if in.Status == models.ModUninstallStatusCompleted || in.Status == models.ModUninstallStatusFailed {
		updates["completed_at"] = now
	}
	if err := s.db.WithContext(ctx).Model(&req).Updates(updates).Error; err != nil {
		return nil, err
	}

	var bridgeErr error
	if in.Status == models.ModUninstallStatusCompleted {
		bridgeErr = nil
	}
	if in.Status == models.ModUninstallStatusFailed {
		code := "UNINSTALL_FAILED"
		if in.ErrorCode != nil && *in.ErrorCode != "" {
			code = *in.ErrorCode
		}
		bridgeErr = errors.New(code)
	}
	s.deliverUninstallRPC(requestID, bridgeErr)

	req.Status = in.Status
	if in.ErrorCode != nil {
		req.ErrorCode = in.ErrorCode
	}
	return modUninstallViewFromModel(req), nil
}

func (s *Service) deliverUninstallRPC(requestID string, err error) {
	if requestID == "" {
		return
	}
	raw, ok := s.pendingUninstallRPC.Load(requestID)
	if !ok {
		return
	}
	ch := raw.(chan fileRPCResult)
	select {
	case ch <- fileRPCResult{err: err}:
	default:
	}
}

func modUninstallViewFromModel(req models.ModUninstallRequest) *ModUninstallRequestView {
	return &ModUninstallRequestView{
		ID:           req.ID,
		Status:       req.Status,
		InstanceID:   req.InstanceID,
		Source:       req.Source,
		ProjectID:    req.ProjectID,
		Filename:     req.Filename,
		ResourceType: req.ResourceType,
		ErrorCode:    req.ErrorCode,
	}
}

func (s *Service) expireStaleModUninstalls(ctx context.Context, deviceIDs []string, now time.Time) {
	if len(deviceIDs) == 0 {
		return
	}
	_ = s.db.WithContext(ctx).Model(&models.ModUninstallRequest{}).
		Where("device_id IN ? AND status = ? AND expires_at < ?", deviceIDs, models.ModUninstallStatusQueued, now).
		Update("status", models.ModUninstallStatusExpired).Error
}
