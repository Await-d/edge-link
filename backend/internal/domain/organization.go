package domain

import (
	"time"

	"github.com/google/uuid"
)

// Organization 组织实体
type Organization struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Slug                string    `gorm:"type:varchar(100);unique;not null" json:"slug"`
	Name                string    `gorm:"type:varchar(255);not null" json:"name"`
	MaxDevices          int       `gorm:"not null;default:100" json:"max_devices"`
	MaxVirtualNetworks  int       `gorm:"not null;default:10" json:"max_virtual_networks"`
	CreatedAt           time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt           time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// 关联
	VirtualNetworks []VirtualNetwork `gorm:"foreignKey:OrganizationID" json:"virtual_networks,omitempty"`
	PreSharedKeys   []PreSharedKey   `gorm:"foreignKey:OrganizationID" json:"pre_shared_keys,omitempty"`
	AdminUsers      []AdminUser      `gorm:"foreignKey:OrganizationID" json:"admin_users,omitempty"`
}

// TableName 指定表名
func (Organization) TableName() string {
	return "organizations"
}
