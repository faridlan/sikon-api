package postgres

import (
	"context"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type categoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) domain.CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) Create(ctx context.Context, category *domain.Category) error {
	model := FromCategoryDomain(category)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return TranslateError(err)
	}

	category.ID = model.ID
	category.CreatedAt = model.CreatedAt
	category.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *categoryRepository) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	var model CategoryModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, TranslateError(err)
	}
	return model.ToDomain(), nil
}

func (r *categoryRepository) Fetch(ctx context.Context, limit, offset int) ([]domain.Category, int64, error) {
	var models []CategoryModel
	var total int64

	if err := r.db.WithContext(ctx).Model(&CategoryModel{}).Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	if err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Order("name ASC").Find(&models).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	categories := make([]domain.Category, len(models))
	for i, model := range models {
		categories[i] = *model.ToDomain()
	}
	return categories, total, nil
}

func (r *categoryRepository) Update(ctx context.Context, category *domain.Category) error {
	model := FromCategoryDomain(category)
	if err := r.db.WithContext(ctx).Model(&CategoryModel{ID: model.ID}).Updates(model).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}

func (r *categoryRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&CategoryModel{}).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}
