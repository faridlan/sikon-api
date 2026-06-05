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
	SalesID   string
	CreatedAt time.Time
	UpdatedAt time.Time

	Creator *User
	Sales   *User
}

// Input Struct
type CustomerCreateInput struct {
	Name      string
	Phone     string
	Address   string
	CreatedBy string
	SalesID   string
}

type CustomerUpdateInput struct {
	Name    string
	Phone   string
	Address string
	SalesID string
}

type CustomerRepository interface {
	Create(ctx context.Context, customer *Customer) error
	GetByID(ctx context.Context, id string) (*Customer, error)
	Fetch(ctx context.Context, limit, offset int, filterSalesID string) ([]Customer, int64, error)
	Update(ctx context.Context, customer *Customer) error
	Delete(ctx context.Context, id string) error
}

type CustomerUsecase interface {
	CreateCustomer(ctx context.Context, input CustomerCreateInput) (*Customer, error)
	GetCustomer(ctx context.Context, id string, operatorID string, operatorRole Role) (*Customer, error)
	ListCustomers(ctx context.Context, query PaginationQuery, requestedSalesID, operatorID string, operatorRole Role) ([]Customer, PaginationMeta, error)
	UpdateCustomer(ctx context.Context, id string, input CustomerUpdateInput) (*Customer, error)
	DeleteCustomer(ctx context.Context, id string) error
}
