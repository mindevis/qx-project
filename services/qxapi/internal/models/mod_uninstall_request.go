package models

import "time"

const (
	ModUninstallStatusQueued     = "queued"
	ModUninstallStatusDispatched = "dispatched"
	ModUninstallStatusCompleted  = "completed"
	ModUninstallStatusFailed     = "failed"
	ModUninstallStatusExpired    = "expired"
)

type ModUninstallRequest struct {
	ID           string     `gorm:"type:char(36);primaryKey" json:"id"`
	DeviceID     string     `gorm:"type:char(36);not null;index" json:"device_id"`
	InstanceID   string     `gorm:"type:char(36);not null;index" json:"instance_id"`
	Source       string     `gorm:"type:varchar(16);not null" json:"source"`
	ProjectID    string     `gorm:"type:varchar(128)" json:"project_id"`
	Filename     string     `gorm:"type:varchar(256);not null" json:"filename"`
	ResourceType string     `gorm:"type:varchar(32);not null;default:mod" json:"resource_type"`
	Status       string     `gorm:"type:varchar(32);not null;default:queued;index" json:"status"`
	ErrorCode    *string    `gorm:"type:varchar(64);column:error_code" json:"error_code,omitempty"`
	ExpiresAt    time.Time  `json:"expires_at"`
	DispatchedAt *time.Time `json:"dispatched_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (ModUninstallRequest) TableName() string {
	return "mod_uninstall_requests"
}
