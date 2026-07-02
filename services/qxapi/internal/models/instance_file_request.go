package models

import "time"

const (
	InstanceFileOpList  = "list"
	InstanceFileOpRead  = "read"
	InstanceFileOpWrite = "write"

	InstanceFileStatusQueued     = "queued"
	InstanceFileStatusDispatched = "dispatched"
	InstanceFileStatusCompleted  = "completed"
	InstanceFileStatusFailed     = "failed"
	InstanceFileStatusExpired    = "expired"
)

type InstanceFileRequest struct {
	ID           string     `gorm:"type:char(36);primaryKey" json:"id"`
	DeviceID     string     `gorm:"type:char(36);not null;index" json:"device_id"`
	InstanceID   string     `gorm:"type:char(36);not null;index" json:"instance_id"`
	Operation    string     `gorm:"type:varchar(16);not null" json:"operation"`
	Path         string     `gorm:"type:text" json:"path"`
	WriteContent string     `gorm:"type:longtext" json:"write_content,omitempty"`
	ResultJSON   string     `gorm:"type:longtext" json:"result_json,omitempty"`
	Status       string     `gorm:"type:varchar(32);not null;default:queued;index" json:"status"`
	ErrorCode    *string    `gorm:"type:varchar(64);column:error_code" json:"error_code,omitempty"`
	ExpiresAt    time.Time  `json:"expires_at"`
	DispatchedAt *time.Time `json:"dispatched_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (InstanceFileRequest) TableName() string {
	return "instance_file_requests"
}
