package domain

import (
	"context"
	"time"
)

type Role string

const (
	RoleOwner Role = "owner"
	// RoleAdmin      Role = "admin"
	RoleAccounting Role = "accounting"
	RoleSales      Role = "sales"
)

type User struct {
	ID         string
	Name       string
	Email      string
	Password   string
	Role       Role
	ImageURL   string
	Phone      string // Nomor WhatsApp (e.g. "6281200000001")
	StatusText string // Label status (e.g. "Online sekarang")
	IsActive   bool   // Status keaktifan
	SortOrder  int    // Urutan prioritas rekomendasi
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type UserFilter struct {
	Search   string
	Role     string
	IsActive *bool
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Fetch(ctx context.Context, limit, offset int, filter UserFilter) ([]User, int64, error)
	GetPublicSalesList(ctx context.Context) ([]User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id string) error
}

type UserRegisterInput struct {
	Name       string
	Email      string
	Password   string
	Role       Role
	ImageURL   string
	Phone      string
	StatusText string
	IsActive   bool
	SortOrder  int
}

type UserUpdateInput struct {
	Name       string
	Role       Role
	ImageURL   string
	Phone      string
	StatusText string
	IsActive   *bool
	SortOrder  *int
}

type UserUsecase interface {
	Register(ctx context.Context, input UserRegisterInput) (*User, error)
	GetProfile(ctx context.Context, userID string) (*User, error)
	ListUsers(c context.Context, query PaginationQuery, filter UserFilter) ([]User, PaginationMeta, error)
	GetPublicSalesList(ctx context.Context) ([]User, error)
	UpdateUser(ctx context.Context, id string, input UserUpdateInput) (*User, error)
	DeleteUser(ctx context.Context, id string) error
}
