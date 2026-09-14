package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type FabricColorModel struct {
	ID         string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	MaterialID *string   `gorm:"type:uuid;not null;index"`
	Name       string    `gorm:"type:varchar(100);not null"`
	HexCode    string    `gorm:"type:varchar(20);not null"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

func (FabricColorModel) TableName() string {
	return "fabric_colors"
}

func (m *FabricColorModel) ToDomain() domain.FabricColor {
	return domain.FabricColor{
		ID:         m.ID,
		MaterialID: m.MaterialID,
		Name:       m.Name,
		HexCode:    m.HexCode,
		CreatedAt:  m.CreatedAt,
	}
}

type ProductFabricModel struct {
	ID              string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ProductID       string    `gorm:"type:uuid;not null;index"`
	MaterialID      string    `gorm:"type:uuid;not null;index"`
	QtyPerUnit      float64   `gorm:"type:numeric(15,4);not null;default:1.5"`
	PriceAdjustment float64   `gorm:"type:decimal(12,2);default:0"`
	IsDefault       bool      `gorm:"default:false"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`

	// Relasi: sumber kebenaran nama/komposisi/warna/harga cost
	Material *MaterialModel `gorm:"foreignKey:MaterialID"`
}

func (ProductFabricModel) TableName() string {
	return "product_fabrics"
}

func (m *ProductFabricModel) ToDomain() domain.ProductFabric {
	fab := domain.ProductFabric{
		ID:              m.ID,
		ProductID:       m.ProductID,
		MaterialID:      m.MaterialID,
		QtyPerUnit:      m.QtyPerUnit,
		PriceAdjustment: m.PriceAdjustment,
		IsDefault:       m.IsDefault,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}

	if m.Material != nil {
		mat := m.Material.ToDomain()
		fab.Material = mat
	}

	return fab
}

// --- WholesalePriceModel, ProductModelViewModel, DesignerModel: TIDAK BERUBAH ---

type WholesalePriceModel struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ProductID string    `gorm:"type:uuid;not null"`
	FabricID  *string   `gorm:"type:uuid"`
	MinQty    int       `gorm:"not null"`
	MaxQty    *int      `gorm:"type:int"`
	UnitPrice float64   `gorm:"type:decimal(12,2);not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (WholesalePriceModel) TableName() string {
	return "wholesale_prices"
}

func (m *WholesalePriceModel) ToDomain() domain.WholesalePrice {
	return domain.WholesalePrice{
		ID:        m.ID,
		ProductID: m.ProductID,
		FabricID:  m.FabricID,
		MinQty:    m.MinQty,
		MaxQty:    m.MaxQty,
		UnitPrice: m.UnitPrice,
		CreatedAt: m.CreatedAt,
	}
}

type ProductModelViewModel struct {
	ID             string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ProductModelID string    `gorm:"type:uuid;not null"`
	Side           string    `gorm:"type:varchar(20);not null"`
	ArtURL         string    `gorm:"type:varchar(255);not null"`
	MaskURL        string    `gorm:"type:varchar(255);not null"`
	Width          int       `gorm:"default:600"`
	Height         int       `gorm:"default:600"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}

func (ProductModelViewModel) TableName() string {
	return "product_model_views"
}

func (m *ProductModelViewModel) ToDomain() domain.ProductModelView {
	return domain.ProductModelView{
		ID:             m.ID,
		ProductModelID: m.ProductModelID,
		Side:           m.Side,
		ArtURL:         m.ArtURL,
		MaskURL:        m.MaskURL,
		Width:          m.Width,
		Height:         m.Height,
		CreatedAt:      m.CreatedAt,
	}
}

type DesignerModel struct {
	ID          string                  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ProductID   string                  `gorm:"type:uuid;not null"`
	Name        string                  `gorm:"type:varchar(255);not null"`
	Type        string                  `gorm:"type:varchar(100);not null"`
	Description string                  `gorm:"type:text"`
	CreatedAt   time.Time               `gorm:"autoCreateTime"`
	UpdatedAt   time.Time               `gorm:"autoUpdateTime"`
	Views       []ProductModelViewModel `gorm:"foreignKey:ProductModelID;constraint:OnDelete:CASCADE;"`
}

func (DesignerModel) TableName() string {
	return "product_models"
}

func (m *DesignerModel) ToDomain() domain.ProductModel {
	dm := domain.ProductModel{
		ID:          m.ID,
		ProductID:   m.ProductID,
		Name:        m.Name,
		Type:        m.Type,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}

	if len(m.Views) > 0 {
		var views []domain.ProductModelView
		for _, v := range m.Views {
			views = append(views, v.ToDomain())
		}
		dm.Views = views
	}

	return dm
}
