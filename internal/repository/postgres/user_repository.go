package postgres

import (
	"context"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository adalah constructor untuk membuat instance userRepository
func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	model := FromUserDomain(user)

	err := r.db.WithContext(ctx).Create(model).Error
	if err != nil {
		return TranslateError(err)
	}

	user.ID = model.ID
	user.CreatedAt = model.CreatedAt
	user.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		return nil, TranslateError(err)
	}

	return model.ToDomain(), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var model UserModel
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error
	if err != nil {
		return nil, TranslateError(err)
	}

	return model.ToDomain(), nil
}

func (r *userRepository) Fetch(ctx context.Context, limit, offset int, filter domain.UserFilter) ([]domain.User, int64, error) {
	var models []UserModel
	var total int64

	query := r.db.WithContext(ctx).Model(&UserModel{})

	// 1. Filter Pencarian Teks (Nama atau Email)
	if filter.Search != "" {
		searchTerm := "%" + filter.Search + "%"
		query = query.Where("name ILIKE ? OR email ILIKE ?", searchTerm, searchTerm)
	}

	// 2. Filter Role
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}

	// 3. Filter Status Aktif (jika di-set)
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, TranslateError(err)
	}

	err = query.Limit(limit).Offset(offset).Order("sort_order ASC, created_at DESC").Find(&models).Error
	if err != nil {
		return nil, 0, TranslateError(err)
	}

	users := make([]domain.User, len(models))
	for i, model := range models {
		users[i] = *model.ToDomain()
	}

	return users, total, nil
}

// GetPublicSalesList mengambil khusus marketing/sales aktif untuk ditampilkan di Landing Page
func (r *userRepository) GetPublicSalesList(ctx context.Context) ([]domain.User, error) {
	var models []UserModel

	err := r.db.WithContext(ctx).
		Where("role = ? AND is_active = ?", domain.RoleSales, true).
		Order("sort_order ASC, created_at ASC").
		Find(&models).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	users := make([]domain.User, len(models))
	for i, model := range models {
		users[i] = *model.ToDomain()
	}

	return users, nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	model := FromUserDomain(user)

	db := GetTx(ctx, r.db)

	err := db.WithContext(ctx).Model(&UserModel{ID: model.ID}).Updates(model).Error
	if err != nil {
		return TranslateError(err)
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	db := GetTx(ctx, r.db)

	err := db.WithContext(ctx).Where("id = ?", id).Delete(&UserModel{}).Error
	if err != nil {
		return TranslateError(err)
	}

	return nil
}
