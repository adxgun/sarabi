package database

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sarabi/internal/types"
)

type logsRepository struct {
	db *gorm.DB
}

func NewLogsRepository(db *gorm.DB) LogsRepository {
	return &logsRepository{db: db}
}

func (l logsRepository) Save(ctx context.Context, log *types.Log) error {
	return l.db.Save(log).Error
}

func (l logsRepository) FindAll(ctx context.Context, applicationID uuid.UUID, filter types.Filter) ([]*types.Log, error) {
	var values []*types.Log
	query := l.db.WithContext(ctx).
		Where("application_id = ?", applicationID)
	if filter.Environment != "" {
		query = query.Where("environment = ?", filter.Environment)
	}

	if err := query.Order("timestamp ASC").Find(&values).Error; err != nil {
		return nil, err
	}
	return values, nil
}
