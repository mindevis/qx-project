package models

import "time"

type OfflineProfile struct {
	ID             string    `gorm:"type:char(36);primaryKey" json:"id"`
	UserID         *string   `gorm:"type:char(36);index" json:"user_id,omitempty"`
	GuestSessionID *string   `gorm:"type:char(36);index" json:"guest_session_id,omitempty"`
	Username       string    `gorm:"type:varchar(16);not null" json:"username"`
	OfflineUUID    string    `gorm:"type:char(36);not null" json:"offline_uuid"`
	CreatedAt      time.Time `json:"created_at"`
}

func (OfflineProfile) TableName() string {
	return "offline_profiles"
}
