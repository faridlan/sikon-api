package usecase

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type productUsecase struct {
	productRepo    domain.ProductRepository
	categoryRepo   domain.CategoryRepository
	storageService domain.StorageService
	txManager      domain.TransactionManager
	contextTimeout time.Duration
}

func NewProductUsecase(pr domain.ProductRepository, cr domain.CategoryRepository, ss domain.StorageService, tm domain.TransactionManager, timeout time.Duration) domain.ProductUsecase {
	return &productUsecase{
		productRepo:    pr,
		categoryRepo:   cr,
		storageService: ss,
		txManager:      tm,
		contextTimeout: timeout,
	}
}

func (u *productUsecase) CreateProduct(c context.Context, input domain.ProductCreateInput) (*domain.Product, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Validasi Bisnis: Pastikan CategoryID ada di database
	_, err := u.categoryRepo.GetByID(ctx, input.CategoryID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrBadParamInput, "Kategori tidak ditemukan")
		}
		return nil, err
	}

	product := &domain.Product{
		CategoryID:  input.CategoryID,
		Name:        input.Name,
		Description: input.Description,
		BasePrice:   input.BasePrice,
	}

	// 🚨 MAPPING ARRAY GAMBAR
	if len(input.ImageURLs) > 0 {
		var images []domain.ProductImage
		for i, url := range input.ImageURLs {
			images = append(images, domain.ProductImage{
				ImageURL:  url,
				IsPrimary: i == 0, // Gambar pertama otomatis diset sebagai primary
			})
		}
		product.Images = images
	}

	if err := u.productRepo.Create(ctx, product); err != nil {
		return nil, err
	}

	return product, nil
}

func (u *productUsecase) GetProduct(c context.Context, id string) (*domain.Product, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	product, err := u.productRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Produk dengan ID tersebut tidak ditemukan")
		}
		return nil, err
	}

	return product, nil
}

func (u *productUsecase) ListProducts(c context.Context, filter domain.ProductFilter, query domain.PaginationQuery) ([]domain.Product, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	offset := query.GetOffset()
	limit := query.Limit

	products, totalItems, err := u.productRepo.Fetch(ctx, filter, limit, offset)
	if err != nil {
		return nil, domain.PaginationMeta{}, err
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))
	meta := domain.PaginationMeta{
		CurrentPage: query.Page,
		Limit:       limit,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}

	return products, meta, nil
}

func (u *productUsecase) UpdateProduct(c context.Context, id string, input domain.ProductUpdateInput) (*domain.Product, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingProduct, err := u.productRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Produk dengan ID tersebut tidak ditemukan")
		}
		return nil, err
	}

	if input.CategoryID != "" && input.CategoryID != existingProduct.CategoryID {
		_, err := u.categoryRepo.GetByID(ctx, input.CategoryID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, domain.NewError(domain.ErrBadParamInput, "Kategori baru tidak ditemukan")
			}
			return nil, err
		}
		existingProduct.CategoryID = input.CategoryID
	}

	if input.Name != "" {
		existingProduct.Name = input.Name
	}
	if input.Description != "" {
		existingProduct.Description = input.Description
	}
	if input.BasePrice > 0 {
		existingProduct.BasePrice = input.BasePrice
	}

	var urlsToDelete []string

	if input.ImageURLs != nil {
		oldImagesMap := make(map[string]string)
		for _, img := range existingProduct.Images {
			oldImagesMap[img.ImageURL] = img.ID
		}

		var newImages []domain.ProductImage
		for i, url := range input.ImageURLs {
			imageID := oldImagesMap[url]
			newImages = append(newImages, domain.ProductImage{
				ID:        imageID,
				ImageURL:  url,
				IsPrimary: i == 0,
			})
			delete(oldImagesMap, url)
		}

		for url := range oldImagesMap {
			urlsToDelete = append(urlsToDelete, url)
		}

		existingProduct.Images = newImages
	}

	// 🚨 BUNGKUS DENGAN TRANSACTION MANAGER
	err = u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Gunakan txCtx agar Repository membaca "Kertas Buram"
		if err := u.productRepo.Update(txCtx, existingProduct); err != nil {
			return err // Otomatis Rollback
		}
		return nil // Otomatis Commit
	})

	if err != nil {
		return nil, err
	}

	// Jika sukses, bersihkan file di storage (Background)
	if len(urlsToDelete) > 0 {
		go func(urls []string) {
			for _, urlToDelete := range urls {
				_ = u.storageService.DeleteFile(context.Background(), urlToDelete)
			}
		}(urlsToDelete)
	}

	return existingProduct, nil
}

func (u *productUsecase) DeleteProduct(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingProduct, err := u.productRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Produk dengan ID tersebut tidak ditemukan")
		}
		return err
	}

	// Hapus dari Database Postgres (Cascade otomatis menghapus data di product_images)
	if err := u.productRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Kumpulkan semua URL gambar milik produk ini
	var urlsToDelete []string
	for _, img := range existingProduct.Images {
		urlsToDelete = append(urlsToDelete, img.ImageURL)
	}

	// Bersihkan storage Supabase di background
	if len(urlsToDelete) > 0 {
		go func(urls []string) {
			for _, urlToDelete := range urls {
				_ = u.storageService.DeleteFile(context.Background(), urlToDelete)
			}
		}(urlsToDelete)
	}

	return nil
}
