package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type SpecTemplateModel struct {
	ID              string         `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name            string         `gorm:"type:varchar(255);not null"`
	Spec            string         `gorm:"type:text;not null"`
	Description     string         `gorm:"type:text"`
	Composition     string         `gorm:"type:varchar(255)"`
	CareInstruction string         `gorm:"type:text"`
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (SpecTemplateModel) TableName() string {
	return "spec_templates"
}

func (m *SpecTemplateModel) ToDomain() *domain.SpecTemplate {
	return &domain.SpecTemplate{
		ID:              m.ID,
		Name:            m.Name,
		Spec:            m.Spec,
		Description:     m.Description,
		Composition:     m.Composition,
		CareInstruction: m.CareInstruction,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func FromSpecTemplateDomain(d *domain.SpecTemplate) *SpecTemplateModel {
	return &SpecTemplateModel{
		ID:              d.ID,
		Name:            d.Name,
		Spec:            d.Spec,
		Description:     d.Description,
		Composition:     d.Composition,
		CareInstruction: d.CareInstruction,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}
}
