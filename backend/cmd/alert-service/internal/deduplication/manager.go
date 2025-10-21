package deduplication

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	// Redis键前缀
	alertDedupeKeyPrefix = "alert:dedupe:"
	alertLockKeyPrefix   = "alert:lock:"

	// 默认配置
	defaultDedupeWindow    = 30 * time.Minute
	defaultSilentPeriod    = 5 * time.Minute
	defaultLockTimeout     = 5 * time.Second
	defaultEscalationCount = 10
)

// Config 去重配置
type Config struct {
	// DedupeWindow 去重时间窗口（同一告警在此时间内只保留一条）
	DedupeWindow time.Duration

	// SilentPeriod 静默期（告警解决后的静默时间）
	SilentPeriod time.Duration

	// EscalationThreshold 升级阈值（告警出现次数超过此值时提升严重程度）
	EscalationThreshold int

	// LockTimeout 分布式锁超时时间
	LockTimeout time.Duration
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		DedupeWindow:        defaultDedupeWindow,
		SilentPeriod:        defaultSilentPeriod,
		EscalationThreshold: defaultEscalationCount,
		LockTimeout:         defaultLockTimeout,
	}
}

// AlertKey 告警唯一键
type AlertKey struct {
	DeviceID  string
	AlertType string
}

// String 生成字符串表示
func (k AlertKey) String() string {
	return fmt.Sprintf("%s:%s", k.DeviceID, k.AlertType)
}

// Hash 生成哈希值
func (k AlertKey) Hash() string {
	h := sha256.New()
	h.Write([]byte(k.String()))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// AlertDedupeInfo 去重信息
type AlertDedupeInfo struct {
	AlertID       string    `json:"alert_id"`
	DeviceID      string    `json:"device_id"`
	AlertType     string    `json:"alert_type"`
	FirstSeenAt   time.Time `json:"first_seen_at"`
	LastSeenAt    time.Time `json:"last_seen_at"`
	OccurrenceCount int     `json:"occurrence_count"`
	CurrentSeverity string  `json:"current_severity"`
	Escalated       bool    `json:"escalated"`
}

// Manager 告警去重管理器
type Manager struct {
	redis  *redis.Client
	config *Config
	logger *zap.Logger
}

// NewManager 创建去重管理器
func NewManager(redisClient *redis.Client, config *Config, logger *zap.Logger) *Manager {
	if config == nil {
		config = DefaultConfig()
	}

	return &Manager{
		redis:  redisClient,
		config: config,
		logger: logger,
	}
}

// ShouldCreateAlert 检查是否应该创建新告警
// 返回: (shouldCreate bool, existingAlertID *uuid.UUID, shouldEscalate bool, err error)
func (m *Manager) ShouldCreateAlert(ctx context.Context, key AlertKey, severity string) (bool, *uuid.UUID, bool, error) {
	// 尝试获取分布式锁
	lockKey := alertLockKeyPrefix + key.Hash()
	locked, err := m.acquireLock(ctx, lockKey)
	if err != nil {
		return false, nil, false, fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		// 无法获取锁，说明有其他实例正在处理
		m.logger.Debug("Failed to acquire lock, skipping", zap.String("key", key.String()))
		return false, nil, false, nil
	}
	defer m.releaseLock(ctx, lockKey)

	// 检查Redis中是否存在去重信息
	dedupeKey := alertDedupeKeyPrefix + key.Hash()
	data, err := m.redis.Get(ctx, dedupeKey).Result()

	if err == redis.Nil {
		// 不存在，应该创建新告警
		return true, nil, false, nil
	}

	if err != nil {
		return false, nil, false, fmt.Errorf("failed to get dedupe info: %w", err)
	}

	// 解析去重信息
	var info AlertDedupeInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		m.logger.Error("Failed to unmarshal dedupe info", zap.Error(err))
		// 数据损坏，创建新告警
		return true, nil, false, nil
	}

	// 检查是否在去重窗口内
	if time.Since(info.LastSeenAt) > m.config.DedupeWindow {
		// 超出去重窗口，创建新告警
		return true, nil, false, nil
	}

	// 在去重窗口内，更新现有告警
	alertID, err := uuid.Parse(info.AlertID)
	if err != nil {
		m.logger.Error("Invalid alert ID in dedupe info", zap.Error(err))
		return true, nil, false, nil
	}

	// 检查是否需要升级
	shouldEscalate := !info.Escalated &&
		info.OccurrenceCount >= m.config.EscalationThreshold &&
		severity != "critical"

	return false, &alertID, shouldEscalate, nil
}

// RecordAlert 记录告警信息用于去重
func (m *Manager) RecordAlert(ctx context.Context, alertID uuid.UUID, key AlertKey, severity string, isNew bool) error {
	dedupeKey := alertDedupeKeyPrefix + key.Hash()
	now := time.Now()

	var info AlertDedupeInfo

	if isNew {
		// 新告警
		info = AlertDedupeInfo{
			AlertID:         alertID.String(),
			DeviceID:        key.DeviceID,
			AlertType:       key.AlertType,
			FirstSeenAt:     now,
			LastSeenAt:      now,
			OccurrenceCount: 1,
			CurrentSeverity: severity,
			Escalated:       false,
		}
	} else {
		// 更新现有告警
		data, err := m.redis.Get(ctx, dedupeKey).Result()
		if err != nil {
			return fmt.Errorf("failed to get existing dedupe info: %w", err)
		}

		if err := json.Unmarshal([]byte(data), &info); err != nil {
			return fmt.Errorf("failed to unmarshal dedupe info: %w", err)
		}

		info.LastSeenAt = now
		info.OccurrenceCount++
		info.CurrentSeverity = severity
	}

	// 序列化并保存到Redis
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal dedupe info: %w", err)
	}

	// 设置过期时间为去重窗口的2倍（防止边界问题）
	ttl := m.config.DedupeWindow * 2
	if err := m.redis.Set(ctx, dedupeKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to save dedupe info: %w", err)
	}

	m.logger.Debug("Recorded alert dedupe info",
		zap.String("alert_id", alertID.String()),
		zap.String("key", key.String()),
		zap.Int("occurrence_count", info.OccurrenceCount),
	)

	return nil
}

// MarkEscalated 标记告警已升级
func (m *Manager) MarkEscalated(ctx context.Context, key AlertKey) error {
	dedupeKey := alertDedupeKeyPrefix + key.Hash()

	data, err := m.redis.Get(ctx, dedupeKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get dedupe info: %w", err)
	}

	var info AlertDedupeInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return fmt.Errorf("failed to unmarshal dedupe info: %w", err)
	}

	info.Escalated = true

	newData, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal dedupe info: %w", err)
	}

	ttl := m.config.DedupeWindow * 2
	if err := m.redis.Set(ctx, dedupeKey, newData, ttl).Err(); err != nil {
		return fmt.Errorf("failed to save dedupe info: %w", err)
	}

	return nil
}

// RemoveDedupeInfo 移除去重信息（用于告警解决）
func (m *Manager) RemoveDedupeInfo(ctx context.Context, key AlertKey) error {
	dedupeKey := alertDedupeKeyPrefix + key.Hash()

	if err := m.redis.Del(ctx, dedupeKey).Err(); err != nil {
		return fmt.Errorf("failed to delete dedupe info: %w", err)
	}

	m.logger.Debug("Removed dedupe info", zap.String("key", key.String()))
	return nil
}

// IsInSilentPeriod 检查告警是否在静默期内
func (m *Manager) IsInSilentPeriod(ctx context.Context, key AlertKey) (bool, error) {
	silentKey := "alert:silent:" + key.Hash()

	exists, err := m.redis.Exists(ctx, silentKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check silent period: %w", err)
	}

	return exists > 0, nil
}

// SetSilentPeriod 设置静默期
func (m *Manager) SetSilentPeriod(ctx context.Context, key AlertKey) error {
	silentKey := "alert:silent:" + key.Hash()

	if err := m.redis.Set(ctx, silentKey, "1", m.config.SilentPeriod).Err(); err != nil {
		return fmt.Errorf("failed to set silent period: %w", err)
	}

	m.logger.Debug("Set silent period",
		zap.String("key", key.String()),
		zap.Duration("duration", m.config.SilentPeriod),
	)

	return nil
}

// GetOccurrenceCount 获取告警出现次数
func (m *Manager) GetOccurrenceCount(ctx context.Context, key AlertKey) (int, error) {
	dedupeKey := alertDedupeKeyPrefix + key.Hash()

	data, err := m.redis.Get(ctx, dedupeKey).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get dedupe info: %w", err)
	}

	var info AlertDedupeInfo
	if err := json.Unmarshal([]byte(data), &info); err != nil {
		return 0, fmt.Errorf("failed to unmarshal dedupe info: %w", err)
	}

	return info.OccurrenceCount, nil
}

// acquireLock 获取分布式锁
func (m *Manager) acquireLock(ctx context.Context, key string) (bool, error) {
	result, err := m.redis.SetNX(ctx, key, "1", m.config.LockTimeout).Result()
	if err != nil {
		return false, err
	}
	return result, nil
}

// releaseLock 释放分布式锁
func (m *Manager) releaseLock(ctx context.Context, key string) error {
	return m.redis.Del(ctx, key).Err()
}
