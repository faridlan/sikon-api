package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type SpecTemplateModel struct {
	ID              string             `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name            string             `gorm:"type:varchar(255);not null"`
	Spec            string             `gorm:"type:text;not null"`
	Description     string             `gorm:"type:text"`
	Composition     string             `gorm:"type:varchar(255)"`
	CareInstruction string             `gorm:"type:text"`
	CreatedAt       time.Time          `gorm:"autoCreateTime"`
	UpdatedAt       time.Time          `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt     `gorm:"index"`
	Colors          []FabricColorModel `gorm:"foreignKey:SpecTemplateID;constraint:OnDelete:CASCADE;"`
}

func (SpecTemplateModel) TableName() string {
	return "spec_templates"
}

func (m *SpecTemplateModel) ToDomain() *domain.SpecTemplate {
	specTemplate := &domain.SpecTemplate{
		ID:              m.ID,
		Name:            m.Name,
		Spec:            m.Spec,
		Description:     m.Description,
		Composition:     m.Composition,
		CareInstruction: m.CareInstruction,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}

	if len(m.Colors) > 0 {
		var colors []domain.FabricColor
		for _, c := range m.Colors {
			colors = append(colors, c.ToDomain())
		}
		specTemplate.Colors = colors
	}

	return specTemplate
}

func FromSpecTemplateDomain(d *domain.SpecTemplate) *SpecTemplateModel {
	model := &SpecTemplateModel{
		ID:              d.ID,
		Name:            d.Name,
		Spec:            d.Spec,
		Description:     d.Description,
		Composition:     d.Composition,
		CareInstruction: d.CareInstruction,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}

	if len(d.Colors) > 0 {
		var colors []FabricColorModel
		for _, c := range d.Colors {
			var fabricIDPtr *string
			if c.FabricID != "" {
				fabricID := c.FabricID
				fabricIDPtr = &fabricID
			}

			colors = append(colors, FabricColorModel{
				ID:             c.ID,
				FabricID:       fabricIDPtr,
				SpecTemplateID: c.SpecTemplateID,
				Name:           c.Name,
				HexCode:        c.HexCode,
				CreatedAt:      c.CreatedAt,
			})
		}
		model.Colors = colors
	}

	return model
}
