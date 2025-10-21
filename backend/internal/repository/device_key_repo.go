package repository

import (
	"context"
	"time"

	"github.com/edgelink/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeviceKeyRepository 设备密钥仓储接口
type DeviceKeyRepository interface {
	Create(ctx context.Context, key *domain.DeviceKey) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.DeviceKey, error)
	FindActiveByDevice(ctx context.Context, deviceID uuid.UUID) (*domain.DeviceKey, error)
	FindExpiringKeys(ctx context.Context, before time.Time) ([]domain.DeviceKey, error)
	Update(ctx context.Context, key *domain.DeviceKey) error
	RevokeKey(ctx context.Context, id uuid.UUID) error
}

type deviceKeyRepository struct {
	db *gorm.DB
}

// NewDeviceKeyRepository 创建设备密钥仓储实例
func NewDeviceKeyRepository(db *gorm.DB) DeviceKeyRepository {
	return &deviceKeyRepository{db: db}
}

func (r *deviceKeyRepository) Create(ctx context.Context, key *domain.DeviceKey) error {
	return r.db.WithContext(ctx).Create(key).Error
}

func (r *deviceKeyRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.DeviceKey, error) {
	var key domain.DeviceKey
	err := r.db.WithContext(ctx).First(&key, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *deviceKeyRepository) FindActiveByDevice(ctx context.Context, deviceID uuid.UUID) (*domain.DeviceKey, error) {
	var key domain.DeviceKey
	err := r.db.WithContext(ctx).
		Where("device_id = ? AND status = ?", deviceID, domain.KeyStatusActive).
		First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *deviceKeyRepository) FindExpiringKeys(ctx context.Context, before time.Time) ([]domain.DeviceKey, error) {
	var keys []domain.DeviceKey
	err := r.db.WithContext(ctx).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < ?", domain.KeyStatusActive, before).
		Find(&keys).Error
	return keys, err
}

func (r *deviceKeyRepository) Update(ctx context.Context, key *domain.DeviceKey) error {
	return r.db.WithContext(ctx).Save(key).Error
}

func (r *deviceKeyRepository) RevokeKey(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&domain.DeviceKey{}).
		Where("id = ?", id).
		Update("status", domain.KeyStatusRevoked).Error
}
