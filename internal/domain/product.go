package domain

import (
	"context"
	"time"
)

type ProductImage struct {
	ID        string
	ProductID string
	ImageURL  string
	IsPrimary bool
}

type Product struct {
	ID          string
	CategoryID  string
	Name        string
	Description string
	BasePrice   float64
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Images   []ProductImage
	Category *Category // Relasi
}

type ProductCreateInput struct {
	CategoryID  string
	Name        string
	Description string
	BasePrice   float64
	ImageURLs   []string // URL gambar produk
}

type ProductUpdateInput struct {
	CategoryID  string
	Name        string
	Description string
	BasePrice   float64
	ImageURLs   []string // URL gambar produk
}

type ProductFilter struct {
	Search     string // Pencarian nama produk
	CategoryID string // Filter kategori produk
}

type ProductRepository interface {
	Create(ctx context.Context, product *Product) error
	GetByID(ctx context.Context, id string) (*Product, error)
	Fetch(ctx context.Context, filter ProductFilter, limit, offset int) ([]Product, int64, error)
	Update(ctx context.Context, product *Product) error
	Delete(ctx context.Context, id string) error
}

type ProductUsecase interface {
	CreateProduct(ctx context.Context, input ProductCreateInput) (*Product, error)
	GetProduct(ctx context.Context, id string) (*Product, error)
	ListProducts(c context.Context, filter ProductFilter, query PaginationQuery) ([]Product, PaginationMeta, error)
	UpdateProduct(ctx context.Context, id string, input ProductUpdateInput) (*Product, error)
	DeleteProduct(ctx context.Context, id string) error
}
