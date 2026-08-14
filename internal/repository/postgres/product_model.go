package postgres

import (
	"encoding/json"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type ProductModel struct {
	ID            string                `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	CategoryID    string                `gorm:"type:uuid;not null"`
	Name          string                `gorm:"type:varchar(255);not null"`
	Description   string                `gorm:"type:text"`
	BasePrice     float64               `gorm:"type:decimal(12,2);not null;default:0"`
	Slug          string                `gorm:"type:varchar(255);uniqueIndex"`
	GSMInfo       string                `gorm:"type:varchar(100)"`
	FabricSummary string                `gorm:"type:varchar(100)"`
	Rating        float64               `gorm:"type:decimal(3,2);default:0.00"`
	SoldCount     int                   `gorm:"default:0"`
	ReviewCount   int                   `gorm:"default:0"`
	KeyFeatures   string                `gorm:"type:jsonb;default:'[]'"` // Disimpan sebagai JSON String di GORM
	Images        []ProductImageModel   `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE;"`
	Fabrics       []ProductFabricModel  `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE;"`
	Wholesale     []WholesalePriceModel `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE;"`
	DesignModel   *DesignerModel        `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE;"`
	CreatedAt     time.Time             `gorm:"autoCreateTime"`
	UpdatedAt     time.Time             `gorm:"autoUpdateTime"`
	DeletedAt     gorm.DeletedAt        `gorm:"index"`

	// Relasi
	Category *CategoryModel `gorm:"foreignKey:CategoryID"`
}

func (ProductModel) TableName() string {
	return "products"
}

func (m *ProductModel) ToDomain() *domain.Product {
	product := &domain.Product{
		ID:            m.ID,
		CategoryID:    m.CategoryID,
		Name:          m.Name,
		Description:   m.Description,
		BasePrice:     m.BasePrice,
		Slug:          m.Slug,
		GSMInfo:       m.GSMInfo,
		FabricSummary: m.FabricSummary,
		Rating:        m.Rating,
		SoldCount:     m.SoldCount,
		ReviewCount:   m.ReviewCount,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}

	// Unmarshal KeyFeatures JSON string ke []string
	if m.KeyFeatures != "" {
		var features []string
		if err := json.Unmarshal([]byte(m.KeyFeatures), &features); err == nil {
			product.KeyFeatures = features
		}
	}

	if m.Category != nil {
		product.Category = m.Category.ToDomain()
	}

	if len(m.Images) > 0 {
		var images []domain.ProductImage
		for _, img := range m.Images {
			images = append(images, img.ToDomain())
		}
		product.Images = images
	}

	if len(m.Fabrics) > 0 {
		var fabrics []domain.ProductFabric
		for _, fab := range m.Fabrics {
			fabrics = append(fabrics, fab.ToDomain())
		}
		product.Fabrics = fabrics
	}

	if len(m.Wholesale) > 0 {
		var wholesales []domain.WholesalePrice
		for _, w := range m.Wholesale {
			wholesales = append(wholesales, w.ToDomain())
		}
		product.Wholesale = wholesales
	}

	if m.DesignModel != nil {
		dm := m.DesignModel.ToDomain()
		product.DesignModel = &dm
	}

	return product
}

func FromProductDomain(d *domain.Product) *ProductModel {
	// Marshal []string KeyFeatures ke JSON String
	keyFeaturesJSON, _ := json.Marshal(d.KeyFeatures)
	if string(keyFeaturesJSON) == "null" {
		keyFeaturesJSON = []byte("[]")
	}

	model := &ProductModel{
		ID:            d.ID,
		CategoryID:    d.CategoryID,
		Name:          d.Name,
		Description:   d.Description,
		BasePrice:     d.BasePrice,
		Slug:          d.Slug,
		GSMInfo:       d.GSMInfo,
		FabricSummary: d.FabricSummary,
		Rating:        d.Rating,
		SoldCount:     d.SoldCount,
		ReviewCount:   d.ReviewCount,
		KeyFeatures:   string(keyFeaturesJSON),
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}

	if len(d.Images) > 0 {
		var imgModels []ProductImageModel
		for _, img := range d.Images {
			imgModels = append(imgModels, ProductImageModel{
				ID:        img.ID,
				ProductID: d.ID,
				ImageURL:  img.ImageURL,
				IsPrimary: img.IsPrimary,
			})
		}
		model.Images = imgModels
	}

	if len(d.Fabrics) > 0 {
		var fabModels []ProductFabricModel
		for _, fab := range d.Fabrics {
			var colorModels []FabricColorModel
			for _, c := range fab.Colors {
				fabricID := fab.ID
				colorModels = append(colorModels, FabricColorModel{
					ID:             c.ID,
					FabricID:       &fabricID,
					SpecTemplateID: c.SpecTemplateID,
					Name:           c.Name,
					HexCode:        c.HexCode,
				})
			}

			fabModels = append(fabModels, ProductFabricModel{
				ID:              fab.ID,
				ProductID:       d.ID,
				SpecTemplateID:  fab.SpecTemplateID, // Dihubungkan ke SpecTemplateID
				Name:            fab.Name,
				Description:     fab.Description,
				Composition:     fab.Composition,
				CareInstruction: fab.CareInstruction,
				BasePrice:       fab.BasePrice,
				PriceAdjustment: fab.PriceAdjustment,
				IsDefault:       fab.IsDefault,
				Colors:          colorModels,
			})
		}
		model.Fabrics = fabModels
	}

	if len(d.Wholesale) > 0 {
		var wsModels []WholesalePriceModel
		for _, w := range d.Wholesale {
			wsModels = append(wsModels, WholesalePriceModel{
				ID:        w.ID,
				ProductID: d.ID,
				FabricID:  w.FabricID,
				MinQty:    w.MinQty,
				MaxQty:    w.MaxQty,
				UnitPrice: w.UnitPrice,
			})
		}
		model.Wholesale = wsModels
	}

	if d.DesignModel != nil {
		var viewModels []ProductModelViewModel
		for _, v := range d.DesignModel.Views {
			viewModels = append(viewModels, ProductModelViewModel{
				ID:             v.ID,
				ProductModelID: d.DesignModel.ID,
				Side:           v.Side,
				ArtURL:         v.ArtURL,
				MaskURL:        v.MaskURL,
				Width:          v.Width,
				Height:         v.Height,
			})
		}

		model.DesignModel = &DesignerModel{
			ID:          d.DesignModel.ID,
			ProductID:   d.ID,
			Name:        d.DesignModel.Name,
			Type:        d.DesignModel.Type,
			Description: d.DesignModel.Description,
			Views:       viewModels,
		}
	}

	return model
}
