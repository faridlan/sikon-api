package domain

import (
	"context"
	"time"
)

type Product struct {
	ID          string
	CategoryID  string
	Name        string
	Description string
	BasePrice   float64
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Category *Category
}

type ProductRepository interface {
	Create(ctx context.Context, product *Product) error
	GetByID(ctx context.Context, id string) (*Product, error)
	Fetch(ctx context.Context, limit, offset int) ([]Product, int64, error)
	Update(ctx context.Context, product *Product) error
	Delete(ctx context.Context, id string) error
}

type ProductUsecase interface {
	CreateProduct(ctx context.Context, product *Product) error
	GetProduct(ctx context.Context, id string) (*Product, error)
	ListProducts(c context.Context, query PaginationQuery) ([]Product, PaginationMeta, error)
	UpdateProduct(ctx context.Context, product *Product) error
	DeleteProduct(ctx context.Context, id string) error
}
