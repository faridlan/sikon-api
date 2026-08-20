package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type WorkerModel struct {
	ID         string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name       string         `gorm:"type:varchar(255);not null"`
	Phone      string         `gorm:"type:varchar(50)"`
	Role       string         `gorm:"type:varchar(50);not null"`
	SalaryType string         `gorm:"type:varchar(50);not null"`
	Status     string         `gorm:"type:varchar(20);default:'active'"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (WorkerModel) TableName() string {
	return "workers"
}

func (m *WorkerModel) ToDomain() *domain.Worker {
	return &domain.Worker{
		ID:         m.ID,
		Name:       m.Name,
		Phone:      m.Phone,
		Role:       domain.WorkerRole(m.Role),
		SalaryType: domain.WorkerSalaryType(m.SalaryType),
		Status:     domain.WorkerStatus(m.Status),
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

func FromWorkerDomain(d *domain.Worker) *WorkerModel {
	return &WorkerModel{
		ID:         d.ID,
		Name:       d.Name,
		Phone:      d.Phone,
		Role:       string(d.Role),
		SalaryType: string(d.SalaryType),
		Status:     string(d.Status),
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}
