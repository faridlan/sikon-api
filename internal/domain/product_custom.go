package domain

import "time"

type FabricColor struct {
	ID       string
	FabricID string

	// Penambahan field baru (aman, pointer memastikan backward compatibility)
	SpecTemplateID *string

	Name      string
	HexCode   string
	CreatedAt time.Time
}

type ProductFabric struct {
	ID              string
	ProductID       string
	SpecTemplateID  *string
	SpecTemplate    *SpecTemplate
	Name            string
	Description     string
	Composition     string
	CareInstruction string
	BasePrice       float64
	PriceAdjustment float64
	IsDefault       bool
	CreatedAt       time.Time
	UpdatedAt       time.Time

	Colors []FabricColor
}

// ... (sisanya tidak berubah)
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
