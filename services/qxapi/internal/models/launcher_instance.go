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
	Mods           InstanceResourceList `gorm:"type:text" json:"mods"`
	ResourcePacks  InstanceResourceList `gorm:"type:text" json:"resource_packs"`
	Shaders        InstanceResourceList `gorm:"type:text" json:"shaders"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

func (LauncherInstance) TableName() string {
	return "launcher_instances"
}
