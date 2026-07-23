package entity

import "time"

// Journal represents a smart emotion & focus journal entry
type Journal struct {
	ID            uint      `json:"id"`
	UserID        uint      `json:"user_id"`
	Content       string    `json:"content"`
	DecisionScore float64   `json:"decision_score"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// UserConfig represents user preferences for the journal app
type UserConfig struct {
	ID            uint   `json:"id"`
	UserID        uint   `json:"user_id"`
	Theme         string `json:"theme"`
	Notifications bool   `json:"notifications"`
}

// LlmMetric represents LLM performance and cognitive telemetry data
type LlmMetric struct {
	ID         uint      `json:"id"`
	UserID     uint      `json:"user_id"`
	LatencyMs  int64     `json:"latency_ms"`
	TokenCount int       `json:"token_count"`
	ErrorLog   string    `json:"error_log"`
	CreatedAt  time.Time `json:"created_at"`
}
