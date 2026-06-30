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

	// 🚨 Ambil DB dari Context (Otomatis menggunakan Transaksi jika dipanggil dari Usecase)
	db := GetTx(ctx, r.db)

	// 1. Update data utama (Tabel Products)
	if err := db.WithContext(ctx).Model(&ProductModel{ID: model.ID}).Updates(model).Error; err != nil {
		return TranslateError(err)
	}

	// 2. Kumpulkan ID gambar yang DIPERTAHANKAN
	var keptImageIDs []string
	for _, img := range model.Images {
		if img.ID != "" {
			keptImageIDs = append(keptImageIDs, img.ID)
		}
	}

	// 3. Hapus gambar lama yang TIDAK ADA di request baru (Cegah Error 23502 Constraint)
	if len(keptImageIDs) > 0 {
		if err := db.WithContext(ctx).Where("product_id = ? AND id NOT IN ?", model.ID, keptImageIDs).Delete(&ProductImageModel{}).Error; err != nil {
			return TranslateError(err)
		}
	} else {
		// Jika frontend mengirimkan array kosong (Semua gambar dihapus)
		if err := db.WithContext(ctx).Where("product_id = ?", model.ID).Delete(&ProductImageModel{}).Error; err != nil {
			return TranslateError(err)
		}
	}

	// 4. Upsert (Insert gambar baru ATAU Update gambar lama)
	for _, img := range model.Images {
		img.ProductID = model.ID // Pastikan foreign key selalu terkait

		if img.ID == "" {
			// Jika tidak ada ID, berarti ini gambar baru yang baru diupload
			if err := db.WithContext(ctx).Create(&img).Error; err != nil {
				return TranslateError(err)
			}
		} else {
			// Jika ada ID, update datanya (misal: urutan is_primary berubah)
			if err := db.WithContext(ctx).Model(&ProductImageModel{ID: img.ID}).Updates(img).Error; err != nil {
				return TranslateError(err)
			}
		}
	}

	return nil
}

func (r *productRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&ProductModel{}).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}
