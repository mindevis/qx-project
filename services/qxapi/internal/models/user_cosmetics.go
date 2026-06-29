package models

import "time"

const (
	CosmeticsSkinModelSteve = "steve"
	CosmeticsSkinModelAlex  = "alex"

	CosmeticsCapeNone   = "none"
	CosmeticsCapeQX       = "qx"
	CosmeticsCapeCustom   = "custom"

	CosmeticsWingsNone   = "none"
	CosmeticsWingsAngel  = "angel"
	CosmeticsWingsDemon  = "demon"
	CosmeticsWingsDragon = "dragon"
)

type UserCosmetics struct {
	UserID    string    `gorm:"type:char(36);primaryKey" json:"user_id"`
	SkinModel string    `gorm:"type:varchar(16);not null;default:steve" json:"skin_model"`
	HasSkin   bool      `gorm:"not null;default:false" json:"has_skin"`
	CapeType  string    `gorm:"type:varchar(32);not null;default:none" json:"cape_type"`
	HasCape   bool      `gorm:"not null;default:false" json:"has_cape"`
	WingsType string    `gorm:"type:varchar(32);not null;default:none" json:"wings_type"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UserCosmetics) TableName() string {
	return "user_cosmetics"
}
