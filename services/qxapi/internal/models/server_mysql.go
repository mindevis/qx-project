package models

import "time"

const (
	MySQLStatusNotInstalled = "not_installed"
	MySQLStatusInstalling   = "installing"
	MySQLStatusInstalled    = "installed"
	MySQLStatusStarting     = "starting"
	MySQLStatusRunning      = "running"
	MySQLStatusStopping     = "stopping"
	MySQLStatusError        = "error"

	MySQLEngineMariaDB = "mariadb"
	MySQLEnginePercona = "percona"
	MySQLMethodDocker  = "docker"
	MySQLMethodNative  = "native"
)

type ServerMySQL struct {
	ServerID     string    `gorm:"type:char(36);primaryKey" json:"server_id"`
	Status       string    `gorm:"type:varchar(32);not null;default:not_installed" json:"status"`
	Engine       string    `gorm:"type:varchar(32);not null;default:''" json:"engine"`
	Version      string    `gorm:"type:varchar(32);not null;default:''" json:"version"`
	Method       string    `gorm:"type:varchar(16);not null;default:''" json:"method"`
	BindAddr     string    `gorm:"type:varchar(64);not null;default:127.0.0.1" json:"bind_addr"`
	Port         int       `gorm:"not null;default:3306" json:"port"`
	Image        string    `gorm:"type:varchar(128);not null;default:''" json:"image"`
	RootPassword string    `gorm:"type:varchar(128);not null;default:''" json:"-"`
	LastError    string    `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (ServerMySQL) TableName() string { return "server_mysql" }

type ServerMySQLDatabase struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	ServerID  string    `gorm:"type:char(36);not null;index;uniqueIndex:ux_mysql_db_name" json:"server_id"`
	Name      string    `gorm:"type:varchar(64);not null;uniqueIndex:ux_mysql_db_name" json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func (ServerMySQLDatabase) TableName() string { return "server_mysql_database" }

type ServerMySQLUser struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	ServerID  string    `gorm:"type:char(36);not null;index;uniqueIndex:ux_mysql_user_host" json:"server_id"`
	Username  string    `gorm:"type:varchar(32);not null;uniqueIndex:ux_mysql_user_host" json:"username"`
	Host      string    `gorm:"type:varchar(255);not null;uniqueIndex:ux_mysql_user_host" json:"host"`
	Password  string    `gorm:"type:varchar(128);not null;default:''" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ServerMySQLUser) TableName() string { return "server_mysql_user" }

type ServerMySQLGrant struct {
	ID         string     `gorm:"type:char(36);primaryKey" json:"id"`
	UserID     string     `gorm:"type:char(36);not null;uniqueIndex:ux_mysql_grant" json:"user_id"`
	DatabaseID string     `gorm:"type:char(36);not null;uniqueIndex:ux_mysql_grant" json:"database_id"`
	Privileges StringList `gorm:"type:text" json:"privileges"`
}

func (ServerMySQLGrant) TableName() string { return "server_mysql_grant" }
