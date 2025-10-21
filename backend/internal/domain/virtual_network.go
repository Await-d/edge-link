package domain

import (
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// VirtualNetwork 虚拟网络实体
type VirtualNetwork struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrganizationID uuid.UUID      `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name           string         `gorm:"type:varchar(255);not null" json:"name"`
	CIDR           string         `gorm:"type:cidr;not null" json:"cidr"`
	GatewayIP      string         `gorm:"type:inet;not null" json:"gateway_ip"`
	DNSServers     pq.StringArray `gorm:"type:inet[]" json:"dns_servers"`
	CreatedAt      time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"not null;default:now()" json:"updated_at"`

	// 关联
	Organization *Organization `gorm:"foreignKey:OrganizationID" json:"organization,omitempty"`
	Devices      []Device      `gorm:"foreignKey:VirtualNetworkID" json:"devices,omitempty"`
}

// TableName 指定表名
func (VirtualNetwork) TableName() string {
	return "virtual_networks"
}

// CIDRIP 辅助方法：解析 CIDR
func (vn *VirtualNetwork) CIDRIP() (*net.IPNet, error) {
	_, ipnet, err := net.ParseCIDR(vn.CIDR)
	return ipnet, err
}

// Gateway 辅助方法：解析网关 IP
func (vn *VirtualNetwork) Gateway() net.IP {
	return net.ParseIP(vn.GatewayIP)
}

// pq.StringArray already implements sql.Scanner and driver.Valuer interfaces
