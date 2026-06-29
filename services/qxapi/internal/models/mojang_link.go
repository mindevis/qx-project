package models

import "time"

type MojangLink struct {
	UserID          string    `gorm:"type:char(36);primaryKey" json:"user_id"`
	MinecraftUUID   string    `gorm:"type:char(36);not null" json:"minecraft_uuid"`
	Username        string    `gorm:"type:varchar(16);not null" json:"username"`
	AccessTokenEnc  []byte    `gorm:"type:blob" json:"-"`
	RefreshTokenEnc []byte    `gorm:"type:blob;not null" json:"-"`
	LinkedAt        time.Time `json:"linked_at"`
}

func (MojangLink) TableName() string {
	return "mojang_links"
}
