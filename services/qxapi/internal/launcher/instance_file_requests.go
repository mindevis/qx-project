package launcher

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/qxproject/qx/pkg/protocol"
	"github.com/qxproject/qx/services/qxapi/internal/models"
	"gorm.io/gorm"
)

const instanceFileRequestTTL = 30 * time.Second
const instanceFileBridgeTimeout = 25 * time.Second

type fileRPCResult struct {
	resultJSON string
	err        error
}

type InstanceFileRequestView struct {
	ID           string  `json:"id"`
	Status       string  `json:"status"`
	InstanceID   string  `json:"instance_id"`
	Operation    string  `json:"operation"`
	Path         string  `json:"path"`
	WriteContent string  `json:"write_content,omitempty"`
	ResultJSON   string  `json:"result_json,omitempty"`
	ErrorCode    *string `json:"error_code,omitempty"`
}

type UpdateInstanceFileRequestInput struct {
	Status     string
	ResultJSON string
	ErrorCode  *string
}

func (s *Service) ListInstanceFiles(ctx context.Context, owner Owner, instanceID, path string) ([]protocol.FileEntry, error) {
	raw, err := s.runInstanceFileBridge(ctx, owner, instanceID, models.InstanceFileOpList, path, "")
	if err != nil {
		return nil, err
	}
	var result protocol.InstanceFilesListResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result.Entries, nil
}

func (s *Service) ReadInstanceFile(ctx context.Context, owner Owner, instanceID, path string) (*protocol.InstanceFilesReadResult, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, ErrValidation
	}
	raw, err := s.runInstanceFileBridge(ctx, owner, instanceID, models.InstanceFileOpRead, path, "")
	if err != nil {
		return nil, err
	}
	var result protocol.InstanceFilesReadResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) WriteInstanceFile(ctx context.Context, owner Owner, instanceID, path, content string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return ErrValidation
	}
	if len(content) > 2*1024*1024 {
		return ErrValidation
	}
	_, err := s.runInstanceFileBridge(ctx, owner, instanceID, models.InstanceFileOpWrite, path, content)
	return err
}

func (s *Service) runInstanceFileBridge(ctx context.Context, owner Owner, instanceID, operation, path, writeContent string) ([]byte, error) {
	deviceID, err := s.requireLinkedDevice(ctx, owner)
	if err != nil {
		return nil, err
	}
	if _, err := s.GetInstance(ctx, owner, instanceID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	req := models.InstanceFileRequest{
		ID:           uuid.NewString(),
		DeviceID:     deviceID,
		InstanceID:   instanceID,
		Operation:    operation,
		Path:         path,
		WriteContent: writeContent,
		Status:       models.InstanceFileStatusQueued,
		ExpiresAt:    now.Add(instanceFileRequestTTL),
		CreatedAt:    now,
	}
	if err := s.db.WithContext(ctx).Create(&req).Error; err != nil {
		return nil, err
	}

	ch := make(chan fileRPCResult, 1)
	s.pendingFileRPC.Store(req.ID, ch)
	defer s.pendingFileRPC.Delete(req.ID)

	waitCtx, cancel := context.WithTimeout(ctx, instanceFileBridgeTimeout)
	defer cancel()

	select {
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		if operation == models.InstanceFileOpWrite {
			return []byte(`{"status":"ok"}`), nil
		}
		return []byte(res.resultJSON), nil
	case <-waitCtx.Done():
		_ = s.db.WithContext(context.Background()).Model(&req).Update("status", models.InstanceFileStatusExpired).Error
		if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
			return nil, ErrBridgeTimeout
		}
		return nil, waitCtx.Err()
	}
}

func (s *Service) FetchPendingInstanceFile(ctx context.Context, deviceID string) (*InstanceFileRequestView, error) {
	if deviceID == "" {
		return nil, ErrValidation
	}
	now := time.Now().UTC()
	s.expireStaleInstanceFiles(ctx, deviceID, now)

	var reqs []models.InstanceFileRequest
	err := s.db.WithContext(ctx).
		Where("device_id = ? AND status = ?", deviceID, models.InstanceFileStatusQueued).
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
		"status":        models.InstanceFileStatusDispatched,
		"dispatched_at": dispatched,
	}).Error; err != nil {
		return nil, err
	}
	req.Status = models.InstanceFileStatusDispatched
	return instanceFileViewFromModel(req, true), nil
}

func (s *Service) UpdateInstanceFileRequest(ctx context.Context, deviceID, requestID string, in UpdateInstanceFileRequestInput) (*InstanceFileRequestView, error) {
	if deviceID == "" || requestID == "" || in.Status == "" {
		return nil, ErrValidation
	}
	var req models.InstanceFileRequest
	if err := s.db.WithContext(ctx).Where("id = ? AND device_id = ?", requestID, deviceID).First(&req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	updates := map[string]any{"status": in.Status}
	if in.ResultJSON != "" {
		updates["result_json"] = in.ResultJSON
	}
	if in.ErrorCode != nil {
		updates["error_code"] = in.ErrorCode
	}
	now := time.Now().UTC()
	if in.Status == models.InstanceFileStatusCompleted || in.Status == models.InstanceFileStatusFailed {
		updates["completed_at"] = now
	}
	if err := s.db.WithContext(ctx).Model(&req).Updates(updates).Error; err != nil {
		return nil, err
	}

	var bridgeErr error
	if in.Status == models.InstanceFileStatusCompleted {
		bridgeErr = nil
	}
	if in.Status == models.InstanceFileStatusFailed {
		code := "FILE_OP_FAILED"
		if in.ErrorCode != nil && *in.ErrorCode != "" {
			code = *in.ErrorCode
		}
		bridgeErr = errors.New(code)
	}
	s.deliverFileRPC(requestID, in.ResultJSON, bridgeErr)

	req.Status = in.Status
	req.ResultJSON = in.ResultJSON
	if in.ErrorCode != nil {
		req.ErrorCode = in.ErrorCode
	}
	return instanceFileViewFromModel(req, false), nil
}

func (s *Service) deliverFileRPC(requestID, resultJSON string, err error) {
	if requestID == "" {
		return
	}
	raw, ok := s.pendingFileRPC.Load(requestID)
	if !ok {
		return
	}
	ch := raw.(chan fileRPCResult)
	select {
	case ch <- fileRPCResult{resultJSON: resultJSON, err: err}:
	default:
	}
}

func instanceFileViewFromModel(req models.InstanceFileRequest, includePayload bool) *InstanceFileRequestView {
	view := &InstanceFileRequestView{
		ID:         req.ID,
		Status:     req.Status,
		InstanceID: req.InstanceID,
		Operation:  req.Operation,
		Path:       req.Path,
		ErrorCode:  req.ErrorCode,
	}
	if includePayload {
		view.WriteContent = req.WriteContent
	}
	return view
}

func (s *Service) expireStaleInstanceFiles(ctx context.Context, deviceID string, now time.Time) {
	_ = s.db.WithContext(ctx).Model(&models.InstanceFileRequest{}).
		Where("device_id = ? AND status = ? AND expires_at < ?", deviceID, models.InstanceFileStatusQueued, now).
		Update("status", models.InstanceFileStatusExpired).Error
}

func (s *Service) requireLinkedDevice(ctx context.Context, owner Owner) (string, error) {
	deviceID, err := s.FindLinkedDevice(ctx, owner)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", ErrDeviceNotLinked
		}
		return "", err
	}
	return deviceID, nil
}
