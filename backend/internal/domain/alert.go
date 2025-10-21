package domain

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Severity 告警严重程度枚举
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// AlertType 告警类型枚举
type AlertType string

const (
	AlertTypeDeviceOffline  AlertType = "device_offline"
	AlertTypeHighLatency    AlertType = "high_latency"
	AlertTypeFailedAuth     AlertType = "failed_auth"
	AlertTypeKeyExpiration  AlertType = "key_expiration"
	AlertTypeTunnelFailure  AlertType = "tunnel_failure"
)

// AlertStatus 告警状态枚举
type AlertStatus string

const (
	AlertStatusActive       AlertStatus = "active"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved     AlertStatus = "resolved"
)

// JSONB 自定义JSON类型
type JSONB map[string]interface{}

// Scan 实现sql.Scanner接口
func (j *JSONB) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Value 实现driver.Valuer接口
func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

// Alert 告警实体
type Alert struct {
	ID              uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DeviceID        *uuid.UUID   `gorm:"type:uuid;index" json:"device_id,omitempty"`
	Severity        Severity     `gorm:"type:severity_enum;not null;index" json:"severity"`
	Type            AlertType    `gorm:"type:alert_type_enum;not null;index" json:"type"`
	Title           string       `gorm:"type:varchar(255);not null" json:"title"`
	Message         string       `gorm:"type:text;not null" json:"message"`
	Metadata        JSONB        `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	Status          AlertStatus  `gorm:"type:alert_status_enum;not null;default:'active';index" json:"status"`
	AcknowledgedBy  *uuid.UUID   `json:"acknowledged_by,omitempty"`
	AcknowledgedAt  *time.Time   `json:"acknowledged_at,omitempty"`
	ResolvedAt      *time.Time   `json:"resolved_at,omitempty"`

	// 去重相关字段
	OccurrenceCount int        `gorm:"default:1;not null" json:"occurrence_count"`
	FirstSeenAt     time.Time  `gorm:"not null;default:now()" json:"first_seen_at"`
	LastSeenAt      time.Time  `gorm:"not null;default:now()" json:"last_seen_at"`

	CreatedAt       time.Time  `gorm:"not null;default:now();index" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"not null;default:now()" json:"updated_at"`

	// 关联
	Device *Device `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
}

// TableName 指定表名
func (Alert) TableName() string {
	return "alerts"
}
