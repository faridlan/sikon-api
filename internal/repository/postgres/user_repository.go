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
		return TranslateError(err) // Harus di-return agar fungsinya berhenti dan melempar error
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

func (r *userRepository) Fetch(ctx context.Context, limit, offset int) ([]domain.User, int64, error) {
	var models []UserModel
	var total int64

	if err := r.db.WithContext(ctx).Model(&UserModel{}).Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err) // Jangan lupa pakai helper di sini juga
	}

	if err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	users := make([]domain.User, len(models))
	for i, model := range models {
		users[i] = *model.ToDomain()
	}

	return users, total, nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	model := FromUserDomain(user)

	err := r.db.WithContext(ctx).Model(&UserModel{ID: model.ID}).Updates(model).Error
	if err != nil {
		return TranslateError(err)
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&UserModel{}).Error
	if err != nil {
		return TranslateError(err)
	}

	return nil
}
