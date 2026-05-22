package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// UserModel merepresentasikan tabel "users" di database
type UserModel struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name      string    `gorm:"type:varchar(255);not null"`
	Email     string    `gorm:"type:varchar(255);unique;not null"`
	Password  string    `gorm:"type:varchar(255);not null"`
	Role      string    `gorm:"type:varchar(50);not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName memberitahu GORM nama tabel pastinya
func (UserModel) TableName() string {
	return "users"
}

// ToDomain mengubah data dari Database menjadi entitas murni Bisnis (Domain)
func (m *UserModel) ToDomain() *domain.User {
	return &domain.User{
		ID:        m.ID,
		Name:      m.Name,
		Email:     m.Email,
		Password:  m.Password,
		Role:      domain.Role(m.Role),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// FromUserDomain mengubah data dari entitas Bisnis menjadi format Database
func FromUserDomain(d *domain.User) *UserModel {
	return &UserModel{
		ID:        d.ID,
		Name:      d.Name,
		Email:     d.Email,
		Password:  d.Password,
		Role:      string(d.Role),
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}
