package repository

import (
	"context"
	"time"

	"github.com/edgelink/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AlertRepository 告警仓储接口
type AlertRepository interface {
	// Create 创建新告警
	Create(ctx context.Context, alert *domain.Alert) error

	// FindByID 根据ID查找告警
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Alert, error)

	// FindByFilters 根据过滤条件查找告警
	FindByFilters(ctx context.Context, filters *AlertFilters) ([]*domain.Alert, int64, error)

	// FindActiveAlerts 查找所有活跃告警
	FindActiveAlerts(ctx context.Context, limit int) ([]*domain.Alert, error)

	// FindByDeviceID 查找设备的所有告警
	FindByDeviceID(ctx context.Context, deviceID uuid.UUID, limit int) ([]*domain.Alert, error)

	// FindBySeverity 根据严重程度查找告警
	FindBySeverity(ctx context.Context, severity domain.Severity, activeOnly bool, limit int) ([]*domain.Alert, error)

	// Update 更新告警
	Update(ctx context.Context, alert *domain.Alert) error

	// Acknowledge 确认告警
	Acknowledge(ctx context.Context, id uuid.UUID, acknowledgedBy uuid.UUID) error

	// Resolve 解决告警
	Resolve(ctx context.Context, id uuid.UUID) error

	// UpdateOccurrence 更新告警出现次数和最后出现时间
	UpdateOccurrence(ctx context.Context, id uuid.UUID, occurrenceCount int) error

	// EscalateSeverity 提升告警严重程度
	EscalateSeverity(ctx context.Context, id uuid.UUID, newSeverity domain.Severity) error

	// FindActiveByDeviceAndType 查找设备的特定类型的活跃告警
	FindActiveByDeviceAndType(ctx context.Context, deviceID uuid.UUID, alertType domain.AlertType) (*domain.Alert, error)

	// ResolveByDeviceAndType 解决设备的特定类型告警
	ResolveByDeviceAndType(ctx context.Context, deviceID uuid.UUID, alertType domain.AlertType) error

	// GetAlertStats 获取告警统计
	GetAlertStats(ctx context.Context, startTime, endTime time.Time) (*AlertStats, error)
}

// AlertFilters 告警查询过滤条件
type AlertFilters struct {
	DeviceID   *uuid.UUID
	Severity   *domain.Severity
	AlertType  *domain.AlertType
	Status     *domain.AlertStatus
	StartTime  *time.Time
	EndTime    *time.Time
	Limit      int
	Offset     int
}

// AlertStats 告警统计信息
type AlertStats struct {
	TotalAlerts         int64            `json:"total_alerts"`
	ActiveAlerts        int64            `json:"active_alerts"`
	AcknowledgedAlerts  int64            `json:"acknowledged_alerts"`
	ResolvedAlerts      int64            `json:"resolved_alerts"`
	CriticalAlerts      int64            `json:"critical_alerts"`
	AlertsBySeverity    map[string]int64 `json:"alerts_by_severity"`
	AlertsByType        map[string]int64 `json:"alerts_by_type"`
}

// alertRepository Alert仓储的GORM实现
type alertRepository struct {
	db *gorm.DB
}

// NewAlertRepository 创建Alert仓储实例
func NewAlertRepository(db *gorm.DB) AlertRepository {
	return &alertRepository{db: db}
}

// Create 创建新告警
func (r *alertRepository) Create(ctx context.Context, alert *domain.Alert) error {
	return r.db.WithContext(ctx).Create(alert).Error
}

// FindByID 根据ID查找告警
func (r *alertRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Alert, error) {
	var alert domain.Alert
	err := r.db.WithContext(ctx).
		Preload("Device").
		Where("id = ?", id).
		First(&alert).Error
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

// FindByFilters 根据过滤条件查找告警
func (r *alertRepository) FindByFilters(ctx context.Context, filters *AlertFilters) ([]*domain.Alert, int64, error) {
	var alerts []*domain.Alert
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Alert{}).Preload("Device")

	// 应用过滤条件
	if filters.DeviceID != nil {
		query = query.Where("device_id = ?", *filters.DeviceID)
	}
	if filters.Severity != nil {
		query = query.Where("severity = ?", *filters.Severity)
	}
	if filters.AlertType != nil {
		query = query.Where("type = ?", *filters.AlertType)
	}
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
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

	// 应用分页和排序
	query = query.Order("created_at DESC")
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Find(&alerts).Error
	return alerts, total, err
}

// FindActiveAlerts 查找所有活跃告警
func (r *alertRepository) FindActiveAlerts(ctx context.Context, limit int) ([]*domain.Alert, error) {
	var alerts []*domain.Alert
	query := r.db.WithContext(ctx).
		Preload("Device").
		Where("status = ?", domain.AlertStatusActive).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&alerts).Error
	return alerts, err
}

// FindByDeviceID 查找设备的所有告警
func (r *alertRepository) FindByDeviceID(ctx context.Context, deviceID uuid.UUID, limit int) ([]*domain.Alert, error) {
	var alerts []*domain.Alert
	query := r.db.WithContext(ctx).
		Where("device_id = ?", deviceID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&alerts).Error
	return alerts, err
}

// FindBySeverity 根据严重程度查找告警
func (r *alertRepository) FindBySeverity(ctx context.Context, severity domain.Severity, activeOnly bool, limit int) ([]*domain.Alert, error) {
	var alerts []*domain.Alert
	query := r.db.WithContext(ctx).
		Preload("Device").
		Where("severity = ?", severity)

	if activeOnly {
		query = query.Where("status = ?", domain.AlertStatusActive)
	}

	query = query.Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&alerts).Error
	return alerts, err
}

// Update 更新告警
func (r *alertRepository) Update(ctx context.Context, alert *domain.Alert) error {
	return r.db.WithContext(ctx).Save(alert).Error
}

// Acknowledge 确认告警
func (r *alertRepository) Acknowledge(ctx context.Context, id uuid.UUID, acknowledgedBy uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&domain.Alert{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":          domain.AlertStatusAcknowledged,
			"acknowledged_by": acknowledgedBy,
			"acknowledged_at": now,
			"updated_at":      now,
		}).Error
}

// Resolve 解决告警
func (r *alertRepository) Resolve(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&domain.Alert{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      domain.AlertStatusResolved,
			"resolved_at": now,
			"updated_at":  now,
		}).Error
}

// UpdateOccurrence 更新告警出现次数和最后出现时间
func (r *alertRepository) UpdateOccurrence(ctx context.Context, id uuid.UUID, occurrenceCount int) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&domain.Alert{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"occurrence_count": occurrenceCount,
			"last_seen_at":     now,
			"updated_at":       now,
		}).Error
}

// EscalateSeverity 提升告警严重程度
func (r *alertRepository) EscalateSeverity(ctx context.Context, id uuid.UUID, newSeverity domain.Severity) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&domain.Alert{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"severity":   newSeverity,
			"updated_at": now,
		}).Error
}

// FindActiveByDeviceAndType 查找设备的特定类型的活跃告警
func (r *alertRepository) FindActiveByDeviceAndType(ctx context.Context, deviceID uuid.UUID, alertType domain.AlertType) (*domain.Alert, error) {
	var alert domain.Alert
	err := r.db.WithContext(ctx).
		Where("device_id = ? AND type = ? AND status = ?", deviceID, alertType, domain.AlertStatusActive).
		First(&alert).Error

	if err != nil {
		return nil, err
	}
	return &alert, nil
}

// ResolveByDeviceAndType 解决设备的特定类型告警
func (r *alertRepository) ResolveByDeviceAndType(ctx context.Context, deviceID uuid.UUID, alertType domain.AlertType) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&domain.Alert{}).
		Where("device_id = ? AND type = ? AND status = ?", deviceID, alertType, domain.AlertStatusActive).
		Updates(map[string]interface{}{
			"status":      domain.AlertStatusResolved,
			"resolved_at": now,
			"updated_at":  now,
		}).Error
}

// GetAlertStats 获取告警统计
func (r *alertRepository) GetAlertStats(ctx context.Context, startTime, endTime time.Time) (*AlertStats, error) {
	var stats AlertStats
	stats.AlertsBySeverity = make(map[string]int64)
	stats.AlertsByType = make(map[string]int64)

	baseQuery := r.db.WithContext(ctx).Model(&domain.Alert{}).
		Where("created_at BETWEEN ? AND ?", startTime, endTime)

	// 总告警数
	if err := baseQuery.Count(&stats.TotalAlerts).Error; err != nil {
		return nil, err
	}

	// 活跃告警数
	if err := baseQuery.Where("status = ?", domain.AlertStatusActive).Count(&stats.ActiveAlerts).Error; err != nil {
		return nil, err
	}

	// 已确认告警数
	if err := r.db.WithContext(ctx).Model(&domain.Alert{}).
		Where("created_at BETWEEN ? AND ? AND status = ?", startTime, endTime, domain.AlertStatusAcknowledged).
		Count(&stats.AcknowledgedAlerts).Error; err != nil {
		return nil, err
	}

	// 已解决告警数
	if err := r.db.WithContext(ctx).Model(&domain.Alert{}).
		Where("created_at BETWEEN ? AND ? AND status = ?", startTime, endTime, domain.AlertStatusResolved).
		Count(&stats.ResolvedAlerts).Error; err != nil {
		return nil, err
	}

	// 严重告警数
	if err := r.db.WithContext(ctx).Model(&domain.Alert{}).
		Where("created_at BETWEEN ? AND ? AND severity = ?", startTime, endTime, domain.SeverityCritical).
		Count(&stats.CriticalAlerts).Error; err != nil {
		return nil, err
	}

	// 按严重程度分组
	var severityCounts []struct {
		Severity string
		Count    int64
	}
	if err := r.db.WithContext(ctx).Model(&domain.Alert{}).
		Select("severity, COUNT(*) as count").
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Group("severity").
		Scan(&severityCounts).Error; err != nil {
		return nil, err
	}
	for _, sc := range severityCounts {
		stats.AlertsBySeverity[sc.Severity] = sc.Count
	}

	// 按告警类型分组
	var typeCounts []struct {
		Type  string
		Count int64
	}
	if err := r.db.WithContext(ctx).Model(&domain.Alert{}).
		Select("type, COUNT(*) as count").
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Group("type").
		Scan(&typeCounts).Error; err != nil {
		return nil, err
	}
	for _, tc := range typeCounts {
		stats.AlertsByType[tc.Type] = tc.Count
	}

	return &stats, nil
}
