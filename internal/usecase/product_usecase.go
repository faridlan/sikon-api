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

func (u *productUsecase) CreateProduct(c context.Context, product *domain.Product) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	_, err := u.categoryRepo.GetByID(ctx, product.CategoryID)
	if err != nil {
		// PENAMBAHAN IF STATEMENT
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrBadParamInput, "Kategori tidak ditemukan")
		}
		return err
	}

	if err := u.productRepo.Create(ctx, product); err != nil {
		return err
	}

	return nil
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

func (u *productUsecase) UpdateProduct(c context.Context, product *domain.Product) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingProduct, err := u.productRepo.GetByID(ctx, product.ID)
	if err != nil {
		// PENAMBAHAN IF STATEMENT
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Produk dengan ID tersebut tidak ditemukan")
		}
		return err
	}

	if product.CategoryID != "" && product.CategoryID != existingProduct.CategoryID {
		_, err := u.categoryRepo.GetByID(ctx, product.CategoryID)
		if err != nil {
			// PENAMBAHAN IF STATEMENT
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NewError(domain.ErrBadParamInput, "Kategori baru tidak ditemukan")
			}
			return err
		}
		existingProduct.CategoryID = product.CategoryID
	}

	if product.Name != "" {
		existingProduct.Name = product.Name
	}
	if product.Description != "" {
		existingProduct.Description = product.Description
	}
	if product.BasePrice > 0 {
		existingProduct.BasePrice = product.BasePrice
	}

	return u.productRepo.Update(ctx, existingProduct)
}

func (u *productUsecase) DeleteProduct(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.productRepo.Delete(ctx, id)
}
