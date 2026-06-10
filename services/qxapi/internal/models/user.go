package models

import "time"

type User struct {
	ID           string    `gorm:"type:char(36);primaryKey" json:"id"`
	Email        string    `gorm:"type:varchar(255);not null;unique" json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null" json:"-"`
	Username     *string   `gorm:"type:varchar(32)" json:"username,omitempty"`
	Tier         string    `gorm:"type:varchar(16);not null;default:free" json:"tier"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string {
	return "users"
}
