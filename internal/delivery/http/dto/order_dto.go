package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type OrderItemRequest struct {
	ProductID string                 `json:"product_id" validate:"required,uuid"`
	Qty       int                    `json:"qty" validate:"required,gt=0"`
	Details   map[string]interface{} `json:"details"`
}

type OrderCreateRequest struct {
	CustomerID      string             `json:"customer_id" validate:"required,uuid"`
	SalesID         string             `json:"sales_id" validate:"required,uuid"`
	ShippingCost    float64            `json:"shipping_cost"`
	CourierName     string             `json:"courier_name"`
	ShippingAddress string             `json:"shipping_address"`
	Notes           string             `json:"notes"`
	Items           []OrderItemRequest `json:"items" validate:"required,min=1"`
}

type OrderResponse struct {
	ID            string    `json:"id"`
	OrderNumber   string    `json:"order_number"`
	TotalAmount   float64   `json:"total_amount"`
	PaymentStatus string    `json:"payment_status"`
	OrderStatus   string    `json:"order_status"`
	CreatedAt     time.Time `json:"created_at"`
}

func ToOrderResponse(o *domain.Order) OrderResponse {
	return OrderResponse{
		ID:            o.ID,
		OrderNumber:   o.OrderNumber,
		TotalAmount:   o.TotalAmount,
		PaymentStatus: string(o.PaymentStatus),
		OrderStatus:   string(o.OrderStatus),
		CreatedAt:     o.CreatedAt,
	}
}
