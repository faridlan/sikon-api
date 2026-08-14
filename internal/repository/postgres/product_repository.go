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
	db := GetTx(ctx, r.db)

	if err := db.WithContext(ctx).Create(model).Error; err != nil {
		return TranslateError(err)
	}

	product.ID = model.ID
	product.CreatedAt = model.CreatedAt
	product.UpdatedAt = model.UpdatedAt

	// Sync ID Images
	for i := range product.Images {
		if i < len(model.Images) {
			product.Images[i].ID = model.Images[i].ID
			product.Images[i].ProductID = model.ID
		}
	}

	// Sync ID Fabrics & Colors
	for i := range product.Fabrics {
		if i < len(model.Fabrics) {
			product.Fabrics[i].ID = model.Fabrics[i].ID
			product.Fabrics[i].ProductID = model.ID
			for j := range product.Fabrics[i].Colors {
				if j < len(model.Fabrics[i].Colors) {
					product.Fabrics[i].Colors[j].ID = model.Fabrics[i].Colors[j].ID
					fabricID := model.Fabrics[i].ID
					product.Fabrics[i].Colors[j].FabricID = fabricID
				}
			}
		}
	}

	// Sync ID Wholesale
	for i := range product.Wholesale {
		if i < len(model.Wholesale) {
			product.Wholesale[i].ID = model.Wholesale[i].ID
			product.Wholesale[i].ProductID = model.ID
		}
	}

	// Sync ID DesignModel & Views
	if product.DesignModel != nil && model.DesignModel != nil {
		product.DesignModel.ID = model.DesignModel.ID
		product.DesignModel.ProductID = model.ID
		for i := range product.DesignModel.Views {
			if i < len(model.DesignModel.Views) {
				product.DesignModel.Views[i].ID = model.DesignModel.Views[i].ID
				product.DesignModel.Views[i].ProductModelID = model.DesignModel.ID
			}
		}
	}

	return nil
}

func (r *productRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	var model ProductModel
	if err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Images").
		Preload("Fabrics.SpecTemplate").
		Preload("Fabrics.SpecTemplate.Colors"). // Preload warna global milik Master SpecTemplate
		Preload("Fabrics.Colors").
		Preload("Wholesale").
		Preload("DesignModel.Views").
		Where("id = ?", id).
		First(&model).Error; err != nil {
		return nil, TranslateError(err)
	}
	return model.ToDomain(), nil
}

func (r *productRepository) GetBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	var model ProductModel
	if err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("Images").
		Preload("Fabrics.SpecTemplate").
		Preload("Fabrics.SpecTemplate.Colors"). // Preload warna global milik Master SpecTemplate
		Preload("Fabrics.Colors").
		Preload("Wholesale").
		Preload("DesignModel.Views").
		Where("slug = ?", slug).
		First(&model).Error; err != nil {
		return nil, TranslateError(err)
	}
	return model.ToDomain(), nil
}

func (r *productRepository) Fetch(ctx context.Context, filter domain.ProductFilter, limit, offset int) ([]domain.Product, int64, error) {
	var models []ProductModel
	var total int64

	query := r.db.WithContext(ctx).Model(&ProductModel{})

	// 1. Filter Search (Pencarian Nama / Description)
	if filter.Search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	// 2. Filter Kategori
	if filter.CategoryID != "" {
		query = query.Where("category_id = ?", filter.CategoryID)
	}

	// 3. Filter Rentang Harga
	if filter.MinPrice > 0 {
		query = query.Where("base_price >= ?", filter.MinPrice)
	}
	if filter.MaxPrice > 0 {
		query = query.Where("base_price <= ?", filter.MaxPrice)
	}

	// 4. Hitung Total Rows
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	// 5. Sorting Dinamis
	orderClause := "created_at DESC"
	switch filter.SortBy {
	case "popular":
		orderClause = "sold_count DESC, rating DESC"
	case "price_low":
		orderClause = "base_price ASC"
	case "price_high":
		orderClause = "base_price DESC"
	case "newest":
		orderClause = "created_at DESC"
	}

	// 6. Fetch dengan Preload
	err := query.
		Preload("Category").
		Preload("Images").
		Preload("Fabrics.SpecTemplate").
		Preload("Fabrics.SpecTemplate.Colors"). // Preload warna global milik Master SpecTemplate
		Preload("Fabrics.Colors").
		Preload("Wholesale").
		Preload("DesignModel.Views").
		Limit(limit).
		Offset(offset).
		Order(orderClause).
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
	db := GetTx(ctx, r.db)

	// 1. Update data utama (Tabel products)
	if err := db.WithContext(ctx).Model(&ProductModel{ID: model.ID}).Updates(model).Error; err != nil {
		return TranslateError(err)
	}

	// 2. Sync Product Images
	if err := syncProductImages(ctx, db, model.ID, model.Images); err != nil {
		return err
	}

	// 3. Sync Product Fabrics & Colors
	if err := syncProductFabrics(ctx, db, model.ID, model.Fabrics); err != nil {
		return err
	}

	// 4. Sync Wholesale Prices
	if err := syncWholesalePrices(ctx, db, model.ID, model.Wholesale); err != nil {
		return err
	}

	// 5. Sync Designer Model & Views
	if err := syncDesignerModel(ctx, db, model.ID, model.DesignModel); err != nil {
		return err
	}

	return nil
}

func (r *productRepository) Delete(ctx context.Context, id string) error {
	db := GetTx(ctx, r.db)
	if err := db.WithContext(ctx).Where("id = ?", id).Delete(&ProductModel{}).Error; err != nil {
		return TranslateError(err)
	}
	return nil
}

// --- HELPER FUNCTIONS FOR UPDATE SYNC ---

func syncProductImages(ctx context.Context, db *gorm.DB, productID string, images []ProductImageModel) error {
	var keptIDs []string
	for _, img := range images {
		if img.ID != "" {
			keptIDs = append(keptIDs, img.ID)
		}
	}

	deleteQuery := db.WithContext(ctx).Where("product_id = ?", productID)
	if len(keptIDs) > 0 {
		deleteQuery = deleteQuery.Where("id NOT IN ?", keptIDs)
	}
	if err := deleteQuery.Delete(&ProductImageModel{}).Error; err != nil {
		return TranslateError(err)
	}

	for _, img := range images {
		img.ProductID = productID
		if img.ID == "" {
			if err := db.WithContext(ctx).Create(&img).Error; err != nil {
				return TranslateError(err)
			}
		} else {
			if err := db.WithContext(ctx).Model(&ProductImageModel{ID: img.ID}).Updates(img).Error; err != nil {
				return TranslateError(err)
			}
		}
	}
	return nil
}

func syncProductFabrics(ctx context.Context, db *gorm.DB, productID string, fabrics []ProductFabricModel) error {
	var keptFabricIDs []string
	for _, fab := range fabrics {
		if fab.ID != "" {
			keptFabricIDs = append(keptFabricIDs, fab.ID)
		}
	}

	deleteFabQuery := db.WithContext(ctx).Where("product_id = ?", productID)
	if len(keptFabricIDs) > 0 {
		deleteFabQuery = deleteFabQuery.Where("id NOT IN ?", keptFabricIDs)
	}
	if err := deleteFabQuery.Delete(&ProductFabricModel{}).Error; err != nil {
		return TranslateError(err)
	}

	for _, fab := range fabrics {
		fab.ProductID = productID
		if fab.ID == "" {
			if err := db.WithContext(ctx).Create(&fab).Error; err != nil {
				return TranslateError(err)
			}
		} else {
			if err := db.WithContext(ctx).Model(&ProductFabricModel{ID: fab.ID}).Updates(fab).Error; err != nil {
				return TranslateError(err)
			}
			// Sync colors inside this fabric
			var keptColorIDs []string
			for _, c := range fab.Colors {
				if c.ID != "" {
					keptColorIDs = append(keptColorIDs, c.ID)
				}
			}
			deleteColorQuery := db.WithContext(ctx).Where("fabric_id = ?", fab.ID)
			if len(keptColorIDs) > 0 {
				deleteColorQuery = deleteColorQuery.Where("id NOT IN ?", keptColorIDs)
			}
			if err := deleteColorQuery.Delete(&FabricColorModel{}).Error; err != nil {
				return TranslateError(err)
			}

			for _, c := range fab.Colors {
				fabricID := fab.ID
				c.FabricID = &fabricID
				if c.ID == "" {
					if err := db.WithContext(ctx).Create(&c).Error; err != nil {
						return TranslateError(err)
					}
				} else {
					if err := db.WithContext(ctx).Model(&FabricColorModel{ID: c.ID}).Updates(c).Error; err != nil {
						return TranslateError(err)
					}
				}
			}
		}
	}
	return nil
}

func syncWholesalePrices(ctx context.Context, db *gorm.DB, productID string, wholesales []WholesalePriceModel) error {
	var keptIDs []string
	for _, w := range wholesales {
		if w.ID != "" {
			keptIDs = append(keptIDs, w.ID)
		}
	}

	deleteQuery := db.WithContext(ctx).Where("product_id = ?", productID)
	if len(keptIDs) > 0 {
		deleteQuery = deleteQuery.Where("id NOT IN ?", keptIDs)
	}
	if err := deleteQuery.Delete(&WholesalePriceModel{}).Error; err != nil {
		return TranslateError(err)
	}

	for _, w := range wholesales {
		w.ProductID = productID
		if w.ID == "" {
			if err := db.WithContext(ctx).Create(&w).Error; err != nil {
				return TranslateError(err)
			}
		} else {
			if err := db.WithContext(ctx).Model(&WholesalePriceModel{ID: w.ID}).Updates(w).Error; err != nil {
				return TranslateError(err)
			}
		}
	}
	return nil
}

func syncDesignerModel(ctx context.Context, db *gorm.DB, productID string, dm *DesignerModel) error {
	if dm == nil {
		if err := db.WithContext(ctx).Where("product_id = ?", productID).Delete(&DesignerModel{}).Error; err != nil {
			return TranslateError(err)
		}
		return nil
	}

	dm.ProductID = productID
	if dm.ID == "" {
		if err := db.WithContext(ctx).Create(dm).Error; err != nil {
			return TranslateError(err)
		}
	} else {
		if err := db.WithContext(ctx).Model(&DesignerModel{ID: dm.ID}).Updates(dm).Error; err != nil {
			return TranslateError(err)
		}
	}
	return nil
}
