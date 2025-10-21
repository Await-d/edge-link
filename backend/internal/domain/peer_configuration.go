package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// PeerConfiguration 对等设备配置实体
type PeerConfiguration struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DeviceID            uuid.UUID      `gorm:"type:uuid;not null;index" json:"device_id"`
	PeerDeviceID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"peer_device_id"`
	PeerPublicKey       string         `gorm:"type:text;not null" json:"peer_public_key"`
	PeerVirtualIP       string         `gorm:"type:inet;not null" json:"peer_virtual_ip"`
	AllowedIPs          pq.StringArray `gorm:"type:cidr[]" json:"allowed_ips"`
	PersistentKeepalive *int           `gorm:"default:25" json:"persistent_keepalive,omitempty"`
	CreatedAt           time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"not null;default:now()" json:"updated_at"`

	// 关联
	Device     *Device `gorm:"foreignKey:DeviceID" json:"device,omitempty"`
	PeerDevice *Device `gorm:"foreignKey:PeerDeviceID" json:"peer_device,omitempty"`
}

// TableName 指定表名
func (PeerConfiguration) TableName() string {
	return "peer_configurations"
}
