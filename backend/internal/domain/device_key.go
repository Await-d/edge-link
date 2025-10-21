package domain

import (
	"time"

	"github.com/google/uuid"
)

// KeyStatus 密钥状态枚举
type KeyStatus string

const (
	KeyStatusActive          KeyStatus = "active"
	KeyStatusPendingRotation KeyStatus = "pending_rotation"
	KeyStatusRevoked         KeyStatus = "revoked"
	KeyStatusExpired         KeyStatus = "expired"
)

// DeviceKey 设备密钥实体
type DeviceKey struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DeviceID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"device_id"`
	PublicKey string     `gorm:"type:text;not null" json:"public_key"`
	Status    KeyStatus  `gorm:"type:key_status_enum;not null;default:'active';index" json:"status"`
	ValidFrom time.Time  `gorm:"not null;default:now()" json:"valid_from"`
	ExpiresAt *time.Time `gorm:"index" json:"expires_at,omitempty"`
	CreatedAt time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time  `gorm:"not null;default:now()" json:"updated_at"`

	// 关联
	Device *Device `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
}

// TableName 指定表名
func (DeviceKey) TableName() string {
	return "device_keys"
}

// IsExpired 检查密钥是否已过期
func (dk *DeviceKey) IsExpired() bool {
	return dk.ExpiresAt != nil && time.Now().After(*dk.ExpiresAt)
}
