package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type ProductImageModel struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ProductID string    `gorm:"type:uuid;not null"`
	ImageURL  string    `gorm:"type:varchar(255);not null"`
	IsPrimary bool      `gorm:"default:false"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (ProductImageModel) TableName() string {
	return "product_images"
}

func (m *ProductImageModel) ToDomain() domain.ProductImage {
	return domain.ProductImage{
		ID:        m.ID,
		ProductID: m.ProductID,
		ImageURL:  m.ImageURL,
		IsPrimary: m.IsPrimary,
	}
}
