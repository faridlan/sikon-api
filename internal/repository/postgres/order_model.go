package postgres

import (
	"encoding/json"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type OrderItemModel struct {
	ID         string         `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrderID    string         `gorm:"type:uuid;not null"`
	ProductID  string         `gorm:"type:uuid;not null"`
	CustomName string         `gorm:"type:varchar(255)"`
	Qty        int            `gorm:"not null"`
	Price      float64        `gorm:"type:decimal(12,2);not null"`
	Details    datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`

	Product *ProductModel `gorm:"foreignKey:ProductID"`
}

func (OrderItemModel) TableName() string {
	return "order_items"
}

type OrderModel struct {
	ID                string         `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrderNumber       string         `gorm:"type:varchar(100);unique;not null"`
	BatchPoID         *string        `gorm:"type:uuid;index"`
	CustomerID        string         `gorm:"type:uuid;not null"`
	SalesID           string         `gorm:"type:uuid;not null"`
	Subtotal          float64        `gorm:"type:decimal(15,2);not null;default:0"`
	DiscountAmount    float64        `gorm:"type:decimal(15,2);not null;default:0"`
	IsTaxable         bool           `gorm:"type:boolean;not null;default:false"`      // 👈
	TaxPpnRate        float64        `gorm:"type:decimal(5,2);not null;default:12.00"` // 👈
	TaxPph22Rate      float64        `gorm:"type:decimal(5,2);not null;default:1.50"`  // 👈
	DppPpn            float64        `gorm:"type:decimal(15,2);not null;default:0"`    // 👈
	DppPph            float64        `gorm:"type:decimal(15,2);not null;default:0"`    // 👈
	TaxPpn            float64        `gorm:"type:decimal(15,2);not null;default:0"`
	TaxPph            float64        `gorm:"type:decimal(15,2);not null;default:0"`
	TotalAmount       float64        `gorm:"type:decimal(15,2);not null"`
	PaguAmount        float64        `gorm:"type:decimal(15,2);not null;default:0"` // 👈
	NetReceivedAmount float64        `gorm:"type:decimal(15,2);not null;default:0"` // 👈
	TotalQty          int            `gorm:"->;column:total_qty"`
	ShippingCost      float64        `gorm:"type:decimal(12,2);not null"`
	CourierName       string         `gorm:"type:varchar(100)"`
	ShippingAddress   string         `gorm:"type:text"`
	OrderStatus       string         `gorm:"type:varchar(50);not null"`
	PaymentStatus     string         `gorm:"type:varchar(50);not null"`
	ValidUntil        *time.Time     `gorm:"type:date"`
	TermsConditions   string         `gorm:"type:text"`
	Notes             string         `gorm:"type:text"`
	CreatedAt         time.Time      `gorm:"autoCreateTime"`
	UpdatedAt         time.Time      `gorm:"autoUpdateTime"`
	DeletedAt         gorm.DeletedAt `gorm:"index"`
	ApprovedAt        *time.Time     `gorm:"column:approved_at"`

	Items    []OrderItemModel `gorm:"foreignKey:OrderID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Customer *CustomerModel   `gorm:"foreignKey:CustomerID"`
	Sales    *UserModel       `gorm:"foreignKey:SalesID"`
	BatchPO  *BatchPOModel    `gorm:"foreignKey:BatchPoID"`
}

func (OrderModel) TableName() string {
	return "orders"
}

func (m *OrderItemModel) ToDomain() domain.OrderItem {
	var detailsAny any
	if len(m.Details) > 0 {
		_ = json.Unmarshal(m.Details, &detailsAny)
	}

	item := domain.OrderItem{
		ID:         m.ID,
		OrderID:    m.OrderID,
		ProductID:  m.ProductID,
		CustomName: m.CustomName,
		Qty:        m.Qty,
		Price:      m.Price,
		Details:    detailsAny,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}

	if m.Product != nil {
		item.Product = m.Product.ToDomain()
	}

	return item
}

func (m *OrderModel) ToDomain() *domain.Order {
	batchPoID := ""
	if m.BatchPoID != nil {
		batchPoID = *m.BatchPoID
	}

	order := &domain.Order{
		ID:                m.ID,
		OrderNumber:       m.OrderNumber,
		BatchPoID:         batchPoID,
		CustomerID:        m.CustomerID,
		SalesID:           m.SalesID,
		Subtotal:          m.Subtotal,
		DiscountAmount:    m.DiscountAmount,
		IsTaxable:         m.IsTaxable,
		TaxPpnRate:        m.TaxPpnRate,
		TaxPph22Rate:      m.TaxPph22Rate,
		DppPpn:            m.DppPpn,
		DppPph:            m.DppPph,
		TaxPpn:            m.TaxPpn,
		TaxPph:            m.TaxPph,
		TotalAmount:       m.TotalAmount,
		PaguAmount:        m.PaguAmount,
		NetReceivedAmount: m.NetReceivedAmount,
		TotalQty:          m.TotalQty,
		ShippingCost:      m.ShippingCost,
		CourierName:       m.CourierName,
		ShippingAddress:   m.ShippingAddress,
		OrderStatus:       domain.OrderStatus(m.OrderStatus),
		PaymentStatus:     domain.PaymentStatus(m.PaymentStatus),
		ValidUntil:        m.ValidUntil,
		TermsConditions:   m.TermsConditions,
		Notes:             m.Notes,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
		ApprovedAt:        m.ApprovedAt,
	}

	if len(m.Items) > 0 {
		for _, item := range m.Items {
			order.Items = append(order.Items, item.ToDomain())
		}
	}
	if m.Customer != nil {
		order.Customer = m.Customer.ToDomain()
	}
	if m.Sales != nil {
		order.Sales = m.Sales.ToDomain()
	}
	if m.BatchPO != nil {
		order.BatchPO = m.BatchPO.ToDomain()
	}

	return order
}

func FromOrderDomain(d *domain.Order) *OrderModel {
	var batchPoID *string
	if d.BatchPoID != "" {
		batchPoID = &d.BatchPoID
	}

	model := &OrderModel{
		ID:                d.ID,
		OrderNumber:       d.OrderNumber,
		BatchPoID:         batchPoID,
		CustomerID:        d.CustomerID,
		SalesID:           d.SalesID,
		Subtotal:          d.Subtotal,
		DiscountAmount:    d.DiscountAmount,
		IsTaxable:         d.IsTaxable,
		TaxPpnRate:        d.TaxPpnRate,
		TaxPph22Rate:      d.TaxPph22Rate,
		DppPpn:            d.DppPpn,
		DppPph:            d.DppPph,
		TaxPpn:            d.TaxPpn,
		TaxPph:            d.TaxPph,
		TotalAmount:       d.TotalAmount,
		PaguAmount:        d.PaguAmount,
		NetReceivedAmount: d.NetReceivedAmount,
		ShippingCost:      d.ShippingCost,
		CourierName:       d.CourierName,
		ShippingAddress:   d.ShippingAddress,
		OrderStatus:       string(d.OrderStatus),
		PaymentStatus:     string(d.PaymentStatus),
		ValidUntil:        d.ValidUntil,
		TermsConditions:   d.TermsConditions,
		Notes:             d.Notes,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
		ApprovedAt:        d.ApprovedAt,
	}

	if len(d.Items) > 0 {
		for _, item := range d.Items {
			var detailsJSON datatypes.JSON
			if item.Details != nil {
				bytes, err := json.Marshal(item.Details)
				if err == nil {
					detailsJSON = datatypes.JSON(bytes)
				}
			}

			model.Items = append(model.Items, OrderItemModel{
				ID:         item.ID,
				OrderID:    item.OrderID,
				ProductID:  item.ProductID,
				CustomName: item.CustomName,
				Qty:        item.Qty,
				Price:      item.Price,
				Details:    detailsJSON,
				CreatedAt:  item.CreatedAt,
				UpdatedAt:  item.UpdatedAt,
			})
		}
	}
	return model
}
