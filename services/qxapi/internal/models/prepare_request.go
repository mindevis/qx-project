package models

import "time"

const (
	PrepareStatusQueued      = "queued"
	PrepareStatusPreparing   = "preparing"
	PrepareStatusDownloading = "downloading"
	PrepareStatusCompleted   = "completed"
	PrepareStatusFailed      = "failed"
	PrepareStatusExpired     = "expired"
)

type PrepareRequest struct {
	ID           string     `gorm:"type:char(36);primaryKey" json:"id"`
	DeviceID     string     `gorm:"type:char(36);not null;index" json:"device_id"`
	InstanceID   string     `gorm:"type:char(36);not null;index" json:"instance_id"`
	Status       string     `gorm:"type:varchar(32);not null;default:queued;index" json:"status"`
	ErrorCode    *string    `gorm:"type:varchar(64);column:error_code" json:"error_code,omitempty"`
	ExpiresAt    time.Time  `json:"expires_at"`
	DispatchedAt *time.Time `json:"dispatched_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (PrepareRequest) TableName() string {
	return "prepare_requests"
}
