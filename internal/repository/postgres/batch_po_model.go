package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type BatchPOModel struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name      string         `gorm:"type:varchar(255);not null"`
	StartDate time.Time      `gorm:"type:date;not null"`
	EndDate   time.Time      `gorm:"type:date;not null"`
	Status    string         `gorm:"type:varchar(50);not null;default:'draft'"`
	Quota     int            `gorm:"type:int;default:0"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (BatchPOModel) TableName() string {
	return "batch_pos"
}

// ToDomain mengubah model database menjadi entitas bisnis (Domain)
func (m *BatchPOModel) ToDomain() *domain.BatchPO {
	return &domain.BatchPO{
		ID:        m.ID,
		Name:      m.Name,
		StartDate: m.StartDate,
		EndDate:   m.EndDate,
		Status:    domain.BatchPOStatus(m.Status),
		Quota:     m.Quota,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// FromBatchPODomain mengubah entitas bisnis menjadi model database
func FromBatchPODomain(d *domain.BatchPO) *BatchPOModel {
	return &BatchPOModel{
		ID:        d.ID,
		Name:      d.Name,
		StartDate: d.StartDate,
		EndDate:   d.EndDate,
		Status:    string(d.Status),
		Quota:     d.Quota,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
