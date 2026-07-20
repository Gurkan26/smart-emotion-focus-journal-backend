package models

import "time"

type User struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Email        string     `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"not null" json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	UserConfig   UserConfig `json:"config,omitempty"`
}

type UserConfig struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	UserID        uint   `gorm:"uniqueIndex;not null" json:"user_id"`
	Theme         string `gorm:"default:'dark'" json:"theme"`
	Notifications bool   `gorm:"default:true" json:"notifications"`
}
