package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type ProductModel struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	CategoryID  string    `gorm:"type:uuid;not null"`
	Name        string    `gorm:"type:varchar(255);not null"`
	Description string    `gorm:"type:text"`
	BasePrice   float64   `gorm:"type:decimal(12,2);not null;default:0"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`

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

	return product
}

func FromProductDomain(d *domain.Product) *ProductModel {
	return &ProductModel{
		ID:          d.ID,
		CategoryID:  d.CategoryID,
		Name:        d.Name,
		Description: d.Description,
		BasePrice:   d.BasePrice,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}
