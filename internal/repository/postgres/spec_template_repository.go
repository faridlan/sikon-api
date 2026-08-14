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
	if err := r.db.WithContext(ctx).
		Preload("Colors"). // <-- TAMBAHKAN PRELOAD COLORS DI SINI
		Where("id = ?", id).
		First(&model).Error; err != nil {
		return nil, TranslateError(err)
	}

	return model.ToDomain(), nil
}

func (r *specTemplateRepository) Fetch(ctx context.Context, limit, offset int) ([]domain.SpecTemplate, int64, error) {
	var models []SpecTemplateModel
	var totalItems int64

	if err := r.db.WithContext(ctx).Model(&SpecTemplateModel{}).Count(&totalItems).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	if err := r.db.WithContext(ctx).
		Preload("Colors"). // <-- TAMBAHKAN PRELOAD COLORS DI SINI
		Limit(limit).
		Offset(offset).
		Find(&models).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	specTemplates := make([]domain.SpecTemplate, len(models))
	for i, m := range models {
		specTemplates[i] = *m.ToDomain()
	}

	return specTemplates, totalItems, nil
}

func (r *specTemplateRepository) Update(ctx context.Context, specTemplate *domain.SpecTemplate) error {
	model := FromSpecTemplateDomain(specTemplate)

	// Menggunakan Transaction untuk sync warna jika ada
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Update data utama spec template
		if err := tx.Model(&SpecTemplateModel{}).Where("id = ?", model.ID).Updates(model).Error; err != nil {
			return TranslateError(err)
		}

		// 2. Jika ada update warna (replace/sync warna)
		if len(model.Colors) > 0 {
			// Hapus warna lama yang terikat dengan spec_template_id ini
			if err := tx.Where("spec_template_id = ?", model.ID).Delete(&FabricColorModel{}).Error; err != nil {
				return TranslateError(err)
			}

			// Insert warna baru
			for i := range model.Colors {
				model.Colors[i].SpecTemplateID = &model.ID
			}
			if err := tx.Create(&model.Colors).Error; err != nil {
				return TranslateError(err)
			}
		}

		return nil
	})
}

func (r *specTemplateRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&SpecTemplateModel{}, "id = ?", id).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}
