package models

import "time"

const (
	ResourceExportStatusQueued     = "queued"
	ResourceExportStatusDispatched = "dispatched"
	ResourceExportStatusCompleted  = "completed"
	ResourceExportStatusFailed     = "failed"
	ResourceExportStatusExpired    = "expired"
)

type InstanceResourceExportRequest struct {
	ID           string     `gorm:"type:char(36);primaryKey" json:"id"`
	DeviceID     string     `gorm:"type:char(36);not null;index" json:"device_id"`
	InstanceID   string     `gorm:"type:char(36);not null;index" json:"instance_id"`
	Filename     string     `gorm:"type:varchar(256);not null" json:"filename"`
	ResourceType string     `gorm:"type:varchar(32);not null;default:mod" json:"resource_type"`
	ContentB64   string     `gorm:"type:longtext" json:"content_b64,omitempty"`
	FileSize     int64      `gorm:"type:bigint" json:"file_size,omitempty"`
	Status       string     `gorm:"type:varchar(32);not null;default:queued;index" json:"status"`
	ErrorCode    *string    `gorm:"type:varchar(64);column:error_code" json:"error_code,omitempty"`
	ExpiresAt    time.Time  `json:"expires_at"`
	DispatchedAt *time.Time `json:"dispatched_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (InstanceResourceExportRequest) TableName() string {
	return "instance_resource_export_requests"
}
