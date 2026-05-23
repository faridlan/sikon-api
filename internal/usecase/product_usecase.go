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
	contextTimeout time.Duration
}

func NewProductUsecase(pr domain.ProductRepository, cr domain.CategoryRepository, timeout time.Duration) domain.ProductUsecase {
	return &productUsecase{
		productRepo:    pr,
		categoryRepo:   cr,
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

func (u *productUsecase) ListProducts(c context.Context, query domain.PaginationQuery) ([]domain.Product, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	offset := query.GetOffset()
	limit := query.Limit

	products, totalItems, err := u.productRepo.Fetch(ctx, limit, offset)
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

	// Jika CategoryID diganti, validasi lagi kategorinya
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

	if err := u.productRepo.Update(ctx, existingProduct); err != nil {
		return nil, err
	}

	return existingProduct, nil
}

func (u *productUsecase) DeleteProduct(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.productRepo.Delete(ctx, id)
}
