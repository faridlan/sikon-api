package domain

import (
	"context"
	"time"
)

type PaymentType string

const (
	PaymentTypeDP          PaymentType = "dp"
	PaymentTypeSettlement  PaymentType = "settlement"
	PaymentTypeInstallment PaymentType = "installment"
)

type Payment struct {
	ID              string
	OrderID         string
	BankAccountID   string
	Amount          float64
	PaymentDate     time.Time
	ReferenceNumber string
	PaymentType     PaymentType
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type PaymentCreateInput struct {
	OrderID         string
	BankAccountID   string
	Amount          float64
	PaymentDate     time.Time
	ReferenceNumber string
	PaymentType     PaymentType
}

type PaymentUpdateInput struct {
	ReferenceNumber string
	PaymentType     PaymentType
}

type PaymentRepository interface {
	Create(ctx context.Context, payment *Payment) error
	GetByID(ctx context.Context, id string) (*Payment, error)
	Fetch(ctx context.Context, limit, offset int) ([]Payment, int64, error)
	Update(ctx context.Context, payment *Payment) error
	Delete(ctx context.Context, id string) error

	// Custom Query
	GetByOrderID(ctx context.Context, orderID string) ([]Payment, error)
}

type PaymentUsecase interface {
	ProcessPayment(ctx context.Context, input PaymentCreateInput) error
	GetPayment(ctx context.Context, id string) (*Payment, error)
	ListPayments(c context.Context, query PaginationQuery) ([]Payment, PaginationMeta, error)
	UpdatePayment(ctx context.Context, id string, input PaymentUpdateInput) error
	DeletePayment(ctx context.Context, id string) error
	GetPaymentsByOrderID(ctx context.Context, orderID string) ([]Payment, error)
}
