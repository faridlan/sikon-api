package domain

import (
	"context"
	"time"
)

type Category struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CategoryRepository interface {
	Create(ctx context.Context, category *Category) error
	GetByID(ctx context.Context, id string) (*Category, error)
	Fetch(ctx context.Context, limit, offset int) ([]Category, int64, error)
	Update(ctx context.Context, category *Category) error
	Delete(ctx context.Context, id string) error
}

type CategoryUsecase interface {
	CreateCategory(ctx context.Context, category *Category) error
	GetCategory(ctx context.Context, id string) (*Category, error)
	ListCategories(c context.Context, query PaginationQuery) ([]Category, PaginationMeta, error)
	UpdateCategory(ctx context.Context, category *Category) error
	DeleteCategory(ctx context.Context, id string) error
}
