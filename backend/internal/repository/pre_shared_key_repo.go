package repository

import (
	"context"

	"github.com/edgelink/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PreSharedKeyRepository 预共享密钥仓储接口
type PreSharedKeyRepository interface {
	Create(ctx context.Context, psk *domain.PreSharedKey) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.PreSharedKey, error)
	FindByKeyHash(ctx context.Context, keyHash string) (*domain.PreSharedKey, error)
	FindByOrganization(ctx context.Context, orgID uuid.UUID) ([]domain.PreSharedKey, error)
	Update(ctx context.Context, psk *domain.PreSharedKey) error
	IncrementUsedCount(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type preSharedKeyRepository struct {
	db *gorm.DB
}

// NewPreSharedKeyRepository 创建预共享密钥仓储实例
func NewPreSharedKeyRepository(db *gorm.DB) PreSharedKeyRepository {
	return &preSharedKeyRepository{db: db}
}

func (r *preSharedKeyRepository) Create(ctx context.Context, psk *domain.PreSharedKey) error {
	return r.db.WithContext(ctx).Create(psk).Error
}

func (r *preSharedKeyRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.PreSharedKey, error) {
	var psk domain.PreSharedKey
	err := r.db.WithContext(ctx).First(&psk, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &psk, nil
}

func (r *preSharedKeyRepository) FindByKeyHash(ctx context.Context, keyHash string) (*domain.PreSharedKey, error) {
	var psk domain.PreSharedKey
	err := r.db.WithContext(ctx).
		Where("key_hash = ?", keyHash).
		First(&psk).Error
	if err != nil {
		return nil, err
	}
	return &psk, nil
}

func (r *preSharedKeyRepository) FindByOrganization(ctx context.Context, orgID uuid.UUID) ([]domain.PreSharedKey, error) {
	var psks []domain.PreSharedKey
	err := r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Find(&psks).Error
	return psks, err
}

func (r *preSharedKeyRepository) Update(ctx context.Context, psk *domain.PreSharedKey) error {
	return r.db.WithContext(ctx).Save(psk).Error
}

func (r *preSharedKeyRepository) IncrementUsedCount(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&domain.PreSharedKey{}).
		Where("id = ?", id).
		UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error
}

func (r *preSharedKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.PreSharedKey{}, "id = ?", id).Error
}
