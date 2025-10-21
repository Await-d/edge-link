package repository

import (
	"context"

	"github.com/edgelink/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// VirtualNetworkRepository 虚拟网络仓储接口
type VirtualNetworkRepository interface {
	Create(ctx context.Context, vn *domain.VirtualNetwork) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.VirtualNetwork, error)
	FindByOrganization(ctx context.Context, orgID uuid.UUID) ([]domain.VirtualNetwork, error)
	Update(ctx context.Context, vn *domain.VirtualNetwork) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type virtualNetworkRepository struct {
	db *gorm.DB
}

// NewVirtualNetworkRepository 创建虚拟网络仓储实例
func NewVirtualNetworkRepository(db *gorm.DB) VirtualNetworkRepository {
	return &virtualNetworkRepository{db: db}
}

func (r *virtualNetworkRepository) Create(ctx context.Context, vn *domain.VirtualNetwork) error {
	return r.db.WithContext(ctx).Create(vn).Error
}

func (r *virtualNetworkRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.VirtualNetwork, error) {
	var vn domain.VirtualNetwork
	err := r.db.WithContext(ctx).
		Preload("Organization").
		First(&vn, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &vn, nil
}

func (r *virtualNetworkRepository) FindByOrganization(ctx context.Context, orgID uuid.UUID) ([]domain.VirtualNetwork, error) {
	var vns []domain.VirtualNetwork
	err := r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Find(&vns).Error
	return vns, err
}

func (r *virtualNetworkRepository) Update(ctx context.Context, vn *domain.VirtualNetwork) error {
	return r.db.WithContext(ctx).Save(vn).Error
}

func (r *virtualNetworkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.VirtualNetwork{}, "id = ?", id).Error
}
