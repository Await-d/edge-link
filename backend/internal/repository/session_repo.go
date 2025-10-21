package repository

import (
	"context"
	"time"

	"github.com/edgelink/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SessionRepository 会话仓储接口
type SessionRepository interface {
	// Create 创建新会话
	Create(ctx context.Context, session *domain.Session) error

	// FindByID 根据ID查找会话
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Session, error)

	// FindByDeviceID 查找设备的所有会话
	FindByDeviceID(ctx context.Context, deviceID uuid.UUID, limit int) ([]*domain.Session, error)

	// FindActiveByDevices 查找两个设备间的活跃会话
	FindActiveByDevices(ctx context.Context, deviceAID, deviceBID uuid.UUID) (*domain.Session, error)

	// FindActiveSessions 查找所有活跃会话（未结束）
	FindActiveSessions(ctx context.Context, limit int) ([]*domain.Session, error)

	// FindByVirtualNetwork 查找虚拟网络中的所有会话
	FindByVirtualNetwork(ctx context.Context, virtualNetworkID uuid.UUID, activeOnly bool, limit int) ([]*domain.Session, error)

	// Update 更新会话
	Update(ctx context.Context, session *domain.Session) error

	// EndSession 结束会话
	EndSession(ctx context.Context, id uuid.UUID) error

	// UpdateMetrics 更新会话指标
	UpdateMetrics(ctx context.Context, id uuid.UUID, bytesSent, bytesReceived int64, avgLatencyMs *int) error

	// GetSessionStats 获取会话统计信息
	GetSessionStats(ctx context.Context, startTime, endTime time.Time) (*SessionStats, error)
}

// SessionStats 会话统计信息
type SessionStats struct {
	TotalSessions    int64   `json:"total_sessions"`
	ActiveSessions   int64   `json:"active_sessions"`
	P2PDirectCount   int64   `json:"p2p_direct_count"`
	TURNRelayCount   int64   `json:"turn_relay_count"`
	AvgDuration      float64 `json:"avg_duration_seconds"`
	TotalBytesTransferred int64 `json:"total_bytes_transferred"`
}

// sessionRepository Session仓储的GORM实现
type sessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository 创建Session仓储实例
func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

// Create 创建新会话
func (r *sessionRepository) Create(ctx context.Context, session *domain.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

// FindByID 根据ID查找会话
func (r *sessionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	var session domain.Session
	err := r.db.WithContext(ctx).
		Preload("DeviceA").
		Preload("DeviceB").
		Where("id = ?", id).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// FindByDeviceID 查找设备的所有会话
func (r *sessionRepository) FindByDeviceID(ctx context.Context, deviceID uuid.UUID, limit int) ([]*domain.Session, error) {
	var sessions []*domain.Session
	query := r.db.WithContext(ctx).
		Preload("DeviceA").
		Preload("DeviceB").
		Where("device_a_id = ? OR device_b_id = ?", deviceID, deviceID).
		Order("started_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&sessions).Error
	return sessions, err
}

// FindActiveByDevices 查找两个设备间的活跃会话
func (r *sessionRepository) FindActiveByDevices(ctx context.Context, deviceAID, deviceBID uuid.UUID) (*domain.Session, error) {
	var session domain.Session
	err := r.db.WithContext(ctx).
		Where("((device_a_id = ? AND device_b_id = ?) OR (device_a_id = ? AND device_b_id = ?)) AND ended_at IS NULL",
			deviceAID, deviceBID, deviceBID, deviceAID).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// FindActiveSessions 查找所有活跃会话
func (r *sessionRepository) FindActiveSessions(ctx context.Context, limit int) ([]*domain.Session, error) {
	var sessions []*domain.Session
	query := r.db.WithContext(ctx).
		Preload("DeviceA").
		Preload("DeviceB").
		Where("ended_at IS NULL").
		Order("started_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&sessions).Error
	return sessions, err
}

// FindByVirtualNetwork 查找虚拟网络中的所有会话
func (r *sessionRepository) FindByVirtualNetwork(ctx context.Context, virtualNetworkID uuid.UUID, activeOnly bool, limit int) ([]*domain.Session, error) {
	var sessions []*domain.Session

	// 子查询获取虚拟网络中的设备ID
	subQuery := r.db.Model(&domain.Device{}).
		Select("id").
		Where("virtual_network_id = ?", virtualNetworkID)

	query := r.db.WithContext(ctx).
		Preload("DeviceA").
		Preload("DeviceB").
		Where("device_a_id IN (?) OR device_b_id IN (?)", subQuery, subQuery)

	if activeOnly {
		query = query.Where("ended_at IS NULL")
	}

	query = query.Order("started_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&sessions).Error
	return sessions, err
}

// Update 更新会话
func (r *sessionRepository) Update(ctx context.Context, session *domain.Session) error {
	return r.db.WithContext(ctx).Save(session).Error
}

// EndSession 结束会话
func (r *sessionRepository) EndSession(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&domain.Session{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"ended_at":   now,
			"updated_at": now,
		}).Error
}

// UpdateMetrics 更新会话指标
func (r *sessionRepository) UpdateMetrics(ctx context.Context, id uuid.UUID, bytesSent, bytesReceived int64, avgLatencyMs *int) error {
	updates := map[string]interface{}{
		"bytes_sent":     gorm.Expr("bytes_sent + ?", bytesSent),
		"bytes_received": gorm.Expr("bytes_received + ?", bytesReceived),
		"updated_at":     time.Now(),
	}

	if avgLatencyMs != nil {
		updates["avg_latency_ms"] = *avgLatencyMs
	}

	return r.db.WithContext(ctx).
		Model(&domain.Session{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// GetSessionStats 获取会话统计信息
func (r *sessionRepository) GetSessionStats(ctx context.Context, startTime, endTime time.Time) (*SessionStats, error) {
	var stats SessionStats

	// 总会话数
	if err := r.db.WithContext(ctx).
		Model(&domain.Session{}).
		Where("started_at BETWEEN ? AND ?", startTime, endTime).
		Count(&stats.TotalSessions).Error; err != nil {
		return nil, err
	}

	// 活跃会话数
	if err := r.db.WithContext(ctx).
		Model(&domain.Session{}).
		Where("started_at BETWEEN ? AND ? AND ended_at IS NULL", startTime, endTime).
		Count(&stats.ActiveSessions).Error; err != nil {
		return nil, err
	}

	// P2P直连数量
	if err := r.db.WithContext(ctx).
		Model(&domain.Session{}).
		Where("started_at BETWEEN ? AND ? AND connection_type = ?", startTime, endTime, domain.ConnectionTypeP2PDirect).
		Count(&stats.P2PDirectCount).Error; err != nil {
		return nil, err
	}

	// TURN中继数量
	if err := r.db.WithContext(ctx).
		Model(&domain.Session{}).
		Where("started_at BETWEEN ? AND ? AND connection_type = ?", startTime, endTime, domain.ConnectionTypeTURNRelay).
		Count(&stats.TURNRelayCount).Error; err != nil {
		return nil, err
	}

	// 平均持续时间（秒）
	var avgDuration float64
	err := r.db.WithContext(ctx).
		Model(&domain.Session{}).
		Select("AVG(EXTRACT(EPOCH FROM (COALESCE(ended_at, NOW()) - started_at)))").
		Where("started_at BETWEEN ? AND ?", startTime, endTime).
		Scan(&avgDuration).Error
	if err != nil {
		return nil, err
	}
	stats.AvgDuration = avgDuration

	// 总传输字节数
	var totalBytes int64
	err = r.db.WithContext(ctx).
		Model(&domain.Session{}).
		Select("SUM(bytes_sent + bytes_received)").
		Where("started_at BETWEEN ? AND ?", startTime, endTime).
		Scan(&totalBytes).Error
	if err != nil {
		return nil, err
	}
	stats.TotalBytesTransferred = totalBytes

	return &stats, nil
}
