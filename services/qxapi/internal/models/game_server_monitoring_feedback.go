package models

import "time"

type GameServerMonitoringFeedback struct {
	ID           string    `gorm:"type:char(36);charset:utf8mb4;collation:utf8mb4_unicode_ci;primaryKey" json:"id"`
	UserID       string    `gorm:"type:char(36);charset:utf8mb4;collation:utf8mb4_unicode_ci;not null;uniqueIndex:idx_monitoring_feedback_user_game" json:"user_id"`
	GameServerID string    `gorm:"type:char(36);charset:utf8mb4;collation:utf8mb4_unicode_ci;not null;uniqueIndex:idx_monitoring_feedback_user_game;index" json:"game_server_id"`
	Liked        bool      `gorm:"not null;default:false" json:"liked"`
	Rating       *int      `gorm:"type:tinyint" json:"rating,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (GameServerMonitoringFeedback) TableName() string {
	return "game_server_monitoring_feedback"
}
