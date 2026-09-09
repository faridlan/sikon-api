package postgres

import (
	"context"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

// ==========================================
// REPOSITORY: MATERIAL
// ==========================================
type materialRepository struct {
	db *gorm.DB
}

func NewMaterialRepository(db *gorm.DB) domain.MaterialRepository {
	return &materialRepository{db: db}
}

func (r *materialRepository) Create(ctx context.Context, material *domain.Material) error {
	model := FromMaterialDomain(material)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return TranslateError(err)
	}

	material.ID = model.ID
	material.CreatedAt = model.CreatedAt
	return nil
}

func (r *materialRepository) GetByID(ctx context.Context, id string) (*domain.Material, error) {
	var model MaterialModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, TranslateError(err)
	}
	return model.ToDomain(), nil
}

func (r *materialRepository) Fetch(ctx context.Context, limit, offset int) ([]domain.Material, int64, error) {
	var models []MaterialModel
	var total int64

	query := r.db.WithContext(ctx).Model(&MaterialModel{})
	query.Count(&total)

	err := query.
		Order("name ASC").
		Limit(limit).
		Offset(offset).
		Find(&models).Error

	if err != nil {
		return nil, 0, TranslateError(err)
	}

	var results []domain.Material
	for _, m := range models {
		results = append(results, *m.ToDomain())
	}

	return results, total, nil
}

func (r *materialRepository) Update(ctx context.Context, material *domain.Material) error {
	model := FromMaterialDomain(material)

	result := r.db.WithContext(ctx).Where("id = ?", material.ID).Updates(model)
	if result.Error != nil {
		return TranslateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *materialRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&MaterialModel{})
	if result.Error != nil {
		return TranslateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ==========================================
// REPOSITORY: PRODUCT MATERIAL (Resep / BOM)
// ==========================================
type productMaterialRepository struct {
	db *gorm.DB
}

func NewProductMaterialRepository(db *gorm.DB) domain.ProductMaterialRepository {
	return &productMaterialRepository{db: db}
}

// ReplaceForProduct: hapus semua baris resep lama untuk produk ini, insert baris baru.
// Dipanggil oleh usecase di dalam txManager.RunInTransaction supaya atomic
// (tidak ada momen "resep kosong" kalau request gagal di tengah jalan).
func (r *productMaterialRepository) ReplaceForProduct(ctx context.Context, productID string, items []domain.ProductMaterial) error {
	if err := r.db.WithContext(ctx).Where("product_id = ?", productID).Delete(&ProductMaterialModel{}).Error; err != nil {
		return TranslateError(err)
	}

	if len(items) == 0 {
		return nil
	}

	models := make([]ProductMaterialModel, len(items))
	for i, item := range items {
		models[i] = ProductMaterialModel{
			ProductID:  productID,
			MaterialID: item.MaterialID,
			QtyPerUnit: item.QtyPerUnit,
		}
	}

	if err := r.db.WithContext(ctx).Create(&models).Error; err != nil {
		return TranslateError(err)
	}

	return nil
}

func (r *productMaterialRepository) FetchByProduct(ctx context.Context, productID string) ([]domain.ProductMaterial, error) {
	var models []ProductMaterialModel

	err := r.db.WithContext(ctx).
		Preload("Material").
		Where("product_id = ?", productID).
		Find(&models).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	var results []domain.ProductMaterial
	for _, m := range models {
		results = append(results, *m.ToDomain())
	}

	return results, nil
}
