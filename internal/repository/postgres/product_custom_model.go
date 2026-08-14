package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type FabricColorModel struct {
	ID             string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	FabricID       *string   `gorm:"type:uuid;index"`
	SpecTemplateID *string   `gorm:"type:uuid;index"`
	Name           string    `gorm:"type:varchar(100);not null"`
	HexCode        string    `gorm:"type:varchar(20);not null"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}

func (FabricColorModel) TableName() string {
	return "fabric_colors"
}

func (m *FabricColorModel) ToDomain() domain.FabricColor {
	fabricID := ""
	if m.FabricID != nil {
		fabricID = *m.FabricID
	}

	return domain.FabricColor{
		ID:             m.ID,
		FabricID:       fabricID,
		SpecTemplateID: m.SpecTemplateID,
		Name:           m.Name,
		HexCode:        m.HexCode,
		CreatedAt:      m.CreatedAt,
	}
}

type ProductFabricModel struct {
	ID              string             `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ProductID       string             `gorm:"type:uuid;not null"`
	SpecTemplateID  *string            `gorm:"type:uuid;index"`
	Name            string             `gorm:"type:varchar(255)"`
	Description     string             `gorm:"type:text"`
	Composition     string             `gorm:"type:varchar(255)"`
	CareInstruction string             `gorm:"type:text"`
	BasePrice       float64            `gorm:"type:decimal(12,2);default:0"`
	PriceAdjustment float64            `gorm:"type:decimal(12,2);default:0"`
	IsDefault       bool               `gorm:"default:false"`
	CreatedAt       time.Time          `gorm:"autoCreateTime"`
	UpdatedAt       time.Time          `gorm:"autoUpdateTime"`
	Colors          []FabricColorModel `gorm:"foreignKey:FabricID;constraint:OnDelete:CASCADE;"`

	// Relasi GORM
	SpecTemplate *SpecTemplateModel `gorm:"foreignKey:SpecTemplateID"`
}

func (ProductFabricModel) TableName() string {
	return "product_fabrics"
}

func (m *ProductFabricModel) ToDomain() domain.ProductFabric {
	fabric := domain.ProductFabric{
		ID:              m.ID,
		ProductID:       m.ProductID,
		SpecTemplateID:  m.SpecTemplateID,
		Name:            m.Name,
		Description:     m.Description,
		Composition:     m.Composition,
		CareInstruction: m.CareInstruction,
		BasePrice:       m.BasePrice,
		PriceAdjustment: m.PriceAdjustment,
		IsDefault:       m.IsDefault,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}

	// Jika nama/deskripsi/komposisi/instruksi kosong di level ProductFabric, gunakan fallback dari SpecTemplate
	if m.SpecTemplate != nil {
		fabric.SpecTemplate = m.SpecTemplate.ToDomain()
		if fabric.Name == "" {
			fabric.Name = m.SpecTemplate.Name
		}
		if fabric.Description == "" {
			fabric.Description = m.SpecTemplate.Spec
		}
		if fabric.Composition == "" {
			fabric.Composition = m.SpecTemplate.Composition
		}
		if fabric.CareInstruction == "" {
			fabric.CareInstruction = m.SpecTemplate.CareInstruction
		}
	}

	// 1. Warna khusus yang di-override di level ProductFabric
	if len(m.Colors) > 0 {
		var colors []domain.FabricColor
		for _, c := range m.Colors {
			colors = append(colors, c.ToDomain())
		}
		fabric.Colors = colors
		// 2. Fallback: Warna dari Master SpecTemplate jika ProductFabric tidak memiliki warna override
	} else if m.SpecTemplate != nil && len(m.SpecTemplate.Colors) > 0 {
		var colors []domain.FabricColor
		for _, c := range m.SpecTemplate.Colors {
			colors = append(colors, c.ToDomain())
		}
		fabric.Colors = colors
	}

	return fabric
}

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
