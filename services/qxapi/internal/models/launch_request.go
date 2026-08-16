package models

import "time"

const (
	LaunchStatusQueued      = "queued"
	LaunchStatusDispatched  = "dispatched"
	LaunchStatusPreparing   = "preparing"
	LaunchStatusDownloading = "downloading"
	LaunchStatusLaunching   = "launching"
	LaunchStatusRunning     = "running"
	LaunchStatusCompleted   = "completed"
	LaunchStatusFailed      = "failed"
	LaunchStatusExpired     = "expired"
)

type LaunchRequest struct {
	ID                string     `gorm:"type:char(36);primaryKey" json:"id"`
	DeviceID          string     `gorm:"type:char(36);not null;index" json:"device_id"`
	InstanceID        string     `gorm:"type:char(36);not null;index" json:"instance_id"`
	OfflineProfileID  *string    `gorm:"type:char(36);index" json:"offline_profile_id,omitempty"`
	UseMojangAccount  bool       `gorm:"not null;default:false" json:"use_mojang_account"`
	JoinServerAddress *string    `gorm:"type:varchar(255)" json:"join_server_address,omitempty"`
	JoinServerPort    *int       `gorm:"column:join_server_port" json:"join_server_port,omitempty"`
	Status            string     `gorm:"type:varchar(32);not null;default:queued;index" json:"status"`
	PID               *int       `gorm:"column:pid" json:"pid,omitempty"`
	ExitCode          *int       `gorm:"column:exit_code" json:"exit_code,omitempty"`
	ErrorCode         *string    `gorm:"type:varchar(64);column:error_code" json:"error_code,omitempty"`
	ProgressMessage   string     `gorm:"type:varchar(256)" json:"progress_message,omitempty"`
	ExpiresAt         time.Time  `json:"expires_at"`
	DispatchedAt      *time.Time `json:"dispatched_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

func (LaunchRequest) TableName() string {
	return "launch_requests"
}
