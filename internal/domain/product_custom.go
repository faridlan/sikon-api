package domain

import "time"

// FabricColor menyimpan swatch warna per jenis bahan kain
type FabricColor struct {
	ID        string
	FabricID  string
	Name      string // e.g. "Khaki", "Navy", "Olive"
	HexCode   string // e.g. "#C2B280"
	CreatedAt time.Time
}

// ProductFabric menyimpan opsi bahan kain, harga dasar kain, & daftar warnanya
type ProductFabric struct {
	ID              string
	ProductID       string
	Name            string  // e.g. "Nagata Drill", "American Drill"
	Description     string  // e.g. "Tebal & kokoh, 260gsm"
	Composition     string  // e.g. "100% Cotton"
	CareInstruction string  // e.g. "Cuci mesin air dingin"
	BasePrice       float64 // e.g. 395000 (Nagata) vs 375000 (American)
	PriceAdjustment float64 // Opsi relatif (opsional)
	IsDefault       bool
	CreatedAt       time.Time
	UpdatedAt       time.Time

	Colors []FabricColor // Relasi warna khusus bahan ini
}

// WholesalePrice menyimpan tiering harga grosir (Fleksibel: Umum atau Khusus Per Kain)
type WholesalePrice struct {
	ID        string
	ProductID string
	FabricID  *string // Pointer agar bisa NULL (jika berlaku umum per produk)
	MinQty    int
	MaxQty    *int    // Pointer agar bisa NULL (jika min_qty+)
	UnitPrice float64 // Harga satuan grosir
	CreatedAt time.Time
}

// ProductModelView menyimpan aset PNG Art (Lineart) & Mask (Siluet) per Sisi
type ProductModelView struct {
	ID             string
	ProductModelID string
	Side           string // "front" / "back"
	ArtURL         string // URL PNG Lineart (Saku, Kancing, Jahitan)
	MaskURL        string // URL PNG Mask Siluet (Fill Warna Canvas)
	Width          int    // e.g. 1756
	Height         int    // e.g. 1920
	CreatedAt      time.Time
}

// ProductModel menghubungkan Produk Katalog dengan Template Canvas Designer
type ProductModel struct {
	ID          string
	ProductID   string
	Name        string // e.g. "Series 1 — Lengan Panjang"
	Type        string // "long_sleeve" / "short_sleeve"
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Views []ProductModelView // Views untuk "front" dan "back"
}
