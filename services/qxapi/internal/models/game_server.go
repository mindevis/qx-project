package models

import "time"

const (
	GameServerStatusInstalling = "installing"
	GameServerStatusStarting   = "starting"
	GameServerStatusRunning    = "running"
	GameServerStatusStopped    = "stopped"
	GameServerStatusError      = "error"
)

type GameServer struct {
	ID            string    `gorm:"type:char(36);charset:utf8mb4;collation:utf8mb4_unicode_ci;primaryKey" json:"id"`
	ServerID      string    `gorm:"type:char(36);charset:utf8mb4;collation:utf8mb4_unicode_ci;not null;index" json:"server_id"`
	Name          string    `gorm:"type:varchar(128);not null" json:"name"`
	ServerType    string    `gorm:"type:varchar(32);not null" json:"server_type"`
	MCVersion     string    `gorm:"type:varchar(32);not null" json:"mc_version"`
	LoaderVersion *string   `gorm:"type:varchar(64)" json:"loader_version,omitempty"`
	Address       *string   `gorm:"type:varchar(255)" json:"address,omitempty"`
	Port          int       `gorm:"not null;default:25565" json:"port"`
	RconPassword  *string   `gorm:"type:varchar(64)" json:"-"`
	Status        string    `gorm:"type:varchar(32);not null;default:installing" json:"status"`
	WorkDir       string    `gorm:"type:varchar(512)" json:"work_dir,omitempty"`
	StartCommand  string    `gorm:"type:varchar(255)" json:"-"`
	StartArgsJSON string    `gorm:"type:text" json:"-"`
	JarPath       string    `gorm:"type:varchar(512)" json:"jar_path,omitempty"`

	ShowInMonitoring      bool    `gorm:"not null;default:false;index" json:"show_in_monitoring"`
	MonitoringDescription string  `gorm:"type:text" json:"monitoring_description,omitempty"`
	BannerURL             string  `gorm:"type:varchar(512)" json:"banner_url,omitempty"`
	MonitoringTagsJSON    string  `gorm:"type:text" json:"-"`
	MonitoringModsJSON    string  `gorm:"type:text" json:"-"`
	MonitoringPluginsJSON string  `gorm:"type:text" json:"-"`
	LikesCount            int     `gorm:"not null;default:0" json:"likes_count"`
	RatingSum             int     `gorm:"not null;default:0" json:"-"`
	RatingCount           int     `gorm:"not null;default:0" json:"rating_count"`
	LastError             string  `gorm:"type:text" json:"last_error,omitempty"`

	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (GameServer) TableName() string {
	return "game_servers"
}
