package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role 角色枚举
type Role string

const (
	RoleSuperAdmin      Role = "super_admin"
	RoleAdmin           Role = "admin"
	RoleNetworkOperator Role = "network_operator"
	RoleAuditor         Role = "auditor"
	RoleReadonly        Role = "readonly"
)

// AdminUser 管理员用户实体
type AdminUser struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganizationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"organization_id"`
	Email          string     `gorm:"type:varchar(255);not null;unique;index" json:"email"`
	Name           string     `gorm:"type:varchar(255);not null" json:"name"`
	Role           Role       `gorm:"type:role_enum;not null;default:'readonly';index" json:"role"`
	OIDCSubject    *string    `gorm:"type:varchar(255);index" json:"oidc_subject,omitempty"`
	IsActive       bool       `gorm:"not null;default:true;index" json:"is_active"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
	CreatedAt      time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"not null;default:now()" json:"updated_at"`

	// 关联
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
}

// TableName 指定表名
func (AdminUser) TableName() string {
	return "admin_users"
}

// HasPermission 检查用户是否有特定权限
func (u *AdminUser) HasPermission(requiredRole Role) bool {
	roleHierarchy := map[Role]int{
		RoleReadonly:        1,
		RoleAuditor:         2,
		RoleNetworkOperator: 3,
		RoleAdmin:           4,
		RoleSuperAdmin:      5,
	}
	return roleHierarchy[u.Role] >= roleHierarchy[requiredRole]
}
