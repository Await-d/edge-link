package domain

import (
	"time"

	"github.com/google/uuid"
)

// PreSharedKey 预共享密钥实体
type PreSharedKey struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganizationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"organization_id"`
	KeyHash        string     `gorm:"type:text;not null;unique;index" json:"-"` // 不在JSON中暴露
	Name           *string    `gorm:"type:varchar(255)" json:"name,omitempty"`
	MaxUses        *int       `json:"max_uses,omitempty"`
	UsedCount      int        `gorm:"not null;default:0" json:"used_count"`
	ExpiresAt      *time.Time `gorm:"index" json:"expires_at,omitempty"`
	CreatedAt      time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"not null;default:now()" json:"updated_at"`

	// 关联
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
}

// TableName 指定表名
func (PreSharedKey) TableName() string {
	return "pre_shared_keys"
}

// IsValid 检查PSK是否有效
func (psk *PreSharedKey) IsValid() bool {
	if psk.ExpiresAt != nil && time.Now().After(*psk.ExpiresAt) {
		return false
	}
	if psk.MaxUses != nil && psk.UsedCount >= *psk.MaxUses {
		return false
	}
	return true
}
