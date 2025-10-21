package repository

import (
	"context"
	"time"

	"github.com/edgelink/backend/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminUserRepository 管理员用户仓储接口
type AdminUserRepository interface {
	// Create 创建新管理员用户
	Create(ctx context.Context, user *domain.AdminUser) error

	// FindByID 根据ID查找管理员用户
	FindByID(ctx context.Context, id uuid.UUID) (*domain.AdminUser, error)

	// FindByEmail 根据邮箱查找管理员用户
	FindByEmail(ctx context.Context, email string) (*domain.AdminUser, error)

	// FindByOIDCSubject 根据OIDC Subject查找管理员用户
	FindByOIDCSubject(ctx context.Context, subject string) (*domain.AdminUser, error)

	// FindByOrganizationID 查找组织的所有管理员用户
	FindByOrganizationID(ctx context.Context, organizationID uuid.UUID) ([]*domain.AdminUser, error)

	// FindActiveUsers 查找所有活跃管理员用户
	FindActiveUsers(ctx context.Context, organizationID *uuid.UUID) ([]*domain.AdminUser, error)

	// FindByRole 根据角色查找管理员用户
	FindByRole(ctx context.Context, role domain.Role, organizationID *uuid.UUID) ([]*domain.AdminUser, error)

	// Update 更新管理员用户
	Update(ctx context.Context, user *domain.AdminUser) error

	// UpdateLastLogin 更新最后登录时间
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error

	// SetActive 设置用户活跃状态
	SetActive(ctx context.Context, id uuid.UUID, isActive bool) error

	// Delete 删除管理员用户（软删除）
	Delete(ctx context.Context, id uuid.UUID) error

	// GetUserStats 获取用户统计信息
	GetUserStats(ctx context.Context, organizationID *uuid.UUID) (*AdminUserStats, error)
}

// AdminUserStats 管理员用户统计信息
type AdminUserStats struct {
	TotalUsers       int64            `json:"total_users"`
	ActiveUsers      int64            `json:"active_users"`
	InactiveUsers    int64            `json:"inactive_users"`
	UsersByRole      map[string]int64 `json:"users_by_role"`
	UsersWithOIDC    int64            `json:"users_with_oidc"`
	RecentLogins     int64            `json:"recent_logins_24h"`
}

// adminUserRepository AdminUser仓储的GORM实现
type adminUserRepository struct {
	db *gorm.DB
}

// NewAdminUserRepository 创建AdminUser仓储实例
func NewAdminUserRepository(db *gorm.DB) AdminUserRepository {
	return &adminUserRepository{db: db}
}

// Create 创建新管理员用户
func (r *adminUserRepository) Create(ctx context.Context, user *domain.AdminUser) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// FindByID 根据ID查找管理员用户
func (r *adminUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.AdminUser, error) {
	var user domain.AdminUser
	err := r.db.WithContext(ctx).
		Preload("Organization").
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail 根据邮箱查找管理员用户
func (r *adminUserRepository) FindByEmail(ctx context.Context, email string) (*domain.AdminUser, error) {
	var user domain.AdminUser
	err := r.db.WithContext(ctx).
		Preload("Organization").
		Where("email = ?", email).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByOIDCSubject 根据OIDC Subject查找管理员用户
func (r *adminUserRepository) FindByOIDCSubject(ctx context.Context, subject string) (*domain.AdminUser, error) {
	var user domain.AdminUser
	err := r.db.WithContext(ctx).
		Preload("Organization").
		Where("oidc_subject = ?", subject).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByOrganizationID 查找组织的所有管理员用户
func (r *adminUserRepository) FindByOrganizationID(ctx context.Context, organizationID uuid.UUID) ([]*domain.AdminUser, error) {
	var users []*domain.AdminUser
	err := r.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		Order("created_at DESC").
		Find(&users).Error
	return users, err
}

// FindActiveUsers 查找所有活跃管理员用户
func (r *adminUserRepository) FindActiveUsers(ctx context.Context, organizationID *uuid.UUID) ([]*domain.AdminUser, error) {
	var users []*domain.AdminUser
	query := r.db.WithContext(ctx).
		Preload("Organization").
		Where("is_active = ?", true)

	if organizationID != nil {
		query = query.Where("organization_id = ?", *organizationID)
	}

	err := query.Order("created_at DESC").Find(&users).Error
	return users, err
}

// FindByRole 根据角色查找管理员用户
func (r *adminUserRepository) FindByRole(ctx context.Context, role domain.Role, organizationID *uuid.UUID) ([]*domain.AdminUser, error) {
	var users []*domain.AdminUser
	query := r.db.WithContext(ctx).
		Preload("Organization").
		Where("role = ?", role)

	if organizationID != nil {
		query = query.Where("organization_id = ?", *organizationID)
	}

	err := query.Order("created_at DESC").Find(&users).Error
	return users, err
}

// Update 更新管理员用户
func (r *adminUserRepository) Update(ctx context.Context, user *domain.AdminUser) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// UpdateLastLogin 更新最后登录时间
func (r *adminUserRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&domain.AdminUser{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_login_at": now,
			"updated_at":    now,
		}).Error
}

// SetActive 设置用户活跃状态
func (r *adminUserRepository) SetActive(ctx context.Context, id uuid.UUID, isActive bool) error {
	return r.db.WithContext(ctx).
		Model(&domain.AdminUser{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_active":  isActive,
			"updated_at": time.Now(),
		}).Error
}

// Delete 删除管理员用户（软删除 - 设置为不活跃）
func (r *adminUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.SetActive(ctx, id, false)
}

// GetUserStats 获取用户统计信息
func (r *adminUserRepository) GetUserStats(ctx context.Context, organizationID *uuid.UUID) (*AdminUserStats, error) {
	var stats AdminUserStats
	stats.UsersByRole = make(map[string]int64)

	query := r.db.WithContext(ctx).Model(&domain.AdminUser{})
	if organizationID != nil {
		query = query.Where("organization_id = ?", *organizationID)
	}

	// 总用户数
	if err := query.Count(&stats.TotalUsers).Error; err != nil {
		return nil, err
	}

	// 活跃用户数
	activeQuery := r.db.WithContext(ctx).Model(&domain.AdminUser{}).Where("is_active = ?", true)
	if organizationID != nil {
		activeQuery = activeQuery.Where("organization_id = ?", *organizationID)
	}
	if err := activeQuery.Count(&stats.ActiveUsers).Error; err != nil {
		return nil, err
	}

	// 非活跃用户数
	stats.InactiveUsers = stats.TotalUsers - stats.ActiveUsers

	// 按角色分组
	var roleCounts []struct {
		Role  string
		Count int64
	}
	roleQuery := r.db.WithContext(ctx).Model(&domain.AdminUser{}).
		Select("role, COUNT(*) as count").
		Group("role")
	if organizationID != nil {
		roleQuery = roleQuery.Where("organization_id = ?", *organizationID)
	}
	if err := roleQuery.Scan(&roleCounts).Error; err != nil {
		return nil, err
	}
	for _, rc := range roleCounts {
		stats.UsersByRole[rc.Role] = rc.Count
	}

	// 使用 OIDC 的用户数
	oidcQuery := r.db.WithContext(ctx).Model(&domain.AdminUser{}).
		Where("oidc_subject IS NOT NULL AND oidc_subject != ''")
	if organizationID != nil {
		oidcQuery = oidcQuery.Where("organization_id = ?", *organizationID)
	}
	if err := oidcQuery.Count(&stats.UsersWithOIDC).Error; err != nil {
		return nil, err
	}

	// 最近24小时登录的用户数
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)
	recentQuery := r.db.WithContext(ctx).Model(&domain.AdminUser{}).
		Where("last_login_at >= ?", twentyFourHoursAgo)
	if organizationID != nil {
		recentQuery = recentQuery.Where("organization_id = ?", *organizationID)
	}
	if err := recentQuery.Count(&stats.RecentLogins).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}
