package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EmailHistory 邮件发送历史记录
type EmailHistory struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	AlertID    *uuid.UUID `gorm:"type:uuid;index"`
	Provider   string    `gorm:"type:varchar(50);not null"`
	Recipients []string  `gorm:"type:text[];not null"`
	Subject    string    `gorm:"type:text;not null"`
	Status     string    `gorm:"type:varchar(20);not null;index"` // queued, sent, failed, retrying
	Attempts   int       `gorm:"default:1"`
	LastError  *string   `gorm:"type:text"`
	SentAt     *time.Time
	CreatedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_created_at,sort:desc"`
	UpdatedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

// TableName 指定表名
func (EmailHistory) TableName() string {
	return "email_history"
}

// EmailHistoryRepository 邮件历史仓储接口
type EmailHistoryRepository interface {
	Create(ctx context.Context, history *EmailHistory) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, sentAt *time.Time, lastError *string) error
	IncrementAttempts(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*EmailHistory, error)
	GetByAlertID(ctx context.Context, alertID uuid.UUID) ([]*EmailHistory, error)
	List(ctx context.Context, filter EmailHistoryFilter) ([]*EmailHistory, int64, error)
	DeleteOlderThan(ctx context.Context, days int) (int64, error)
}

// EmailHistoryFilter 邮件历史查询过滤器
type EmailHistoryFilter struct {
	AlertID   *uuid.UUID
	Provider  *string
	Status    *string
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
	Offset    int
}

// emailHistoryRepository 邮件历史仓储实现
type emailHistoryRepository struct {
	db *gorm.DB
}

// NewEmailHistoryRepository 创建邮件历史仓储
func NewEmailHistoryRepository(db *gorm.DB) EmailHistoryRepository {
	return &emailHistoryRepository{db: db}
}

// Create 创建邮件历史记录
func (r *emailHistoryRepository) Create(ctx context.Context, history *EmailHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

// UpdateStatus 更新发送状态
func (r *emailHistoryRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status string,
	sentAt *time.Time,
	lastError *string,
) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if sentAt != nil {
		updates["sent_at"] = sentAt
	}

	if lastError != nil {
		updates["last_error"] = lastError
	}

	return r.db.WithContext(ctx).
		Model(&EmailHistory{}).
		Where("id = ?", id).
		Updates(updates).
		Error
}

// IncrementAttempts 增加尝试次数
func (r *emailHistoryRepository) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&EmailHistory{}).
		Where("id = ?", id).
		UpdateColumn("attempts", gorm.Expr("attempts + ?", 1)).
		Error
}

// GetByID 根据ID获取邮件历史
func (r *emailHistoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*EmailHistory, error) {
	var history EmailHistory
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&history).Error; err != nil {
		return nil, err
	}
	return &history, nil
}

// GetByAlertID 根据告警ID获取邮件历史
func (r *emailHistoryRepository) GetByAlertID(ctx context.Context, alertID uuid.UUID) ([]*EmailHistory, error) {
	var histories []*EmailHistory
	if err := r.db.WithContext(ctx).
		Where("alert_id = ?", alertID).
		Order("created_at DESC").
		Find(&histories).Error; err != nil {
		return nil, err
	}
	return histories, nil
}

// List 列表查询
func (r *emailHistoryRepository) List(
	ctx context.Context,
	filter EmailHistoryFilter,
) ([]*EmailHistory, int64, error) {
	query := r.db.WithContext(ctx).Model(&EmailHistory{})

	// 应用过滤条件
	if filter.AlertID != nil {
		query = query.Where("alert_id = ?", *filter.AlertID)
	}
	if filter.Provider != nil {
		query = query.Where("provider = ?", *filter.Provider)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.StartDate != nil {
		query = query.Where("created_at >= ?", *filter.StartDate)
	}
	if filter.EndDate != nil {
		query = query.Where("created_at <= ?", *filter.EndDate)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var histories []*EmailHistory
	if filter.Limit <= 0 {
		filter.Limit = 50
	}

	if err := query.
		Order("created_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&histories).Error; err != nil {
		return nil, 0, err
	}

	return histories, total, nil
}

// DeleteOlderThan 删除指定天数之前的记录
func (r *emailHistoryRepository) DeleteOlderThan(ctx context.Context, days int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	result := r.db.WithContext(ctx).
		Where("created_at < ?", cutoffDate).
		Delete(&EmailHistory{})

	return result.RowsAffected, result.Error
}
