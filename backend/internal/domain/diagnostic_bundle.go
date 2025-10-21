package domain

import (
	"time"

	"github.com/google/uuid"
)

// DiagnosticStatus 诊断包状态枚举
type DiagnosticStatus string

const (
	DiagnosticStatusRequested  DiagnosticStatus = "requested"
	DiagnosticStatusCollecting DiagnosticStatus = "collecting"
	DiagnosticStatusUploaded   DiagnosticStatus = "uploaded"
	DiagnosticStatusFailed     DiagnosticStatus = "failed"
	DiagnosticStatusExpired    DiagnosticStatus = "expired"
)

// DiagnosticBundle 诊断包实体
type DiagnosticBundle struct {
	ID                        uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DeviceID                  uuid.UUID        `gorm:"type:uuid;not null;index" json:"device_id"`
	Status                    DiagnosticStatus `gorm:"type:diagnostic_status_enum;not null;default:'requested';index" json:"status"`
	S3ObjectKey               *string          `gorm:"type:text" json:"s3_object_key,omitempty"`
	S3Bucket                  *string          `gorm:"type:varchar(255)" json:"s3_bucket,omitempty"`
	FileSizeBytes             *int64           `json:"file_size_bytes,omitempty"`
	IncludeLogs               bool             `gorm:"not null;default:true" json:"include_logs"`
	IncludeWireGuardStats     bool             `gorm:"not null;default:true" json:"include_wireguard_stats"`
	IncludeNetworkTrace       bool             `gorm:"not null;default:false" json:"include_network_trace"`
	CollectionDurationSeconds *int             `gorm:"default:60" json:"collection_duration_seconds,omitempty"`
	ErrorMessage              *string          `gorm:"type:text" json:"error_message,omitempty"`
	RequestedBy               *uuid.UUID       `json:"requested_by,omitempty"`
	RequestedAt               time.Time        `gorm:"not null;default:now();index:,sort:desc" json:"requested_at"`
	CompletedAt               *time.Time       `json:"completed_at,omitempty"`
	ExpiresAt                 *time.Time       `gorm:"index" json:"expires_at,omitempty"`
	CreatedAt                 time.Time        `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt                 time.Time        `gorm:"not null;default:now()" json:"updated_at"`

	// 关联
	Device *Device `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
}

// TableName 指定表名
func (DiagnosticBundle) TableName() string {
	return "diagnostic_bundles"
}

// IsExpired 检查诊断包是否过期
func (db *DiagnosticBundle) IsExpired() bool {
	return db.ExpiresAt != nil && time.Now().After(*db.ExpiresAt)
}
