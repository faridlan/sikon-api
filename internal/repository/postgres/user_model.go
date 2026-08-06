package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type UserModel struct {
	ID         string         `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name       string         `gorm:"type:varchar(255);not null"`
	Email      string         `gorm:"type:varchar(255);unique;not null"`
	Password   string         `gorm:"type:varchar(255);not null"`
	Role       string         `gorm:"type:varchar(50);not null"`
	ImageURL   string         `gorm:"type:varchar(255)"`
	Phone      string         `gorm:"type:varchar(50)"`
	StatusText string         `gorm:"type:varchar(100);default:'Online sekarang'"`
	IsActive   bool           `gorm:"default:true"`
	SortOrder  int            `gorm:"default:0"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (UserModel) TableName() string {
	return "users"
}

func (m *UserModel) ToDomain() *domain.User {
	return &domain.User{
		ID:         m.ID,
		Name:       m.Name,
		Email:      m.Email,
		Password:   m.Password,
		Role:       domain.Role(m.Role),
		ImageURL:   m.ImageURL,
		Phone:      m.Phone,
		StatusText: m.StatusText,
		IsActive:   m.IsActive,
		SortOrder:  m.SortOrder,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

func FromUserDomain(d *domain.User) *UserModel {
	return &UserModel{
		ID:         d.ID,
		Name:       d.Name,
		Email:      d.Email,
		Password:   d.Password,
		Role:       string(d.Role),
		ImageURL:   d.ImageURL,
		Phone:      d.Phone,
		StatusText: d.StatusText,
		IsActive:   d.IsActive,
		SortOrder:  d.SortOrder,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}
