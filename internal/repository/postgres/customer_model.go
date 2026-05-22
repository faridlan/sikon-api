package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type CustomerModel struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name      string    `gorm:"type:varchar(255);not null"`
	Phone     string    `gorm:"type:varchar(50);not null"`
	Address   string    `gorm:"type:text"`
	CreatedBy string    `gorm:"type:uuid"` // Foreign key ke users.id
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	// Relasi (Preload)
	Creator *UserModel `gorm:"foreignKey:CreatedBy"`
}

func (CustomerModel) TableName() string {
	return "customers"
}

func (m *CustomerModel) ToDomain() *domain.Customer {
	customer := &domain.Customer{
		ID:        m.ID,
		Name:      m.Name,
		Phone:     m.Phone,
		Address:   m.Address,
		CreatedBy: m.CreatedBy,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	if m.Creator != nil {
		customer.Creator = m.Creator.ToDomain()
	}

	return customer
}

func FromCustomerDomain(d *domain.Customer) *CustomerModel {
	return &CustomerModel{
		ID:        d.ID,
		Name:      d.Name,
		Phone:     d.Phone,
		Address:   d.Address,
		CreatedBy: d.CreatedBy,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
