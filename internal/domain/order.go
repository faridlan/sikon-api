package domain

import (
	"context"
	"time"
)

type OrderStatus string
type PaymentStatus string

const (
	OrderStatusQuotation  OrderStatus = "quotation"
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusProduction OrderStatus = "production"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusCanceled   OrderStatus = "canceled"

	PaymentStatusUnpaid  PaymentStatus = "unpaid"
	PaymentStatusPartial PaymentStatus = "partial"
	PaymentStatusPaid    PaymentStatus = "paid"
)

type OrderItem struct {
	ID        string
	OrderID   string
	ProductID string
	Qty       int
	Price     float64
	// Details untuk menyimpan JSON variasi (misal: S: 10, M: 20)
	Details   map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time

	Product *Product
}

type Order struct {
	ID              string
	OrderNumber     string
	CustomerID      string
	SalesID         string
	Subtotal        float64
	DiscountAmount  float64
	TaxPpn          float64
	TaxPph          float64
	TotalAmount     float64
	ShippingCost    float64
	CourierName     string
	ShippingAddress string
	OrderStatus     OrderStatus
	PaymentStatus   PaymentStatus
	ValidUntil      *time.Time
	TermsConditions string
	Notes           string
	CreatedAt       time.Time
	UpdatedAt       time.Time

	// Relasi
	Items    []OrderItem
	Customer *Customer
	Sales    *User
}

type OrderItemInput struct {
	ProductID string
	Qty       int
	Price     float64
	Details   map[string]any
}

type OrderCreateInput struct {
	CustomerID      string
	SalesID         string
	ShippingCost    float64
	CourierName     string
	ShippingAddress string
	ValidUntil      *time.Time
	TermsConditions string
	Notes           string
	OrderStatus     OrderStatus
	Items           []OrderItemInput
}

type OrderUpdateInput struct {
	ShippingCost    float64
	CourierName     string
	ShippingAddress string
	ValidUntil      *time.Time
	TermsConditions string
	Notes           string
}

type OrderFilter struct {
	Search        string // Untuk pencarian OrderNumber
	CustomerID    string // Filter by Customer
	SalesID       string // Filter by Sales
	OrderStatus   string // Filter by Order Status
	PaymentStatus string // Filter by Payment Status
	StartDate     string // Filter dari tanggal (format: YYYY-MM-DD)
	EndDate       string // Filter sampai tanggal (format: YYYY-MM-DD)
}

type OrderRepository interface {
	Create(ctx context.Context, order *Order) error
	GetByID(ctx context.Context, id string) (*Order, error)
	Fetch(ctx context.Context, filter OrderFilter, limit, offset int) ([]Order, int64, error)
	Update(ctx context.Context, order *Order) error
	Delete(ctx context.Context, id string) error

	// Custom Query
	UpdateStatus(ctx context.Context, id string, orderStatus OrderStatus, paymentStatus PaymentStatus) error
}

type OrderUsecase interface {
	CreateOrder(ctx context.Context, input OrderCreateInput) (*Order, error)
	GetOrder(ctx context.Context, id string) (*Order, error)
	ListOrders(ctx context.Context, filter OrderFilter, query PaginationQuery) ([]Order, PaginationMeta, error)
	UpdateOrder(ctx context.Context, id string, input OrderUpdateInput) (*Order, error)
	DeleteOrder(ctx context.Context, id string) error
	UpdateOrderStatus(ctx context.Context, id string, status OrderStatus) error
	UpdatePaymentStatus(ctx context.Context, id string, status PaymentStatus) error
}
