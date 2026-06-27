package models

import "time"

const (
	ServerTypeVanilla = "vanilla"

	ServerStatusPending   = "pending"
	ServerStatusDeploying = "deploying"
	ServerStatusOffline   = "offline"
	ServerStatusStarting  = "starting"
	ServerStatusOnline    = "online"
	ServerStatusStopping  = "stopping"
	ServerStatusError     = "error"
)

type Server struct {
	ID             string     `gorm:"type:char(36);primaryKey" json:"id"`
	OwnerID        string     `gorm:"type:char(36);not null;uniqueIndex:idx_servers_owner_slug,priority:1" json:"owner_id"`
	Name           string     `gorm:"type:varchar(128);not null" json:"name"`
	Slug           string     `gorm:"type:varchar(64);not null;uniqueIndex:idx_servers_owner_slug,priority:2" json:"slug"`
	ServerType     string     `gorm:"type:varchar(32);not null;default:vanilla" json:"server_type"`
	Status         string     `gorm:"type:varchar(32);not null;default:pending" json:"status"`
	MCVersion      *string    `gorm:"type:varchar(32)" json:"mc_version,omitempty"`
	ConfigJSON     string     `gorm:"type:text;not null" json:"-"`
	AgentTokenHash *string    `gorm:"type:varchar(255)" json:"-"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (Server) TableName() string {
	return "servers"
}

type SSHCredential struct {
	ServerID       string    `gorm:"type:char(36);primaryKey" json:"-"`
	Host           string    `gorm:"type:varchar(255);not null" json:"host"`
	Port           int       `gorm:"not null;default:22" json:"port"`
	Username       string    `gorm:"type:varchar(64);not null" json:"username"`
	PrivateKeyEnc  []byte    `gorm:"type:blob;not null" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (SSHCredential) TableName() string {
	return "ssh_credentials"
}

type Agent struct {
	ID            string     `gorm:"type:char(36);primaryKey" json:"id"`
	ServerID      string     `gorm:"type:char(36);not null;uniqueIndex" json:"server_id"`
	Hostname      *string    `gorm:"type:varchar(255)" json:"hostname,omitempty"`
	OS            string     `gorm:"type:varchar(64);not null;default:linux" json:"os"`
	AgentVersion  *string    `gorm:"type:varchar(32)" json:"agent_version,omitempty"`
	ConnectedAt   *time.Time `json:"connected_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (Agent) TableName() string {
	return "agents"
}
