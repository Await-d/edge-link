package domain

import (
	"time"

	"github.com/google/uuid"
)

// ResourceType 资源类型枚举
type ResourceType string

const (
	ResourceTypeDevice         ResourceType = "device"
	ResourceTypeVirtualNetwork ResourceType = "virtual_network"
	ResourceTypePreSharedKey   ResourceType = "pre_shared_key"
	ResourceTypeAlert          ResourceType = "alert"
	ResourceTypeOrganization   ResourceType = "organization"
)

// AuditLog 审计日志实体（不可变）
type AuditLog struct {
	ID             uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;index" json:"organization_id"`
	ActorID        *uuid.UUID   `gorm:"type:uuid;index" json:"actor_id,omitempty"`
	Action         string       `gorm:"type:varchar(100);not null;index" json:"action"`
	ResourceType   ResourceType `gorm:"type:resource_type_enum;not null;index" json:"resource_type"`
	ResourceID     uuid.UUID    `gorm:"type:uuid;not null;index" json:"resource_id"`
	BeforeState    *JSONB       `gorm:"type:jsonb" json:"before_state,omitempty"`
	AfterState     *JSONB       `gorm:"type:jsonb" json:"after_state,omitempty"`
	IPAddress      *string      `gorm:"type:inet" json:"ip_address,omitempty"`
	UserAgent      *string      `gorm:"type:text" json:"user_agent,omitempty"`
	CreatedAt      time.Time    `gorm:"not null;default:now();index:,sort:desc" json:"created_at"`

	// 关联
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
}

// TableName 指定表名
func (AuditLog) TableName() string {
	return "audit_logs"
}
