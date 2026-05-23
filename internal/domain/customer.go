package domain

import (
	"context"
	"time"
)

type Customer struct {
	ID        string
	Name      string
	Phone     string
	Address   string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time

	Creator *User
}

// Input Struct
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

type CustomerRepository interface {
	Create(ctx context.Context, customer *Customer) error
	GetByID(ctx context.Context, id string) (*Customer, error)
	Fetch(ctx context.Context, limit, offset int) ([]Customer, int64, error)
	Update(ctx context.Context, customer *Customer) error
	Delete(ctx context.Context, id string) error
}

type CustomerUsecase interface {
	CreateCustomer(ctx context.Context, input CustomerCreateInput) (*Customer, error)
	GetCustomer(ctx context.Context, id string) (*Customer, error)
	ListCustomers(c context.Context, query PaginationQuery) ([]Customer, PaginationMeta, error)
	UpdateCustomer(ctx context.Context, id string, input CustomerUpdateInput) (*Customer, error)
	DeleteCustomer(ctx context.Context, id string) error
}
