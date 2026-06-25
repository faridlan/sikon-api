package postgres

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type OrderItemModel struct {
	ID        string            `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrderID   string            `gorm:"type:uuid;not null"`
	ProductID string            `gorm:"type:uuid;not null"`
	Qty       int               `gorm:"not null"`
	Price     float64           `gorm:"type:decimal(12,2);not null"`
	Details   datatypes.JSONMap `gorm:"type:jsonb"` // Magic dari GORM
	CreatedAt time.Time         `gorm:"autoCreateTime"`
	UpdatedAt time.Time         `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt    `gorm:"index"`

	// Relasi
	Product *ProductModel `gorm:"foreignKey:ProductID"`
}

func (OrderItemModel) TableName() string {
	return "order_items"
}

type OrderModel struct {
	ID              string         `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrderNumber     string         `gorm:"type:varchar(100);unique;not null"`
	BatchPoID       *string        `gorm:"type:uuid;index"`
	CustomerID      string         `gorm:"type:uuid;not null"`
	SalesID         string         `gorm:"type:uuid;not null"`
	Subtotal        float64        `gorm:"type:decimal(15,2);not null;default:0"` // Kolom Baru
	DiscountAmount  float64        `gorm:"type:decimal(15,2);not null;default:0"` // Kolom Baru
	TaxPpn          float64        `gorm:"type:decimal(15,2);not null;default:0"` // Kolom Baru
	TaxPph          float64        `gorm:"type:decimal(15,2);not null;default:0"` // Kolom Baru
	TotalAmount     float64        `gorm:"type:decimal(15,2);not null"`
	ShippingCost    float64        `gorm:"type:decimal(12,2);not null"`
	CourierName     string         `gorm:"type:varchar(100)"`
	ShippingAddress string         `gorm:"type:text"`
	OrderStatus     string         `gorm:"type:varchar(50);not null"`
	PaymentStatus   string         `gorm:"type:varchar(50);not null"`
	ValidUntil      *time.Time     `gorm:"type:date"`
	TermsConditions string         `gorm:"type:text"`
	Notes           string         `gorm:"type:text"`
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	// Relasi
	Items    []OrderItemModel `gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Customer *CustomerModel   `gorm:"foreignKey:CustomerID"`
	Sales    *UserModel       `gorm:"foreignKey:SalesID"`
}

func (OrderModel) TableName() string {
	return "orders"
}

// --- Mapper Order Item ---
func (m *OrderItemModel) ToDomain() domain.OrderItem {

	item := domain.OrderItem{
		ID:        m.ID,
		OrderID:   m.OrderID,
		ProductID: m.ProductID,
		Qty:       m.Qty,
		Price:     m.Price,
		Details:   map[string]any(m.Details), // Konversi JSONB ke map biasa
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	if m.Product != nil {
		productDomain := m.Product.ToDomain()
		item.Product = productDomain
	}

	return item
}

// --- Mapper Order Utama ---
func (m *OrderModel) ToDomain() *domain.Order {
	batchPoID := ""
	if m.BatchPoID != nil {
		batchPoID = *m.BatchPoID
	}

	order := &domain.Order{
		ID:              m.ID,
		OrderNumber:     m.OrderNumber,
		BatchPoID:       batchPoID,
		CustomerID:      m.CustomerID,
		SalesID:         m.SalesID,
		Subtotal:        m.Subtotal,       // Mapping Baru
		DiscountAmount:  m.DiscountAmount, // Mapping Baru
		TaxPpn:          m.TaxPpn,         // Mapping Baru
		TaxPph:          m.TaxPph,         // Mapping Baru
		TotalAmount:     m.TotalAmount,
		ShippingCost:    m.ShippingCost,
		CourierName:     m.CourierName,
		ShippingAddress: m.ShippingAddress,
		OrderStatus:     domain.OrderStatus(m.OrderStatus),
		PaymentStatus:   domain.PaymentStatus(m.PaymentStatus),
		ValidUntil:      m.ValidUntil,
		TermsConditions: m.TermsConditions,
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

func FromOrderDomain(d *domain.Order) *OrderModel {
	var batchPoID *string
	if d.BatchPoID != "" {
		batchPoID = &d.BatchPoID
	}

	model := &OrderModel{
		ID:              d.ID,
		OrderNumber:     d.OrderNumber,
		BatchPoID:       batchPoID,
		CustomerID:      d.CustomerID,
		SalesID:         d.SalesID,
		Subtotal:        d.Subtotal,       // Mapping Baru
		DiscountAmount:  d.DiscountAmount, // Mapping Baru
		TaxPpn:          d.TaxPpn,         // Mapping Baru
		TaxPph:          d.TaxPph,         // Mapping Baru
		TotalAmount:     d.TotalAmount,
		ShippingCost:    d.ShippingCost,
		CourierName:     d.CourierName,
		ShippingAddress: d.ShippingAddress,
		OrderStatus:     string(d.OrderStatus),
		PaymentStatus:   string(d.PaymentStatus),
		ValidUntil:      d.ValidUntil,
		TermsConditions: d.TermsConditions,
		Notes:           d.Notes,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}

	// Mapping detail items
	if len(d.Items) > 0 {
		for _, item := range d.Items {
			model.Items = append(model.Items, OrderItemModel{
				ID:        item.ID,
				OrderID:   item.OrderID,
				ProductID: item.ProductID,
				Qty:       item.Qty,
				Price:     item.Price,
				Details:   datatypes.JSONMap(item.Details), // Konversi map biasa ke JSONB
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			})
		}
	}
	return model
}
