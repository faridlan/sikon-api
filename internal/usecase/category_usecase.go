package usecase

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type categoryUsecase struct {
	categoryRepo   domain.CategoryRepository
	contextTimeout time.Duration
}

func NewCategoryUsecase(cr domain.CategoryRepository, timeout time.Duration) domain.CategoryUsecase {
	return &categoryUsecase{
		categoryRepo:   cr,
		contextTimeout: timeout,
	}
}

func (u *categoryUsecase) CreateCategory(c context.Context, category *domain.Category) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Bisa tambahkan validasi misal nama tidak boleh kosong
	if category.Name == "" {
		return domain.NewError(domain.ErrBadParamInput, "Nama kategori tidak boleh kosong")
	}

	return u.categoryRepo.Create(ctx, category)
}

func (u *categoryUsecase) GetCategory(c context.Context, id string) (*domain.Category, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	category, err := u.categoryRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Kategori tidak ditemukan")
		}
		return nil, err
	}

	return category, nil
}

func (u *categoryUsecase) ListCategories(c context.Context, query domain.PaginationQuery) ([]domain.Category, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	offset := query.GetOffset()
	limit := query.Limit

	categories, totalItems, err := u.categoryRepo.Fetch(ctx, limit, offset)
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

	return categories, meta, nil
}

func (u *categoryUsecase) UpdateCategory(c context.Context, category *domain.Category) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingCategory, err := u.categoryRepo.GetByID(ctx, category.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Kategori tidak ditemukan")
		}
		return err
	}

	if category.Name != "" {
		existingCategory.Name = category.Name
	}

	return u.categoryRepo.Update(ctx, existingCategory)
}

func (u *categoryUsecase) DeleteCategory(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.categoryRepo.Delete(ctx, id)
}
