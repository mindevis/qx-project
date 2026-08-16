package models

import "time"

const (
	LoaderVanilla  = "vanilla"
	LoaderForge    = "forge"
	LoaderNeoForge = "neoforge"
	LoaderFabric   = "fabric"
	LoaderQuilt    = "quilt"
)

type LauncherInstance struct {
	ID             string     `gorm:"type:char(36);primaryKey" json:"id"`
	UserID         *string    `gorm:"type:char(36);index" json:"user_id,omitempty"`
	GuestSessionID *string    `gorm:"type:char(36);index" json:"guest_session_id,omitempty"`
	Name           string     `gorm:"type:varchar(128);not null" json:"name"`
	MCVersion      string     `gorm:"type:varchar(32);not null" json:"mc_version"`
	Loader         string     `gorm:"type:varchar(32);not null;default:vanilla" json:"loader"`
	LoaderVersion  *string    `gorm:"type:varchar(32)" json:"loader_version,omitempty"`
	MaxMemoryMB    *int       `gorm:"type:int" json:"max_memory_mb,omitempty"`
	MinMemoryMB    *int       `gorm:"type:int" json:"min_memory_mb,omitempty"`
	ExtraJVMArgs   StringList `gorm:"type:text" json:"extra_jvm_args,omitempty"`
	WindowWidth    *int       `gorm:"type:int" json:"window_width,omitempty"`
	WindowHeight   *int       `gorm:"type:int" json:"window_height,omitempty"`
	// Set when the instance is created for a specific game server on connect.
	// Content (mods/resource packs/shaders/datapacks) may only change via that
	// server's catalog and the connect-time sync — not from the launcher UI.
	ManagedByGameServerID *string `gorm:"type:char(36);column:managed_by_game_server_id;-:migration" json:"managed_by_game_server_id,omitempty"`
	// mediumtext (16 MB), not text (64 KB): a large modpack's worth of mod
	// metadata (names, filenames, icon URLs, …) easily exceeds MySQL's 64 KB
	// TEXT limit, which fails the write and blocks any further install.
	Mods          InstanceResourceList `gorm:"type:mediumtext" json:"mods"`
	ResourcePacks InstanceResourceList `gorm:"type:mediumtext" json:"resource_packs"`
	Shaders       InstanceResourceList `gorm:"type:mediumtext" json:"shaders"`
	Datapacks     InstanceResourceList `gorm:"type:mediumtext" json:"datapacks"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

func (LauncherInstance) TableName() string {
	return "launcher_instances"
}
