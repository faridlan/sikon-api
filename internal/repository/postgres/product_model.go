package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type ProductModel struct {
	ID          string              `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	CategoryID  string              `gorm:"type:uuid;not null"`
	Name        string              `gorm:"type:varchar(255);not null"`
	Description string              `gorm:"type:text"`
	BasePrice   float64             `gorm:"type:decimal(12,2);not null;default:0"`
	Images      []ProductImageModel `gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE;"`
	CreatedAt   time.Time           `gorm:"autoCreateTime"`
	UpdatedAt   time.Time           `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt      `gorm:"index"`

	// Relasi
	Category *CategoryModel `gorm:"foreignKey:CategoryID"`
}

func (ProductModel) TableName() string {
	return "products"
}

func (m *ProductModel) ToDomain() *domain.Product {
	product := &domain.Product{
		ID:          m.ID,
		CategoryID:  m.CategoryID,
		Name:        m.Name,
		Description: m.Description,
		BasePrice:   m.BasePrice,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}

	// Mapping relasi jika preloaded
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

	return product
}

func FromProductDomain(d *domain.Product) *ProductModel {
	model := &ProductModel{
		ID:          d.ID,
		CategoryID:  d.CategoryID,
		Name:        d.Name,
		Description: d.Description,
		BasePrice:   d.BasePrice,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}

	// 🚨 TAMBAHKAN MAPPING GAMBAR
	if len(d.Images) > 0 {
		var imgModels []ProductImageModel
		for _, img := range d.Images {
			imgModels = append(imgModels, ProductImageModel{
				ID:        img.ID,
				ProductID: d.ID, // Pastikan ProductID ikut terisi
				ImageURL:  img.ImageURL,
				IsPrimary: img.IsPrimary,
			})
		}
		model.Images = imgModels
	}

	return model
}
