package domain

import (
	"context"
	"time"
)

// Customer Entity
type Customer struct {
	ID        string
	Name      string
	Phone     string
	Address   string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time

	// Opsional: Untuk preloading data User (Sales) yang mendaftarkan
	Creator *User
}

// CustomerRepository
type CustomerRepository interface {
	Create(ctx context.Context, customer *Customer) error
	GetByID(ctx context.Context, id string) (*Customer, error)
	Fetch(ctx context.Context, limit, offset int) ([]Customer, int64, error)
	Update(ctx context.Context, customer *Customer) error
	Delete(ctx context.Context, id string) error // Tambahan Delete
}

// Input struct untuk Usecase
type CustomerCreateInput struct {
	Name      string
	Phone     string
	Address   string
	CreatedBy string
}

type CustomerUpdateInput struct {
	Name    string
	Phone   string
	Address string
}

// CustomerUsecase
type CustomerUsecase interface {
	CreateCustomer(ctx context.Context, input CustomerCreateInput) (*Customer, error)
	GetCustomer(ctx context.Context, id string) (*Customer, error)
	ListCustomers(ctx context.Context, page, limit int) ([]Customer, int64, error)
	UpdateCustomer(ctx context.Context, id string, input CustomerUpdateInput) (*Customer, error) // Tambahan Update
	DeleteCustomer(ctx context.Context, id string) error                                         // Tambahan Delete
}
