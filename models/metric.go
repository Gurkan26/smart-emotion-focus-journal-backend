package models

import "time"

type LlmMetric struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"user_id"`
	LatencyMs  int64     `json:"latency_ms"`
	TokenCount int       `json:"token_count"`
	ErrorLog   string    `gorm:"type:text" json:"error_log"`
	CreatedAt  time.Time `json:"created_at"`
}
