package models

import "time"

const (
	ModInstallStatusQueued      = "queued"
	ModInstallStatusDispatched  = "dispatched"
	ModInstallStatusDownloading = "downloading"
	ModInstallStatusCompleted   = "completed"
	ModInstallStatusFailed      = "failed"
	ModInstallStatusExpired     = "expired"
)

type ModInstallRequest struct {
	ID            string     `gorm:"type:char(36);primaryKey" json:"id"`
	DeviceID      string     `gorm:"type:char(36);not null;index" json:"device_id"`
	InstanceID    string     `gorm:"type:char(36);not null;index" json:"instance_id"`
	Source        string     `gorm:"type:varchar(16);not null" json:"source"`
	ProjectID     string     `gorm:"type:varchar(128);not null" json:"project_id"`
	ProjectName   string     `gorm:"type:varchar(256);not null" json:"project_name"`
	VersionID     string     `gorm:"type:varchar(128);not null" json:"version_id"`
	VersionNumber string     `gorm:"type:varchar(64)" json:"version_number"`
	Filename      string     `gorm:"type:varchar(256);not null" json:"filename"`
	DownloadURL   string     `gorm:"type:text;not null" json:"download_url"`
	ResourceType  string     `gorm:"type:varchar(32);not null;default:mod" json:"resource_type"`
	IconURL       string     `gorm:"type:text" json:"icon_url,omitempty"`
	Downloads     int64      `gorm:"type:bigint" json:"downloads,omitempty"`
	FileSize      int64      `gorm:"type:bigint" json:"file_size,omitempty"`
	Status        string     `gorm:"type:varchar(32);not null;default:queued;index" json:"status"`
	ErrorCode     *string    `gorm:"type:varchar(64);column:error_code" json:"error_code,omitempty"`
	ExpiresAt     time.Time  `json:"expires_at"`
	DispatchedAt  *time.Time `json:"dispatched_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (ModInstallRequest) TableName() string {
	return "mod_install_requests"
}
