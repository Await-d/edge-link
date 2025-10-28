package repository

import (
	"context"

	"github.com/edgelink/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeviceRepository 设备仓储接口
type DeviceRepository interface {
	Create(ctx context.Context, device *domain.Device) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Device, error)
	FindByPublicKey(ctx context.Context, publicKey string) (*domain.Device, error)
	FindByVirtualNetwork(ctx context.Context, vnID uuid.UUID, online *bool) ([]domain.Device, error)
	Update(ctx context.Context, device *domain.Device) error
	UpdateOnlineStatus(ctx context.Context, id uuid.UUID, online bool) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByOrganization(ctx context.Context, orgID *uuid.UUID) (int, error)
}

type deviceRepository struct {
	db *gorm.DB
}

// NewDeviceRepository 创建设备仓储实例
func NewDeviceRepository(db *gorm.DB) DeviceRepository {
	return &deviceRepository{db: db}
}

func (r *deviceRepository) Create(ctx context.Context, device *domain.Device) error {
	return r.db.WithContext(ctx).Create(device).Error
}

func (r *deviceRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Device, error) {
	var device domain.Device
	err := r.db.WithContext(ctx).
		Preload("VirtualNetwork").
		First(&device, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *deviceRepository) FindByPublicKey(ctx context.Context, publicKey string) (*domain.Device, error) {
	var device domain.Device
	err := r.db.WithContext(ctx).
		Where("public_key = ?", publicKey).
		First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *deviceRepository) FindByVirtualNetwork(ctx context.Context, vnID uuid.UUID, online *bool) ([]domain.Device, error) {
	var devices []domain.Device
	query := r.db.WithContext(ctx)
	
	// 如果vnID不是零值UUID，则过滤特定虚拟网络
	if vnID != (uuid.UUID{}) {
		query = query.Where("virtual_network_id = ?", vnID)
	}

	if online != nil {
		query = query.Where("online = ?", *online)
	}

	err := query.Order("created_at DESC").Find(&devices).Error
	return devices, err
}

func (r *deviceRepository) Update(ctx context.Context, device *domain.Device) error {
	return r.db.WithContext(ctx).Save(device).Error
}

func (r *deviceRepository) UpdateOnlineStatus(ctx context.Context, id uuid.UUID, online bool) error {
	return r.db.WithContext(ctx).
		Model(&domain.Device{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"online":       online,
			"last_seen_at": gorm.Expr("NOW()"),
		}).Error
}

func (r *deviceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Device{}, "id = ?", id).Error
}

func (r *deviceRepository) CountByOrganization(ctx context.Context, orgID *uuid.UUID) (int, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&domain.Device{})

	if orgID != nil {
		query = query.Joins("JOIN virtual_networks ON devices.virtual_network_id = virtual_networks.id").
			Where("virtual_networks.organization_id = ?", *orgID)
	}

	err := query.Count(&count).Error
	return int(count), err
}
