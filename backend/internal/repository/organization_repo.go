package repository

import (
	"context"

	"github.com/edgelink/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OrganizationRepository 组织仓储接口
type OrganizationRepository interface {
	Create(ctx context.Context, org *domain.Organization) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
	FindBySlug(ctx context.Context, slug string) (*domain.Organization, error)
	Update(ctx context.Context, org *domain.Organization) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]domain.Organization, int64, error)
}

type organizationRepository struct {
	db *gorm.DB
}

// NewOrganizationRepository 创建组织仓储实例
func NewOrganizationRepository(db *gorm.DB) OrganizationRepository {
	return &organizationRepository{db: db}
}

func (r *organizationRepository) Create(ctx context.Context, org *domain.Organization) error {
	return r.db.WithContext(ctx).Create(org).Error
}

func (r *organizationRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	var org domain.Organization
	err := r.db.WithContext(ctx).First(&org, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *organizationRepository) FindBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	var org domain.Organization
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&org).Error
	if err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *organizationRepository) Update(ctx context.Context, org *domain.Organization) error {
	return r.db.WithContext(ctx).Save(org).Error
}

func (r *organizationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Organization{}, "id = ?", id).Error
}

func (r *organizationRepository) List(ctx context.Context, limit, offset int) ([]domain.Organization, int64, error) {
	var orgs []domain.Organization
	var total int64

	if err := r.db.WithContext(ctx).Model(&domain.Organization{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&orgs).Error

	return orgs, total, err
}
