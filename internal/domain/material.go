package domain

import (
	"context"
	"time"
)

// ==========================================
// ENTITAS & INPUT: MATERIAL (Master Bahan untuk HPP)
// ==========================================

type Material struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Unit            string        `json:"unit"`
	UnitPrice       float64       `json:"unit_price"`
	Category        string        `json:"category"`
	Description     string        `json:"description,omitempty"`
	Composition     string        `json:"composition,omitempty"`
	CareInstruction string        `json:"care_instruction,omitempty"`
	GSMInfo         string        `json:"gsm_info,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	Colors          []FabricColor `json:"colors,omitempty"`
}

type MaterialCreateInput struct {
	Name            string
	Unit            string
	UnitPrice       float64
	Category        string
	Description     string
	Composition     string
	CareInstruction string
	GSMInfo         string
	Colors          []FabricColorInput
}

type MaterialUpdateInput struct {
	Name            string
	Unit            string
	UnitPrice       float64
	Category        string
	Description     string
	Composition     string
	CareInstruction string
	GSMInfo         string
	Colors          []FabricColorInput
}

type MaterialRepository interface {
	Create(ctx context.Context, material *Material) error
	GetByID(ctx context.Context, id string) (*Material, error)
	Fetch(ctx context.Context, limit, offset int) ([]Material, int64, error)
	Update(ctx context.Context, material *Material) error
	Delete(ctx context.Context, id string) error
}

type MaterialUsecase interface {
	CreateMaterial(ctx context.Context, input MaterialCreateInput) (*Material, error)
	GetMaterial(ctx context.Context, id string) (*Material, error)
	ListMaterials(ctx context.Context, query PaginationQuery) ([]Material, PaginationMeta, error)
	UpdateMaterial(ctx context.Context, id string, input MaterialUpdateInput) (*Material, error)
	DeleteMaterial(ctx context.Context, id string) error
}

// ==========================================
// ENTITAS & INPUT: PRODUCT MATERIAL (Resep / BOM)
// ==========================================

type ProductMaterial struct {
	ID         string  `json:"id"`
	ProductID  string  `json:"product_id"`
	MaterialID string  `json:"material_id"`
	QtyPerUnit float64 `json:"qty_per_unit"`

	// Relasi (untuk ditampilkan, diisi lewat Preload)
	Material *Material `json:"material,omitempty"`
}

type ProductMaterialItemInput struct {
	MaterialID string
	QtyPerUnit float64
}

// SetProductMaterialsInput dipakai untuk simpan resep produk sekaligus (replace-all),
// bukan tambah satu-satu. Lebih simpel karena resep biasanya diedit sebagai satu kesatuan
// dari halaman "Edit Resep Produk", bukan ditambah baris per baris secara terpisah.
type SetProductMaterialsInput struct {
	ProductID string
	Items     []ProductMaterialItemInput
}

type ProductMaterialRepository interface {
	// ReplaceForProduct menghapus semua resep lama produk ini lalu insert yang baru,
	// dibungkus dalam satu transaction di level usecase.
	ReplaceForProduct(ctx context.Context, productID string, items []ProductMaterial) error
	FetchByProduct(ctx context.Context, productID string) ([]ProductMaterial, error)
}

type ProductMaterialUsecase interface {
	SetProductMaterials(ctx context.Context, input SetProductMaterialsInput) ([]ProductMaterial, error)
	GetProductMaterials(ctx context.Context, productID string) ([]ProductMaterial, error)

	// CalculateMaterialCost = qty order dikali seluruh resep produk ini (atau dinamis dari fabricID terpilih).
	// Dipakai nanti oleh order_usecase.go saat menghitung HPP.
	CalculateMaterialCost(ctx context.Context, productID string, qty int, fabricID ...string) (float64, error)
}
