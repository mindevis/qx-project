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
	ID             string               `gorm:"type:char(36);primaryKey" json:"id"`
	UserID         *string              `gorm:"type:char(36);index" json:"user_id,omitempty"`
	GuestSessionID *string              `gorm:"type:char(36);index" json:"guest_session_id,omitempty"`
	Name           string               `gorm:"type:varchar(128);not null" json:"name"`
	MCVersion      string               `gorm:"type:varchar(32);not null" json:"mc_version"`
	Loader         string               `gorm:"type:varchar(32);not null;default:vanilla" json:"loader"`
	LoaderVersion  *string              `gorm:"type:varchar(32)" json:"loader_version,omitempty"`
	MaxMemoryMB    *int                 `gorm:"type:int" json:"max_memory_mb,omitempty"`
	MinMemoryMB    *int                 `gorm:"type:int" json:"min_memory_mb,omitempty"`
	ExtraJVMArgs   StringList           `gorm:"type:text" json:"extra_jvm_args,omitempty"`
	WindowWidth    *int                 `gorm:"type:int" json:"window_width,omitempty"`
	WindowHeight   *int                 `gorm:"type:int" json:"window_height,omitempty"`
	Mods           InstanceResourceList `gorm:"type:text" json:"mods"`
	ResourcePacks  InstanceResourceList `gorm:"type:text" json:"resource_packs"`
	Shaders        InstanceResourceList `gorm:"type:text" json:"shaders"`
	Datapacks      InstanceResourceList `gorm:"type:text" json:"datapacks"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

func (LauncherInstance) TableName() string {
	return "launcher_instances"
}
