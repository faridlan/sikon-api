package domain

import "time"

type FabricColor struct {
	ID        string
	FabricID  string
	Name      string // e.g. "Khaki", "Navy", "Olive"
	HexCode   string // e.g. "#C2B280"
	CreatedAt time.Time
}

type ProductFabric struct {
	ID              string
	ProductID       string
	SpecTemplateID  *string       // Pointer ke Master Kain Global
	SpecTemplate    *SpecTemplate // Relasi domain
	Name            string        // Custom Name jika ingin override dari SpecTemplate
	Description     string
	Composition     string
	CareInstruction string
	BasePrice       float64 // Harga dasar khusus untuk produk ini
	PriceAdjustment float64 // Penyesuaian harga khusus produk ini
	IsDefault       bool
	CreatedAt       time.Time
	UpdatedAt       time.Time

	Colors []FabricColor
}

type WholesalePrice struct {
	ID        string
	ProductID string
	FabricID  *string
	MinQty    int
	MaxQty    *int
	UnitPrice float64
	CreatedAt time.Time
}

type ProductModelView struct {
	ID             string
	ProductModelID string
	Side           string
	ArtURL         string
	MaskURL        string
	Width          int
	Height         int
	CreatedAt      time.Time
}

type ProductModel struct {
	ID          string
	ProductID   string
	Name        string
	Type        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Views []ProductModelView
}
