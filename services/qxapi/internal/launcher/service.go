package launcher

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrValidation       = errors.New("validation error")
	ErrDeviceNotLinked  = errors.New("device not linked")
	ErrLinkExpired      = errors.New("link expired")
	ErrGuestLoaderOnly  = errors.New("guest may only use vanilla loader")
	ErrDeviceNotPending = errors.New("device is not pending link")
)

const (
	linkTTL         = 15 * time.Minute
	deviceTokenTTL  = 30 * 24 * time.Hour
	guestSessionTTL = 24 * time.Hour
	pollIntervalSec = 3
)

type Service struct {
	db         *gorm.DB
	tokens     *auth.TokenService
	webBaseURL string
	manifest   ManifestProvider
}

func NewService(db *gorm.DB, tokens *auth.TokenService, webBaseURL string) *Service {
	return &Service{db: db, tokens: tokens, webBaseURL: strings.TrimRight(webBaseURL, "/")}
}

type RegisterDeviceInput struct {
	DeviceID        string
	OS              string
	Hostname        string
	LauncherVersion string
}

type RegisterDeviceResult struct {
	DeviceID        string    `json:"device_id"`
	Status          string    `json:"status"`
	UserCode        string    `json:"user_code"`
	LinkURL         string    `json:"link_url"`
	PollIntervalSec int       `json:"poll_interval_sec"`
	ExpiresAt       time.Time `json:"expires_at"`
}

func (s *Service) RegisterDevice(ctx context.Context, in RegisterDeviceInput) (*RegisterDeviceResult, error) {
	deviceID := strings.TrimSpace(in.DeviceID)
	if deviceID == "" {
		return nil, ErrValidation
	}

	now := time.Now().UTC()
	expires := now.Add(linkTTL)
	userCode := generateUserCode()
	linkURL := fmt.Sprintf("%s/launcher/link?device=%s", s.webBaseURL, deviceID)

	var device models.LauncherDevice
	err := s.db.WithContext(ctx).Where("device_id = ?", deviceID).First(&device).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		device = models.LauncherDevice{
			ID:              uuid.NewString(),
			DeviceID:        deviceID,
			Status:          models.DeviceStatusPendingLink,
			UserCode:        &userCode,
			LinkExpiresAt:   &expires,
			LastSeenAt:      &now,
			CreatedAt:       now,
		}
	} else if err != nil {
		return nil, err
	} else {
		device.Status = models.DeviceStatusPendingLink
		device.UserCode = &userCode
		device.LinkExpiresAt = &expires
		device.LastSeenAt = &now
		device.UserID = nil
		device.GuestSessionID = nil
		device.DeviceTokenHash = nil
		device.LinkedAt = nil
	}

	if in.OS != "" {
		os := in.OS
		device.OS = &os
	}
	if in.Hostname != "" {
		host := in.Hostname
		device.Hostname = &host
	}
	if in.LauncherVersion != "" {
		ver := in.LauncherVersion
		device.LauncherVersion = &ver
	}

	if err := s.db.WithContext(ctx).Save(&device).Error; err != nil {
		return nil, err
	}

	return &RegisterDeviceResult{
		DeviceID:        deviceID,
		Status:          models.DeviceStatusPendingLink,
		UserCode:        userCode,
		LinkURL:         linkURL,
		PollIntervalSec: pollIntervalSec,
		ExpiresAt:       expires,
	}, nil
}

type DeviceStatusResult struct {
	Status         string  `json:"status"`
	DeviceToken    *string `json:"device_token,omitempty"`
	OwnerType      *string `json:"owner_type,omitempty"`
	GuestSessionID *string `json:"guest_session_id,omitempty"`
	UserID         *string `json:"user_id,omitempty"`
}

func (s *Service) DeviceStatus(ctx context.Context, deviceID string) (*DeviceStatusResult, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, ErrValidation
	}

	device, err := s.getDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	device.LastSeenAt = &now
	_ = s.db.WithContext(ctx).Model(device).Update("last_seen_at", now).Error

	if device.Status == models.DeviceStatusPendingLink && device.LinkExpiresAt != nil && now.After(*device.LinkExpiresAt) {
		device.Status = models.DeviceStatusExpired
		_ = s.db.WithContext(ctx).Model(device).Update("status", models.DeviceStatusExpired).Error
	}

	result := &DeviceStatusResult{Status: device.Status}
	if device.Status != models.DeviceStatusLinked {
		return result, nil
	}

	token, err := s.tokens.IssueDeviceToken(device.DeviceID, deviceTokenTTL)
	if err != nil {
		return nil, err
	}
	result.DeviceToken = &token

	if device.UserID != nil {
		owner := "user"
		result.OwnerType = &owner
		result.UserID = device.UserID
	} else if device.GuestSessionID != nil {
		owner := "guest"
		result.OwnerType = &owner
		result.GuestSessionID = device.GuestSessionID
	}
	return result, nil
}

type LinkDeviceInput struct {
	DeviceID string
	UserCode string
	UserID   string
}

type LinkDeviceResult struct {
	Status         string `json:"status"`
	GuestToken     string `json:"guest_token,omitempty"`
	GuestExpiresIn int64  `json:"guest_expires_in,omitempty"`
	OwnerType      string `json:"owner_type"`
}

func (s *Service) LinkDevice(ctx context.Context, in LinkDeviceInput) (*LinkDeviceResult, error) {
	deviceID := strings.TrimSpace(in.DeviceID)
	if deviceID == "" {
		return nil, ErrValidation
	}

	device, err := s.getDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if device.Status != models.DeviceStatusPendingLink {
		return nil, ErrDeviceNotPending
	}
	if device.LinkExpiresAt != nil && time.Now().UTC().After(*device.LinkExpiresAt) {
		_ = s.db.WithContext(ctx).Model(device).Update("status", models.DeviceStatusExpired).Error
		return nil, ErrLinkExpired
	}
	if in.UserCode != "" && device.UserCode != nil && !strings.EqualFold(in.UserCode, *device.UserCode) {
		return nil, ErrValidation
	}

	now := time.Now().UTC()
	updates := map[string]any{
		"status":     models.DeviceStatusLinked,
		"linked_at":  now,
		"user_code":  nil,
	}

	if in.UserID != "" {
		updates["user_id"] = in.UserID
		updates["guest_session_id"] = nil
		if err := s.db.WithContext(ctx).Model(device).Updates(updates).Error; err != nil {
			return nil, err
		}
		return &LinkDeviceResult{Status: models.DeviceStatusLinked, OwnerType: "user"}, nil
	}

	guestID := uuid.NewString()
	guestToken, ttl, err := s.tokens.IssueGuestToken(guestID)
	if err != nil {
		return nil, err
	}
	_ = s.db.WithContext(ctx).Where("device_id = ?", deviceID).Delete(&models.GuestSession{}).Error
	guest := models.GuestSession{
		ID:             guestID,
		DeviceID:       deviceID,
		GuestTokenHash: auth.HashToken(guestToken),
		ExpiresAt:      now.Add(guestSessionTTL),
		CreatedAt:      now,
	}
	if err := s.db.WithContext(ctx).Create(&guest).Error; err != nil {
		return nil, err
	}

	updates["guest_session_id"] = guestID
	updates["user_id"] = nil
	if err := s.db.WithContext(ctx).Model(device).Updates(updates).Error; err != nil {
		return nil, err
	}

	return &LinkDeviceResult{
		Status:         models.DeviceStatusLinked,
		GuestToken:     guestToken,
		GuestExpiresIn: int64(ttl.Seconds()),
		OwnerType:      "guest",
	}, nil
}

type Owner struct {
	UserID         string
	GuestSessionID string
	IsGuest        bool
}

type CreateInstanceInput struct {
	Name      string
	MCVersion string
	Loader    string
}

func (s *Service) ListInstances(ctx context.Context, owner Owner) ([]models.LauncherInstance, error) {
	q := s.db.WithContext(ctx).Model(&models.LauncherInstance{})
	q = scopeOwner(q, owner)
	var items []models.LauncherInstance
	if err := q.Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) CreateInstance(ctx context.Context, owner Owner, in CreateInstanceInput) (*models.LauncherInstance, error) {
	name := strings.TrimSpace(in.Name)
	mcVersion := strings.TrimSpace(in.MCVersion)
	loader := strings.TrimSpace(in.Loader)
	if loader == "" {
		loader = models.LoaderVanilla
	}
	if name == "" || mcVersion == "" {
		return nil, ErrValidation
	}
	if owner.IsGuest && loader != models.LoaderVanilla {
		return nil, ErrGuestLoaderOnly
	}

	now := time.Now().UTC()
	inst := models.LauncherInstance{
		ID:        uuid.NewString(),
		Name:      name,
		MCVersion: mcVersion,
		Loader:    loader,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if owner.IsGuest {
		inst.GuestSessionID = &owner.GuestSessionID
	} else {
		inst.UserID = &owner.UserID
	}
	if err := s.db.WithContext(ctx).Create(&inst).Error; err != nil {
		return nil, err
	}
	return &inst, nil
}

func (s *Service) DeleteInstance(ctx context.Context, owner Owner, instanceID string) error {
	q := s.db.WithContext(ctx).Where("id = ?", instanceID)
	q = scopeOwner(q, owner)
	res := q.Delete(&models.LauncherInstance{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) getDevice(ctx context.Context, deviceID string) (*models.LauncherDevice, error) {
	var device models.LauncherDevice
	if err := s.db.WithContext(ctx).Where("device_id = ?", deviceID).First(&device).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &device, nil
}

type DeviceMeResult struct {
	DeviceID       string  `json:"device_id"`
	Status         string  `json:"status"`
	OwnerType      string  `json:"owner_type"`
	UserID         *string `json:"user_id,omitempty"`
	GuestSessionID *string `json:"guest_session_id,omitempty"`
}

type UnlinkDeviceResult struct {
	Status string `json:"status"`
}

func (s *Service) UnlinkDevice(ctx context.Context, deviceID string) (*UnlinkDeviceResult, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, ErrValidation
	}
	device, err := s.getDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if device.Status != models.DeviceStatusLinked {
		return nil, ErrDeviceNotLinked
	}
	_ = s.db.WithContext(ctx).Where("device_id = ?", deviceID).Delete(&models.GuestSession{}).Error
	updates := map[string]any{
		"status":            models.DeviceStatusPendingLink,
		"user_id":           nil,
		"guest_session_id":  nil,
		"device_token_hash": nil,
		"linked_at":         nil,
		"user_code":         nil,
		"link_expires_at":   nil,
	}
	if err := s.db.WithContext(ctx).Model(device).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &UnlinkDeviceResult{Status: models.DeviceStatusPendingLink}, nil
}

func (s *Service) UnlinkDeviceForOwner(ctx context.Context, owner Owner) (*UnlinkDeviceResult, error) {
	deviceID, err := s.FindLinkedDevice(ctx, owner)
	if err != nil {
		return nil, err
	}
	return s.UnlinkDevice(ctx, deviceID)
}

func (s *Service) DeviceMe(ctx context.Context, deviceID string) (*DeviceMeResult, error) {
	device, err := s.getDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	out := &DeviceMeResult{
		DeviceID: device.DeviceID,
		Status:   device.Status,
	}
	switch {
	case device.UserID != nil:
		out.OwnerType = "user"
		out.UserID = device.UserID
	case device.GuestSessionID != nil:
		out.OwnerType = "guest"
		out.GuestSessionID = device.GuestSessionID
	default:
		out.OwnerType = "none"
	}
	return out, nil
}

func scopeOwner(q *gorm.DB, owner Owner) *gorm.DB {
	if owner.IsGuest {
		return q.Where("guest_session_id = ?", owner.GuestSessionID)
	}
	return q.Where("user_id = ?", owner.UserID)
}

func generateUserCode() string {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	const digits = "23456789"
	part := func(charset string, n int) string {
		out := make([]byte, n)
		for i := range out {
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
			out[i] = charset[n.Int64()]
		}
		return string(out)
	}
	return part(letters, 4) + "-" + part(digits, 4)
}
