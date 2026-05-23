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

func (u *categoryUsecase) CreateCategory(c context.Context, input domain.CategoryCreateInput) (*domain.Category, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	category := &domain.Category{
		Name: input.Name,
	}

	if err := u.categoryRepo.Create(ctx, category); err != nil {
		return nil, err
	}

	return category, nil
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

func (u *categoryUsecase) UpdateCategory(c context.Context, id string, input domain.CategoryUpdateInput) (*domain.Category, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingCategory, err := u.categoryRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Kategori tidak ditemukan")
		}
		return nil, err
	}

	if input.Name != "" {
		existingCategory.Name = input.Name
	}

	if err := u.categoryRepo.Update(ctx, existingCategory); err != nil {
		return nil, err
	}

	return existingCategory, nil
}

func (u *categoryUsecase) DeleteCategory(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.categoryRepo.Delete(ctx, id)
}
