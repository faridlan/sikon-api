package domain

import (
	"context"
	"time"
)

type PaymentType string
type PaymentVerificationStatus string

const (
	PaymentTypeDP          PaymentType = "dp"
	PaymentTypeSettlement  PaymentType = "settlement"
	PaymentTypeInstallment PaymentType = "installment"

	PaymentVerificationPending  PaymentVerificationStatus = "pending"
	PaymentVerificationVerified PaymentVerificationStatus = "verified"
	PaymentVerificationRejected PaymentVerificationStatus = "rejected"
)

type Payment struct {
	ID              string
	OrderID         string
	BankAccountID   string
	Amount          float64
	PaymentDate     time.Time
	ReferenceNumber string
	PaymentType     PaymentType
	Status          PaymentVerificationStatus // pending, verified, rejected
	VerifiedByID    *string
	VerifiedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time

	Order       *Order
	BankAccount *BankAccount
	VerifiedBy  *User
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

type PaymentVerifyInput struct {
	Status       PaymentVerificationStatus // verified atau rejected
	VerifiedByID string
}

type PaymentFilter struct {
	Search      string
	PaymentType string
	Status      string // Filter berdasarkan verification status
	StartDate   string
	EndDate     string
}

type PaymentRepository interface {
	Create(ctx context.Context, payment *Payment) error
	GetByID(ctx context.Context, id string) (*Payment, error)
	Fetch(ctx context.Context, limit, offset int, filter PaymentFilter) ([]Payment, int64, error)
	Update(ctx context.Context, payment *Payment) error
	Delete(ctx context.Context, id string) error

	// Custom Query
	GetByOrderID(ctx context.Context, orderID string) ([]Payment, error)
	UpdateVerificationStatus(ctx context.Context, paymentID string, status PaymentVerificationStatus, verifiedByID string, verifiedAt time.Time) error
}

type PaymentUsecase interface {
	ProcessPayment(ctx context.Context, input PaymentCreateInput) (*Payment, error)
	GetPayment(ctx context.Context, id string) (*Payment, error)
	ListPayments(c context.Context, query PaginationQuery, filter PaymentFilter) ([]Payment, PaginationMeta, error)
	UpdatePayment(ctx context.Context, id string, input PaymentUpdateInput) (*Payment, error)
	DeletePayment(ctx context.Context, id string) error
	GetPaymentsByOrderID(ctx context.Context, orderID string) ([]Payment, error)

	// BARU: Verifikasi pembayaran oleh Divisi Finance/Admin
	VerifyPayment(ctx context.Context, paymentID string, input PaymentVerifyInput) (*Payment, error)
}
