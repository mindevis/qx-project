package launcher

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/qxproject/qx/services/qxapi/internal/models"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{3,16}$`)

type CreateProfileInput struct {
	Username string
}

func offlineUUID(username string) string {
	return uuid.NewMD5(uuid.NameSpaceOID, []byte("OfflinePlayer:"+username)).String()
}

func (s *Service) ListProfiles(ctx context.Context, owner Owner) ([]models.OfflineProfile, error) {
	q := s.db.WithContext(ctx).Model(&models.OfflineProfile{})
	q = scopeOwner(q, owner)
	var items []models.OfflineProfile
	if err := q.Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Service) CreateProfile(ctx context.Context, owner Owner, in CreateProfileInput) (*models.OfflineProfile, error) {
	username := strings.TrimSpace(in.Username)
	if !usernamePattern.MatchString(username) {
		return nil, ErrValidation
	}
	now := time.Now().UTC()
	profile := models.OfflineProfile{
		ID:           uuid.NewString(),
		Username:     username,
		OfflineUUID:  offlineUUID(username),
		CreatedAt:    now,
	}
	if owner.IsGuest {
		profile.GuestSessionID = &owner.GuestSessionID
	} else {
		profile.UserID = &owner.UserID
	}
	if err := s.db.WithContext(ctx).Create(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *Service) DeleteProfile(ctx context.Context, owner Owner, profileID string) error {
	q := s.db.WithContext(ctx).Where("id = ?", profileID)
	q = scopeOwner(q, owner)
	res := q.Delete(&models.OfflineProfile{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) getProfile(ctx context.Context, owner Owner, profileID string) (*models.OfflineProfile, error) {
	q := s.db.WithContext(ctx).Where("id = ?", profileID)
	q = scopeOwner(q, owner)
	var profile models.OfflineProfile
	if err := q.First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &profile, nil
}
