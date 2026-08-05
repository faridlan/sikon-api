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

	// Field Tambahan Catalog & Specs
	Slug          string
	GSMInfo       string   // e.g. "210gsm"
	FabricSummary string   // e.g. "Ripstop"
	Rating        float64  // e.g. 4.9
	SoldCount     int      // e.g. 1240
	ReviewCount   int      // e.g. 318
	KeyFeatures   []string // e.g. ["Bahan ripstop anti robek", "Dual chest pocket"]

	CreatedAt time.Time
	UpdatedAt time.Time

	// Relasi
	Category    *Category
	Images      []ProductImage
	Fabrics     []ProductFabric
	Wholesale   []WholesalePrice
	DesignModel *ProductModel // Relasi ke Canvas Designer Template (jika ada)
}

// --- STRUCT INPUT UNTUK USECASE ---

type FabricColorInput struct {
	Name    string
	HexCode string
}

type ProductFabricInput struct {
	Name            string
	Description     string
	Composition     string
	CareInstruction string
	BasePrice       float64
	PriceAdjustment float64
	IsDefault       bool
	Colors          []FabricColorInput
}

type WholesalePriceInput struct {
	FabricID  *string // Nullable
	MinQty    int
	MaxQty    *int // Nullable
	UnitPrice float64
}

type ProductModelViewInput struct {
	Side    string // "front" / "back"
	ArtURL  string
	MaskURL string
	Width   int
	Height  int
}

type ProductModelInput struct {
	Name        string
	Type        string
	Description string
	Views       []ProductModelViewInput
}

type ProductCreateInput struct {
	CategoryID    string
	Name          string
	Description   string
	BasePrice     float64
	Slug          string
	GSMInfo       string
	FabricSummary string
	KeyFeatures   []string
	ImageURLs     []string
	Fabrics       []ProductFabricInput
	Wholesale     []WholesalePriceInput
	DesignModel   *ProductModelInput // Nullable/Optional jika produk tidak punya custom designer
}

type ProductUpdateInput struct {
	CategoryID    string
	Name          string
	Description   string
	BasePrice     float64
	Slug          string
	GSMInfo       string
	FabricSummary string
	KeyFeatures   []string
	ImageURLs     []string
	Fabrics       []ProductFabricInput
	Wholesale     []WholesalePriceInput
	DesignModel   *ProductModelInput
}

type ProductFilter struct {
	Search     string  // Pencarian nama/description
	CategoryID string  // Filter kategori produk
	MinPrice   float64 // Filter rentang harga
	MaxPrice   float64
	SortBy     string // "popular", "price_low", "price_high", "newest"
}

// --- INTERFACES ---

type ProductRepository interface {
	Create(ctx context.Context, product *Product) error
	GetByID(ctx context.Context, id string) (*Product, error)
	GetBySlug(ctx context.Context, slug string) (*Product, error)
	Fetch(ctx context.Context, filter ProductFilter, limit, offset int) ([]Product, int64, error)
	Update(ctx context.Context, product *Product) error
	Delete(ctx context.Context, id string) error
}

type ProductUsecase interface {
	CreateProduct(ctx context.Context, input ProductCreateInput) (*Product, error)
	GetProduct(ctx context.Context, id string) (*Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*Product, error)
	ListProducts(ctx context.Context, filter ProductFilter, query PaginationQuery) ([]Product, PaginationMeta, error)
	UpdateProduct(ctx context.Context, id string, input ProductUpdateInput) (*Product, error)
	DeleteProduct(ctx context.Context, id string) error
}
