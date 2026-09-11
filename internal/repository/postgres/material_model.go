package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==========================================
// MODEL: MATERIAL
// ==========================================
type MaterialModel struct {
	ID              string             `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name            string             `gorm:"type:varchar(255);not null"`
	Unit            string             `gorm:"type:varchar(50);not null"`
	UnitPrice       float64            `gorm:"type:numeric(15,2);not null"`
	Category        string             `gorm:"type:varchar(100)"`
	Description     string             `gorm:"type:text"`
	Composition     string             `gorm:"type:varchar(255)"`
	CareInstruction string             `gorm:"type:text"`
	GSMInfo         string             `gorm:"type:varchar(100)"`
	CreatedAt       time.Time          `gorm:"autoCreateTime"`
	UpdatedAt       time.Time          `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt     `gorm:"index"`
	Colors          []FabricColorModel `gorm:"foreignKey:MaterialID;constraint:OnDelete:CASCADE;"`
}

func (m *MaterialModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

func (MaterialModel) TableName() string {
	return "materials"
}

func (m *MaterialModel) ToDomain() *domain.Material {
	mat := &domain.Material{
		ID:              m.ID,
		Name:            m.Name,
		Unit:            m.Unit,
		UnitPrice:       m.UnitPrice,
		Category:        m.Category,
		Description:     m.Description,
		Composition:     m.Composition,
		CareInstruction: m.CareInstruction,
		GSMInfo:         m.GSMInfo,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
	if len(m.Colors) > 0 {
		mat.Colors = make([]domain.FabricColor, len(m.Colors))
		for i, c := range m.Colors {
			mat.Colors[i] = c.ToDomain()
		}
	}
	return mat
}

func FromMaterialDomain(d *domain.Material) *MaterialModel {
	return &MaterialModel{
		ID:              d.ID,
		Name:            d.Name,
		Unit:            d.Unit,
		UnitPrice:       d.UnitPrice,
		Category:        d.Category,
		Description:     d.Description,
		Composition:     d.Composition,
		CareInstruction: d.CareInstruction,
		GSMInfo:         d.GSMInfo,
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

func (m *ProductMaterialModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
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
