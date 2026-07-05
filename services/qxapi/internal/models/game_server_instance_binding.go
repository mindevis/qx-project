package models

import "time"

type GameServerInstanceBinding struct {
	ID                 string `gorm:"type:char(36);charset:utf8mb4;collation:utf8mb4_unicode_ci;primaryKey" json:"id"`
	UserID             string `gorm:"type:char(36);charset:utf8mb4;collation:utf8mb4_unicode_ci;not null;uniqueIndex:idx_binding_user_game" json:"user_id"`
	GameServerID       string `gorm:"type:char(36);charset:utf8mb4;collation:utf8mb4_unicode_ci;not null;uniqueIndex:idx_binding_user_game;index" json:"game_server_id"`
	LauncherInstanceID string `gorm:"type:char(36);charset:utf8mb4;collation:utf8mb4_unicode_ci;not null;index" json:"launcher_instance_id"`
	// mediumtext (16 MB): enabled-filename lists for a large modpack can exceed
	// MySQL's 64 KB TEXT limit.
	ClientModEnabled          StringList `gorm:"type:mediumtext" json:"client_mod_enabled,omitempty"`
	ClientResourcepackEnabled StringList `gorm:"type:mediumtext" json:"client_resourcepack_enabled,omitempty"`
	ClientShaderEnabled       StringList `gorm:"type:mediumtext" json:"client_shader_enabled,omitempty"`
	CreatedAt                 time.Time  `json:"created_at"`
	UpdatedAt                 time.Time  `json:"updated_at"`
}

func (GameServerInstanceBinding) TableName() string {
	return "game_server_instance_bindings"
}
