package launcher

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/auth"
	"github.com/qxproject/qx/services/qxapi/internal/cosmetics"
	"github.com/qxproject/qx/services/qxapi/internal/mojang"
	"github.com/qxproject/qx/services/qxapi/internal/models"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrValidation       = errors.New("validation error")
	ErrDeviceNotLinked  = errors.New("device not linked")
	ErrLinkExpired      = errors.New("link expired")
	ErrAuthRequired     = errors.New("authentication required")
	ErrDeviceNotPending = errors.New("device is not pending link")
	ErrManifest         = errors.New("manifest build failed")
	ErrMojangSession    = errors.New("mojang session failed")
)

const (
	linkTTL         = 15 * time.Minute
	deviceTokenTTL  = 30 * 24 * time.Hour
	pollIntervalSec = 3
)

type Service struct {
	db         *gorm.DB
	tokens     *auth.TokenService
	webBaseURL string
	manifest   ManifestProvider
	mojang     *mojang.Service
	cosmetics  *cosmetics.Service
}

func NewService(db *gorm.DB, tokens *auth.TokenService, webBaseURL string) *Service {
	return &Service{db: db, tokens: tokens, webBaseURL: strings.TrimRight(webBaseURL, "/")}
}

func (s *Service) SetMojang(m *mojang.Service) {
	s.mojang = m
}

func (s *Service) SetCosmetics(c *cosmetics.Service) {
	s.cosmetics = c
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
	linkURL := fmt.Sprintf("%s/launcher/link?device=%s", s.webBaseURL, deviceID)

	var device models.LauncherDevice
	err := s.db.WithContext(ctx).Where("device_id = ?", deviceID).First(&device).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		device = models.LauncherDevice{
			ID:            uuid.NewString(),
			DeviceID:      deviceID,
			Status:        models.DeviceStatusPendingLink,
			LinkExpiresAt: &expires,
			LastSeenAt:    &now,
			CreatedAt:     now,
		}
	} else if err != nil {
		return nil, err
	} else {
		device.Status = models.DeviceStatusPendingLink
		device.UserCode = nil
		device.LinkExpiresAt = &expires
		device.LastSeenAt = &now
		device.UserID = nil
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
		LinkURL:         linkURL,
		PollIntervalSec: pollIntervalSec,
		ExpiresAt:       expires,
	}, nil
}

type DeviceStatusResult struct {
	Status          string     `json:"status"`
	DeviceID        string     `json:"device_id,omitempty"`
	Hostname        *string    `json:"hostname,omitempty"`
	OS              *string    `json:"os,omitempty"`
	LauncherVersion *string    `json:"launcher_version,omitempty"`
	LinkExpiresAt   *time.Time `json:"link_expires_at,omitempty"`
	LastSeenAt      *time.Time `json:"last_seen_at,omitempty"`
	DeviceToken     *string    `json:"device_token,omitempty"`
	OwnerType *string `json:"owner_type,omitempty"`
	UserID    *string `json:"user_id,omitempty"`
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

	result := &DeviceStatusResult{
		Status:          device.Status,
		DeviceID:        device.DeviceID,
		Hostname:        device.Hostname,
		OS:              device.OS,
		LauncherVersion: device.LauncherVersion,
		LinkExpiresAt:   device.LinkExpiresAt,
		LastSeenAt:      device.LastSeenAt,
	}
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
	}
	return result, nil
}

type LinkDeviceInput struct {
	DeviceID string
	UserID   string
}

type LinkDeviceResult struct {
	Status    string `json:"status"`
	OwnerType string `json:"owner_type"`
}

func (s *Service) LinkDevice(ctx context.Context, in LinkDeviceInput) (*LinkDeviceResult, error) {
	deviceID := strings.TrimSpace(in.DeviceID)
	userID := strings.TrimSpace(in.UserID)
	if deviceID == "" {
		return nil, ErrValidation
	}
	if userID == "" {
		return nil, ErrAuthRequired
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

	now := time.Now().UTC()
	updates := map[string]any{
		"status":           models.DeviceStatusLinked,
		"linked_at":        now,
		"user_code":        nil,
		"user_id":          userID,
		"guest_session_id": nil,
	}
	if err := s.db.WithContext(ctx).Model(device).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &LinkDeviceResult{Status: models.DeviceStatusLinked, OwnerType: "user"}, nil
}

type CreateInstanceInput struct {
	Name          string
	MCVersion     string
	Loader        string
	LoaderVersion string
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
	loaderVersion := strings.TrimSpace(in.LoaderVersion)
	if name == "" || mcVersion == "" {
		return nil, ErrValidation
	}
	if !isSupportedInstanceLoader(loader) {
		return nil, ErrValidation
	}
	if loaderRequiresVersion(loader) && loaderVersion == "" {
		return nil, ErrValidation
	}
	if err := validateLoaderVersion(loader, loaderVersion); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	inst := models.LauncherInstance{
		ID:        uuid.NewString(),
		Name:      name,
		MCVersion: mcVersion,
		Loader:    loader,
		UserID:    &owner.UserID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if loaderVersion != "" {
		inst.LoaderVersion = &loaderVersion
	}
	if err := s.db.WithContext(ctx).Create(&inst).Error; err != nil {
		return nil, err
	}
	return &inst, nil
}

func isSupportedInstanceLoader(loader string) bool {
	switch loader {
	case models.LoaderVanilla, models.LoaderForge, models.LoaderNeoForge, models.LoaderFabric, models.LoaderQuilt:
		return true
	default:
		return false
	}
}

func loaderRequiresVersion(loader string) bool {
	return loader != models.LoaderVanilla
}

func validateLoaderVersion(loader, loaderVersion string) error {
	loaderVersion = strings.TrimSpace(loaderVersion)
	if loaderVersion == "" {
		return nil
	}
	switch loader {
	case models.LoaderNeoForge:
		parts := strings.Split(loaderVersion, ".")
		if len(parts) < 2 {
			return ErrValidation
		}
		major, err := strconv.Atoi(parts[0])
		if err != nil || (major != 20 && major != 21) {
			return ErrValidation
		}
	case models.LoaderForge:
		if strings.HasPrefix(loaderVersion, "20.") || strings.HasPrefix(loaderVersion, "21.") {
			return ErrValidation
		}
	case models.LoaderFabric, models.LoaderQuilt:
		if strings.HasPrefix(loaderVersion, "47.") || strings.HasPrefix(loaderVersion, "21.") {
			return ErrValidation
		}
		parts := strings.Split(loaderVersion, ".")
		if len(parts) < 2 || parts[0] != "0" {
			return ErrValidation
		}
	}
	return nil
}

func instanceLoaderVersion(inst models.LauncherInstance) string {
	if inst.LoaderVersion == nil {
		return ""
	}
	return *inst.LoaderVersion
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
	DeviceID  string  `json:"device_id"`
	Status    string  `json:"status"`
	OwnerType string  `json:"owner_type"`
	UserID    *string `json:"user_id,omitempty"`
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
	default:
		out.OwnerType = "none"
	}
	return out, nil
}
