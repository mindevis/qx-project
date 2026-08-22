package models

import "time"

const (
	GameServerNetworkRoleProxy   = "proxy"
	GameServerNetworkRoleLobby   = "lobby"
	GameServerNetworkRoleBackend = "backend"
)

type GameServerNetwork struct {
	ID               string    `gorm:"type:char(36);charset:utf8mb4;collation:utf8mb4_unicode_ci;primaryKey" json:"id"`
	ServerID         string    `gorm:"type:char(36);charset:utf8mb4;collation:utf8mb4_unicode_ci;not null;index" json:"server_id"`
	Name             string    `gorm:"type:varchar(128);not null" json:"name"`
	ForwardingSecret string    `gorm:"type:varchar(64);not null" json:"-"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (GameServerNetwork) TableName() string {
	return "game_server_networks"
}

type GameServerNetworkMember struct {
	ID           string    `gorm:"type:char(36);charset:utf8mb4;collation:utf8mb4_unicode_ci;primaryKey" json:"id"`
	NetworkID    string    `gorm:"type:char(36);charset:utf8mb4;collation:utf8mb4_unicode_ci;not null;index" json:"network_id"`
	GameServerID string    `gorm:"type:char(36);charset:utf8mb4;collation:utf8mb4_unicode_ci;not null;uniqueIndex" json:"game_server_id"`
	Role         string    `gorm:"type:varchar(32);not null" json:"role"`
	Alias        string    `gorm:"type:varchar(64);not null" json:"alias"`
	SortOrder    int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (GameServerNetworkMember) TableName() string {
	return "game_server_network_members"
}
