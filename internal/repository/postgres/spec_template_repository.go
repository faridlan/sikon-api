package postgres

import (
	"context"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type specTemplateRepository struct {
	db *gorm.DB
}

func NewSpecTemplateRepository(db *gorm.DB) domain.SpecTemplateRepository {
	return &specTemplateRepository{db: db}
}

func (r *specTemplateRepository) Create(ctx context.Context, specTemplate *domain.SpecTemplate) error {
	model := FromSpecTemplateDomain(specTemplate)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return TranslateError(err)
	}

	specTemplate.ID = model.ID
	specTemplate.CreatedAt = model.CreatedAt
	specTemplate.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *specTemplateRepository) GetByID(ctx context.Context, id string) (*domain.SpecTemplate, error) {
	var model SpecTemplateModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, TranslateError(err)
	}
	return model.ToDomain(), nil
}

func (r *specTemplateRepository) Fetch(ctx context.Context, limit, offset int) ([]domain.SpecTemplate, int64, error) {
	var models []SpecTemplateModel
	var total int64

	if err := r.db.WithContext(ctx).Model(&SpecTemplateModel{}).Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	if err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Order("name ASC").Find(&models).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	specTemplates := make([]domain.SpecTemplate, len(models))
	for i, model := range models {
		specTemplates[i] = *model.ToDomain()
	}
	return specTemplates, total, nil
}

func (r *specTemplateRepository) Update(ctx context.Context, specTemplate *domain.SpecTemplate) error {
	model := FromSpecTemplateDomain(specTemplate)
	if err := r.db.WithContext(ctx).Model(&SpecTemplateModel{ID: model.ID}).Updates(model).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}

func (r *specTemplateRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&SpecTemplateModel{}).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}
