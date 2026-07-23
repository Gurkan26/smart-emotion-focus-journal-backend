package repository

import (
	"context"

	"github.com/gurkanfikretgunak/masterfabric-go/internal/domain/journal/entity"
)

type JournalRepository interface {
	Create(ctx context.Context, j *entity.Journal) error
	GetByUserID(ctx context.Context, userID uint) ([]entity.Journal, error)
	GetAvgScore(ctx context.Context, userID uint) (float64, error)
}

type ConfigRepository interface {
	GetByUserID(ctx context.Context, userID uint) (*entity.UserConfig, error)
	Save(ctx context.Context, cfg *entity.UserConfig) error
}

type MetricRepository interface {
	Create(ctx context.Context, m *entity.LlmMetric) error
	GetByUserID(ctx context.Context, userID uint, limit int) ([]entity.LlmMetric, error)
	ClearByUserID(ctx context.Context, userID uint) error
}
