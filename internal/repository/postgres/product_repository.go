package postgres

import (
	"context"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) domain.ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) Create(ctx context.Context, product *domain.Product) error {
	model := FromProductDomain(product)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return TranslateError(err)
	}

	product.ID = model.ID
	product.CreatedAt = model.CreatedAt
	product.UpdatedAt = model.UpdatedAt

	for i := range product.Images {
		product.Images[i].ID = model.Images[i].ID
		product.Images[i].ProductID = model.ID
	}

	return nil
}

func (r *productRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	var model ProductModel
	// 🚨 Tambahkan Preload("Images")
	if err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Images"). // Tarik semua gambar milik produk ini
		Where("id = ?", id).
		First(&model).Error; err != nil {
		return nil, TranslateError(err)
	}
	return model.ToDomain(), nil
}

// Tambahkan parameter filter domain.ProductFilter
func (r *productRepository) Fetch(ctx context.Context, filter domain.ProductFilter, limit, offset int) ([]domain.Product, int64, error) {
	var models []ProductModel
	var total int64

	// 1. Inisialisasi Instance Model
	query := r.db.WithContext(ctx).Model(&ProductModel{})

	// 2. Terapkan Filter Dinamis
	if filter.Search != "" {
		// Gunakan ILIKE untuk pencarian case-insensitive pada field nama
		query = query.Where("name ILIKE ?", "%"+filter.Search+"%")
	}

	if filter.CategoryID != "" {
		// Pencarian presisi untuk UUID Kategori
		query = query.Where("category_id = ?", filter.CategoryID)
	}

	// 3. Count harus dipanggil setelah query kondisi dibuat
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	// 4. Tambahkan Preload, Limit, Offset, lalu Find
	err := query.
		Preload("Category").
		Preload("Images").
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&models).Error

	if err != nil {
		return nil, 0, TranslateError(err)
	}

	products := make([]domain.Product, len(models))
	for i, model := range models {
		products[i] = *model.ToDomain()
	}

	return products, total, nil
}

func (r *productRepository) Update(ctx context.Context, product *domain.Product) error {
	model := FromProductDomain(product)

	// 1. Update data utama (Tabel Products)
	if err := r.db.WithContext(ctx).Model(&ProductModel{ID: model.ID}).Updates(model).Error; err != nil {
		return TranslateError(err)
	}

	// 2. Ganti total (Replace) relasi gambarnya
	// Ini akan menghapus baris lama di product_images dan melakukan INSERT baris baru.
	// Jika model.Images kosong, maka GORM akan menghapus semua gambar untuk produk ini.
	err := r.db.WithContext(ctx).Model(&model).Association("Images").Replace(model.Images)
	if err != nil {
		return TranslateError(err)
	}

	return nil
}

func (r *productRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&ProductModel{}).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}
