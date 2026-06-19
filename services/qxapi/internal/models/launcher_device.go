package models

import "time"

const (
	DeviceStatusPendingLink = "pending_link"
	DeviceStatusLinked      = "linked"
	DeviceStatusExpired     = "expired"
	DeviceStatusRevoked     = "revoked"
)

type LauncherDevice struct {
	ID              string     `gorm:"type:char(36);primaryKey" json:"id"`
	DeviceID        string     `gorm:"type:char(36);not null;uniqueIndex" json:"device_id"`
	UserID          *string    `gorm:"type:char(36);index" json:"user_id,omitempty"`
	GuestSessionID  *string    `gorm:"type:char(36);index" json:"guest_session_id,omitempty"`
	Status          string     `gorm:"type:varchar(32);not null;default:pending_link;index" json:"status"`
	UserCode        *string    `gorm:"type:varchar(16);index" json:"user_code,omitempty"`
	DeviceTokenHash *string    `gorm:"type:varchar(255)" json:"-"`
	Hostname        *string    `gorm:"type:varchar(255)" json:"hostname,omitempty"`
	OS              *string    `gorm:"type:varchar(64)" json:"os,omitempty"`
	LauncherVersion *string    `gorm:"type:varchar(32)" json:"launcher_version,omitempty"`
	LinkExpiresAt   *time.Time `json:"link_expires_at,omitempty"`
	LinkedAt        *time.Time `json:"linked_at,omitempty"`
	LastSeenAt      *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

func (LauncherDevice) TableName() string {
	return "launcher_devices"
}
