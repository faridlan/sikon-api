package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

// ==========================================
// MODEL: MATERIAL
// ==========================================
type MaterialModel struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name      string         `gorm:"type:varchar(255);not null"`
	Unit      string         `gorm:"type:varchar(50);not null"`
	UnitPrice float64        `gorm:"type:numeric(15,2);not null"`
	Category  string         `gorm:"type:varchar(100)"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (MaterialModel) TableName() string {
	return "materials"
}

func (m *MaterialModel) ToDomain() *domain.Material {
	return &domain.Material{
		ID:        m.ID,
		Name:      m.Name,
		Unit:      m.Unit,
		UnitPrice: m.UnitPrice,
		Category:  m.Category,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func FromMaterialDomain(d *domain.Material) *MaterialModel {
	return &MaterialModel{
		ID:        d.ID,
		Name:      d.Name,
		Unit:      d.Unit,
		UnitPrice: d.UnitPrice,
		Category:  d.Category,
	}
}

// ==========================================
// MODEL: PRODUCT MATERIAL (Resep / BOM)
// ==========================================
type ProductMaterialModel struct {
	ID         string  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	ProductID  string  `gorm:"type:uuid;not null;index"`
	MaterialID string  `gorm:"type:uuid;not null;index"`
	QtyPerUnit float64 `gorm:"type:numeric(15,4);not null"`

	// Relasi
	Material MaterialModel `gorm:"foreignKey:MaterialID"`
}

func (ProductMaterialModel) TableName() string {
	return "product_materials"
}

func (m *ProductMaterialModel) ToDomain() *domain.ProductMaterial {
	pm := &domain.ProductMaterial{
		ID:         m.ID,
		ProductID:  m.ProductID,
		MaterialID: m.MaterialID,
		QtyPerUnit: m.QtyPerUnit,
	}

	// Jika Material di-preload dan datanya ada
	if m.Material.ID != "" {
		pm.Material = m.Material.ToDomain()
	}

	return pm
}
