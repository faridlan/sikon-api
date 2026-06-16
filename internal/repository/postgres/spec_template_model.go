package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type SpecTemplateModel struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name      string    `gorm:"type:varchar(255);not null"`
	Spec      string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (SpecTemplateModel) TableName() string {
	return "spec_templates"
}

func (m *SpecTemplateModel) ToDomain() *domain.SpecTemplate {
	return &domain.SpecTemplate{
		ID:        m.ID,
		Name:      m.Name,
		Spec:      m.Spec,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func FromSpecTemplateDomain(d *domain.SpecTemplate) *SpecTemplateModel {
	return &SpecTemplateModel{
		ID:        d.ID,
		Name:      d.Name,
		Spec:      d.Spec,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
