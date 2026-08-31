package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type WorkerModel struct {
	ID         string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID     *string        `gorm:"type:uuid"` // 👈 FK ke User (Nullable)
	Name       string         `gorm:"type:varchar(255);not null"`
	Phone      string         `gorm:"type:varchar(50)"`
	Role       string         `gorm:"type:varchar(50);not null"`
	SalaryType string         `gorm:"type:varchar(50);not null"`
	DailyRate  float64        `gorm:"type:numeric(15,2);not null;default:0"` // 👈 Daily Rate
	Status     string         `gorm:"type:varchar(20);default:'active'"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`

	// Relasi
	User *UserModel `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"` // 👈 Relasi ke UserModel
}

func (WorkerModel) TableName() string {
	return "workers"
}

func (m *WorkerModel) ToDomain() *domain.Worker {
	worker := &domain.Worker{
		ID:         m.ID,
		UserID:     m.UserID,
		Name:       m.Name,
		Phone:      m.Phone,
		Role:       domain.WorkerRole(m.Role),
		SalaryType: domain.WorkerSalaryType(m.SalaryType),
		DailyRate:  m.DailyRate,
		Status:     domain.WorkerStatus(m.Status),
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}

	if m.User != nil {
		worker.User = m.User.ToDomain()
	}

	return worker
}

func FromWorkerDomain(d *domain.Worker) *WorkerModel {
	return &WorkerModel{
		ID:         d.ID,
		UserID:     d.UserID,
		Name:       d.Name,
		Phone:      d.Phone,
		Role:       string(d.Role),
		SalaryType: string(d.SalaryType),
		DailyRate:  d.DailyRate,
		Status:     string(d.Status),
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}
