package models

import "time"

const (
	OllamaStatusNotInstalled = "not_installed"
	OllamaStatusInstalling   = "installing"
	OllamaStatusInstalled    = "installed"
	OllamaStatusStarting     = "starting"
	OllamaStatusRunning      = "running"
	OllamaStatusStopping     = "stopping"
	OllamaStatusPulling      = "pulling"
	OllamaStatusError        = "error"
)

type ServerOllama struct {
	ServerID     string    `gorm:"type:char(36);primaryKey" json:"server_id"`
	Status       string    `gorm:"type:varchar(32);not null;default:not_installed" json:"status"`
	Version      string    `gorm:"type:varchar(64);not null;default:''" json:"version"`
	BinPath      string    `gorm:"type:varchar(512);not null;default:''" json:"bin_path"`
	RootDir      string    `gorm:"type:varchar(512);not null;default:''" json:"root_dir"`
	ModelsDir    string    `gorm:"type:varchar(512);not null;default:''" json:"models_dir"`
	ListenAddr   string    `gorm:"type:varchar(128);not null;default:127.0.0.1:11434" json:"listen_addr"`
	PullingModel string    `gorm:"type:varchar(256);not null;default:''" json:"pulling_model"`
	LastError    string    `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (ServerOllama) TableName() string {
	return "server_ollama"
}
