package repository

import (
	"context"
	"time"

	"github.com/edgelink/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditLogRepository 审计日志仓储接口（只读仓储，不可变）
type AuditLogRepository interface {
	// Create 创建新审计日志（唯一的写操作）
	Create(ctx context.Context, log *domain.AuditLog) error

	// CreateBatch 批量创建审计日志
	CreateBatch(ctx context.Context, logs []*domain.AuditLog) error

	// FindByID 根据ID查找审计日志
	FindByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error)

	// FindByFilters 根据过滤条件查找审计日志
	FindByFilters(ctx context.Context, filters *AuditLogFilters) ([]*domain.AuditLog, int64, error)

	// FindByOrganizationID 查找组织的所有审计日志
	FindByOrganizationID(ctx context.Context, organizationID uuid.UUID, limit int) ([]*domain.AuditLog, error)

	// FindByActorID 查找特定操作者的所有审计日志
	FindByActorID(ctx context.Context, actorID uuid.UUID, limit int) ([]*domain.AuditLog, error)

	// FindByResourceID 查找特定资源的所有审计日志
	FindByResourceID(ctx context.Context, resourceID uuid.UUID, limit int) ([]*domain.AuditLog, error)

	// FindByAction 查找特定操作的所有审计日志
	FindByAction(ctx context.Context, action string, limit int) ([]*domain.AuditLog, error)

	// GetAuditStats 获取审计统计信息
	GetAuditStats(ctx context.Context, organizationID uuid.UUID, startTime, endTime time.Time) (*AuditStats, error)
}

// AuditLogFilters 审计日志查询过滤条件
type AuditLogFilters struct {
	OrganizationID *uuid.UUID
	ActorID        *uuid.UUID
	Action         *string
	ResourceType   *domain.ResourceType
	ResourceID     *uuid.UUID
	IPAddress      *string
	StartTime      *time.Time
	EndTime        *time.Time
	Limit          int
	Offset         int
}

// AuditStats 审计统计信息
type AuditStats struct {
	TotalLogs          int64            `json:"total_logs"`
	UniqueActors       int64            `json:"unique_actors"`
	ActionsByType      map[string]int64 `json:"actions_by_type"`
	ResourcesByType    map[string]int64 `json:"resources_by_type"`
	ActivityByHour     map[int]int64    `json:"activity_by_hour"`
}

// auditLogRepository AuditLog仓储的GORM实现
type auditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository 创建AuditLog仓储实例
func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

// Create 创建新审计日志
func (r *auditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	// 审计日志是不可变的，只能插入
	return r.db.WithContext(ctx).Create(log).Error
}

// CreateBatch 批量创建审计日志
func (r *auditLogRepository) CreateBatch(ctx context.Context, logs []*domain.AuditLog) error {
	return r.db.WithContext(ctx).Create(logs).Error
}

// FindByID 根据ID查找审计日志
func (r *auditLogRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.AuditLog, error) {
	var log domain.AuditLog
	err := r.db.WithContext(ctx).
		Preload("Organization").
		Where("id = ?", id).
		First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// FindByFilters 根据过滤条件查找审计日志
func (r *auditLogRepository) FindByFilters(ctx context.Context, filters *AuditLogFilters) ([]*domain.AuditLog, int64, error) {
	var logs []*domain.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.AuditLog{}).Preload("Organization")

	// 应用过滤条件
	if filters.OrganizationID != nil {
		query = query.Where("organization_id = ?", *filters.OrganizationID)
	}
	if filters.ActorID != nil {
		query = query.Where("actor_id = ?", *filters.ActorID)
	}
	if filters.Action != nil {
		query = query.Where("action = ?", *filters.Action)
	}
	if filters.ResourceType != nil {
		query = query.Where("resource_type = ?", *filters.ResourceType)
	}
	if filters.ResourceID != nil {
		query = query.Where("resource_id = ?", *filters.ResourceID)
	}
	if filters.IPAddress != nil {
		query = query.Where("ip_address = ?", *filters.IPAddress)
	}
	if filters.StartTime != nil {
		query = query.Where("created_at >= ?", *filters.StartTime)
	}
	if filters.EndTime != nil {
		query = query.Where("created_at <= ?", *filters.EndTime)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 应用分页和排序（按时间倒序）
	query = query.Order("created_at DESC")
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Find(&logs).Error
	return logs, total, err
}

// FindByOrganizationID 查找组织的所有审计日志
func (r *auditLogRepository) FindByOrganizationID(ctx context.Context, organizationID uuid.UUID, limit int) ([]*domain.AuditLog, error) {
	var logs []*domain.AuditLog
	query := r.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&logs).Error
	return logs, err
}

// FindByActorID 查找特定操作者的所有审计日志
func (r *auditLogRepository) FindByActorID(ctx context.Context, actorID uuid.UUID, limit int) ([]*domain.AuditLog, error) {
	var logs []*domain.AuditLog
	query := r.db.WithContext(ctx).
		Where("actor_id = ?", actorID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&logs).Error
	return logs, err
}

// FindByResourceID 查找特定资源的所有审计日志
func (r *auditLogRepository) FindByResourceID(ctx context.Context, resourceID uuid.UUID, limit int) ([]*domain.AuditLog, error) {
	var logs []*domain.AuditLog
	query := r.db.WithContext(ctx).
		Where("resource_id = ?", resourceID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&logs).Error
	return logs, err
}

// FindByAction 查找特定操作的所有审计日志
func (r *auditLogRepository) FindByAction(ctx context.Context, action string, limit int) ([]*domain.AuditLog, error) {
	var logs []*domain.AuditLog
	query := r.db.WithContext(ctx).
		Preload("Organization").
		Where("action = ?", action).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&logs).Error
	return logs, err
}

// GetAuditStats 获取审计统计信息
func (r *auditLogRepository) GetAuditStats(ctx context.Context, organizationID uuid.UUID, startTime, endTime time.Time) (*AuditStats, error) {
	var stats AuditStats
	stats.ActionsByType = make(map[string]int64)
	stats.ResourcesByType = make(map[string]int64)
	stats.ActivityByHour = make(map[int]int64)

	baseQuery := r.db.WithContext(ctx).Model(&domain.AuditLog{}).
		Where("organization_id = ? AND created_at BETWEEN ? AND ?", organizationID, startTime, endTime)

	// 总日志数
	if err := baseQuery.Count(&stats.TotalLogs).Error; err != nil {
		return nil, err
	}

	// 唯一操作者数量
	if err := r.db.WithContext(ctx).Model(&domain.AuditLog{}).
		Where("organization_id = ? AND created_at BETWEEN ? AND ? AND actor_id IS NOT NULL", organizationID, startTime, endTime).
		Distinct("actor_id").
		Count(&stats.UniqueActors).Error; err != nil {
		return nil, err
	}

	// 按操作类型分组
	var actionCounts []struct {
		Action string
		Count  int64
	}
	if err := r.db.WithContext(ctx).Model(&domain.AuditLog{}).
		Select("action, COUNT(*) as count").
		Where("organization_id = ? AND created_at BETWEEN ? AND ?", organizationID, startTime, endTime).
		Group("action").
		Scan(&actionCounts).Error; err != nil {
		return nil, err
	}
	for _, ac := range actionCounts {
		stats.ActionsByType[ac.Action] = ac.Count
	}

	// 按资源类型分组
	var resourceCounts []struct {
		ResourceType string
		Count        int64
	}
	if err := r.db.WithContext(ctx).Model(&domain.AuditLog{}).
		Select("resource_type, COUNT(*) as count").
		Where("organization_id = ? AND created_at BETWEEN ? AND ?", organizationID, startTime, endTime).
		Group("resource_type").
		Scan(&resourceCounts).Error; err != nil {
		return nil, err
	}
	for _, rc := range resourceCounts {
		stats.ResourcesByType[rc.ResourceType] = rc.Count
	}

	// 按小时分组活动
	var hourCounts []struct {
		Hour  int
		Count int64
	}
	if err := r.db.WithContext(ctx).Model(&domain.AuditLog{}).
		Select("EXTRACT(HOUR FROM created_at) as hour, COUNT(*) as count").
		Where("organization_id = ? AND created_at BETWEEN ? AND ?", organizationID, startTime, endTime).
		Group("EXTRACT(HOUR FROM created_at)").
		Scan(&hourCounts).Error; err != nil {
		return nil, err
	}
	for _, hc := range hourCounts {
		stats.ActivityByHour[hc.Hour] = hc.Count
	}

	return &stats, nil
}
