package domain

import (
	"time"

	"github.com/google/uuid"
)

// ConnectionType 连接类型枚举
type ConnectionType string

const (
	ConnectionTypeP2PDirect ConnectionType = "p2p_direct"
	ConnectionTypeTURNRelay  ConnectionType = "turn_relay"
)

// Session 会话实体
type Session struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DeviceAID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"device_a_id"`
	DeviceBID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"device_b_id"`
	ConnectionType  ConnectionType `gorm:"type:connection_type_enum;not null" json:"connection_type"`
	StartedAt       time.Time      `gorm:"not null;default:now();index" json:"started_at"`
	EndedAt         *time.Time     `gorm:"index" json:"ended_at,omitempty"`
	LastHandshakeAt *time.Time     `json:"last_handshake_at,omitempty"`
	BytesSent       int64          `gorm:"default:0" json:"bytes_sent"`
	BytesReceived   int64          `gorm:"default:0" json:"bytes_received"`
	AvgLatencyMs    *int           `json:"avg_latency_ms,omitempty"`
	CreatedAt       time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"not null;default:now()" json:"updated_at"`

	// 关联
	DeviceA *Device `gorm:"foreignKey:DeviceAID" json:"device_a,omitempty"`
	DeviceB *Device `gorm:"foreignKey:DeviceBID" json:"device_b,omitempty"`
}

// TableName 指定表名
func (Session) TableName() string {
	return "sessions"
}

// IsActive 检查会话是否活跃
func (s *Session) IsActive() bool {
	return s.EndedAt == nil
}

// Duration 获取会话持续时间
func (s *Session) Duration() time.Duration {
	if s.EndedAt != nil {
		return s.EndedAt.Sub(s.StartedAt)
	}
	return time.Since(s.StartedAt)
}
