package models

import "time"

type Journal struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"index;not null" json:"user_id"`
	Content       string    `gorm:"type:text;not null" json:"content"`
	DecisionScore float64   `gorm:"type:numeric;default:0" json:"decision_score"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
