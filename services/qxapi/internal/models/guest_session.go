package models

import "time"

type GuestSession struct {
	ID             string    `gorm:"type:char(36);primaryKey" json:"id"`
	DeviceID       string    `gorm:"type:varchar(64);not null;uniqueIndex" json:"device_id"`
	GuestTokenHash string    `gorm:"type:varchar(255);not null" json:"-"`
	ExpiresAt      time.Time `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
}

func (GuestSession) TableName() string {
	return "guest_sessions"
}
