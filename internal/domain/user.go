package domain

import (
	"context"
	"time"
)

// Enum untuk Role
type Role string

const (
	RoleAdmin Role = "admin"
	RoleSales Role = "sales"
)

// User Entity (Tanpa tag GORM, murni untuk layer domain)
type User struct {
	ID        string
	Name      string
	Email     string
	Password  string
	Role      Role
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserFilter struct {
	Search string // Opsional: jika nanti butuh cari nama user
	Role   string // Filter spesifik untuk role (sales, admin, dll)
}

// UserRepository mendefinisikan kontrak untuk berinteraksi dengan database
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Fetch(ctx context.Context, limit, offset int, filter UserFilter) ([]User, int64, error)
	Update(ctx context.Context, user *User) error // Tambahan Update
	Delete(ctx context.Context, id string) error  // Tambahan Delete
}

// Input struct untuk Usecase
type UserRegisterInput struct {
	Name     string
	Email    string
	Password string
	Role     Role
}

type UserUpdateInput struct {
	Name string
	Role Role
	// Sengaja tidak memasukan Email dan Password di sini karena biasanya butuh flow khusus (seperti verifikasi)
}

// UserUsecase mendefinisikan kontrak untuk bisnis logika
type UserUsecase interface {
	Register(ctx context.Context, input UserRegisterInput) (*User, error)
	GetProfile(ctx context.Context, userID string) (*User, error)
	ListUsers(c context.Context, query PaginationQuery, filter UserFilter) ([]User, PaginationMeta, error)
	UpdateUser(ctx context.Context, id string, input UserUpdateInput) (*User, error) // Tambahan Update
	DeleteUser(ctx context.Context, id string) error                                 // Tambahan Delete
}
