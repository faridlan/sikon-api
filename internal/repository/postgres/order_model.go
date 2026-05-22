package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type OrderItemModel struct {
	ID        string                 `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrderID   string                 `gorm:"type:uuid;not null"`
	ProductID string                 `gorm:"type:uuid;not null"`
	Qty       int                    `gorm:"not null"`
	Price     float64                `gorm:"type:decimal(12,2);not null"`
	Details   map[string]interface{} `gorm:"type:jsonb;serializer:json"` // Magic dari GORM
	CreatedAt time.Time              `gorm:"autoCreateTime"`
	UpdatedAt time.Time              `gorm:"autoUpdateTime"`
}

func (OrderItemModel) TableName() string {
	return "order_items"
}

type OrderModel struct {
	ID              string    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrderNumber     string    `gorm:"type:varchar(100);unique;not null"`
	CustomerID      string    `gorm:"type:uuid;not null"`
	SalesID         string    `gorm:"type:uuid;not null"`
	TotalAmount     float64   `gorm:"type:decimal(15,2);not null"`
	ShippingCost    float64   `gorm:"type:decimal(12,2);not null"`
	CourierName     string    `gorm:"type:varchar(100)"`
	ShippingAddress string    `gorm:"type:text"`
	OrderStatus     string    `gorm:"type:varchar(50);not null"`
	PaymentStatus   string    `gorm:"type:varchar(50);not null"`
	Notes           string    `gorm:"type:text"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`

	// Relasi
	Items    []OrderItemModel `gorm:"foreignKey:OrderID"`
	Customer *CustomerModel   `gorm:"foreignKey:CustomerID"`
	Sales    *UserModel       `gorm:"foreignKey:SalesID"`
}

func (OrderModel) TableName() string {
	return "orders"
}

// --- Mapper Order Item ---
func (m *OrderItemModel) ToDomain() domain.OrderItem {
	return domain.OrderItem{
		ID:        m.ID,
		OrderID:   m.OrderID,
		ProductID: m.ProductID,
		Qty:       m.Qty,
		Price:     m.Price,
		Details:   m.Details,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// --- Mapper Order Utama ---
func (m *OrderModel) ToDomain() *domain.Order {
	order := &domain.Order{
		ID:              m.ID,
		OrderNumber:     m.OrderNumber,
		CustomerID:      m.CustomerID,
		SalesID:         m.SalesID,
		TotalAmount:     m.TotalAmount,
		ShippingCost:    m.ShippingCost,
		CourierName:     m.CourierName,
		ShippingAddress: m.ShippingAddress,
		OrderStatus:     domain.OrderStatus(m.OrderStatus),
		PaymentStatus:   domain.PaymentStatus(m.PaymentStatus),
		Notes:           m.Notes,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}

	// Map relasi Items
	if len(m.Items) > 0 {
		for _, item := range m.Items {
			order.Items = append(order.Items, item.ToDomain())
		}
	}
	// Map Customer
	if m.Customer != nil {
		order.Customer = m.Customer.ToDomain()
	}
	// Map Sales
	if m.Sales != nil {
		order.Sales = m.Sales.ToDomain()
	}

	return order
}

// (Untuk fungsi FromOrderDomain bisa ditambahkan mirip seperti yang User)
