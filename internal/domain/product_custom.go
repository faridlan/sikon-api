package domain

import "time"

// FabricColor sekarang murni milik Material (bukan lagi milik ProductFabric atau SpecTemplate).
// Kalau kain "American Drill" punya warna Khaki/Olive/Black, warna itu berlaku di SEMUA produk
// yang pakai kain ini — bukan diulang-ulang per produk.
type FabricColor struct {
	ID         string
	MaterialID *string
	Name       string
	HexCode    string
	CreatedAt  time.Time
}

// ProductFabric = link Product <-> Material (kain), plus qty konsumsi & markup harga jual
// yang SPESIFIK untuk kombinasi Product+Kain ini. Semua data deskriptif (nama, komposisi,
// cara rawat, daftar warna) diambil dari Material yang di-link — TIDAK disimpan duplikat lagi.
type ProductFabric struct {
	ID              string
	ProductID       string
	MaterialID      string
	Material        *Material // di-preload, sumber kebenaran nama/komposisi/warna/harga cost
	QtyPerUnit      float64   // konsumsi kain per pcs produk (meter) - dipakai utk HPP kain terpilih
	PriceAdjustment float64   // markup ke harga jual customer (BEDA dari Material.UnitPrice yg itu cost)
	IsDefault       bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// --- WholesalePrice, ProductModelView, DesignerModel: tidak berubah dari sebelumnya ---
// (Input types untuk semua ini tetap tinggal di product.go, tidak dipindah ke sini,
// supaya tidak dobel-declare dengan yang sudah ada)

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
