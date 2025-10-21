package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// NATType NAT类型枚举
type NATType string

const (
	NATTypeNone              NATType = "none"
	NATTypeFullCone          NATType = "full_cone"
	NATTypeRestrictedCone    NATType = "restricted_cone"
	NATTypePortRestrictedCone NATType = "port_restricted_cone"
	NATTypeSymmetric         NATType = "symmetric"
	NATTypeUnknown           NATType = "unknown"
)

// Platform 平台类型枚举
type Platform string

const (
	PlatformDesktopLinux   Platform = "desktop_linux"
	PlatformDesktopWindows Platform = "desktop_windows"
	PlatformDesktopMacOS   Platform = "desktop_macos"
	PlatformMobileIOS      Platform = "mobile_ios"
	PlatformMobileAndroid  Platform = "mobile_android"
	PlatformIoT            Platform = "iot"
	PlatformContainer      Platform = "container"
)

// Device 设备实体
type Device struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	VirtualNetworkID uuid.UUID  `gorm:"type:uuid;not null;index" json:"virtual_network_id"`
	Name             string     `gorm:"type:varchar(255);not null" json:"name"`
	VirtualIP        string     `gorm:"type:inet;not null" json:"virtual_ip"`
	PublicKey        string     `gorm:"type:text;not null;unique" json:"public_key"`
	Platform         Platform        `gorm:"type:platform_enum;not null" json:"platform"`
	NATType          NATType         `gorm:"type:nat_type_enum;default:'unknown'" json:"nat_type"`
	PublicEndpoint   string          `gorm:"type:varchar(255)" json:"public_endpoint,omitempty"`
	Tags             pq.StringArray  `gorm:"type:text[];default:'{}'" json:"tags,omitempty"`
	Online           bool            `gorm:"not null;default:false;index" json:"online"`
	LastSeenAt       *time.Time `gorm:"index" json:"last_seen_at,omitempty"`
	CreatedAt        time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"not null;default:now()" json:"updated_at"`

	// 关联
	VirtualNetwork     *VirtualNetwork     `gorm:"foreignKey:VirtualNetworkID" json:"virtual_network,omitempty"`
	Keys               []DeviceKey         `gorm:"foreignKey:DeviceID" json:"keys,omitempty"`
	PeerConfigurations []PeerConfiguration `gorm:"foreignKey:DeviceID" json:"peer_configurations,omitempty"`
	SessionsAsA        []Session           `gorm:"foreignKey:DeviceAID" json:"-"`
	SessionsAsB        []Session           `gorm:"foreignKey:DeviceBID" json:"-"`
	Alerts             []Alert             `gorm:"foreignKey:DeviceID" json:"alerts,omitempty"`
}

// TableName 指定表名
func (Device) TableName() string {
	return "devices"
}
