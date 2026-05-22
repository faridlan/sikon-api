package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type CategoryModel struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name      string    `gorm:"type:varchar(255);not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (CategoryModel) TableName() string {
	return "categories"
}

func (m *CategoryModel) ToDomain() *domain.Category {
	return &domain.Category{
		ID:        m.ID,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func FromCategoryDomain(d *domain.Category) *CategoryModel {
	return &CategoryModel{
		ID:        d.ID,
		Name:      d.Name,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
