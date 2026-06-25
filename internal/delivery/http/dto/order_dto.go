package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type OrderItemRequest struct {
	ProductID string  `json:"product_id" validate:"required,uuid" example:"999e4567-e89b-12d3-a456-426614174000"`
	Qty       int     `json:"qty" validate:"required,gt=0" example:"100"`
	Price     float64 `json:"price" validate:"omitempty" example:"35000"`

	// @Schema type object
	// @Schema example {"ukuran": "L", "warna": "Hitam"}
	Details map[string]any `json:"details"`
}

type OrderCreateRequest struct {
	BatchPoID       string             `json:"batch_po_id" validate:"required,uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	CustomerID      string             `json:"customer_id" validate:"required,uuid" example:"333e4567-e89b-12d3-a456-426614174000"`
	SalesID         string             `json:"sales_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ShippingCost    float64            `json:"shipping_cost" validate:"omitempty,gte=0" example:"50000"`
	CourierName     string             `json:"courier_name" validate:"omitempty" example:"JNE Trucking"`
	ShippingAddress string             `json:"shipping_address" validate:"omitempty" example:"Jl. Sudirman No. 123, Jakarta"`
	ValidUntil      *time.Time         `json:"valid_until" validate:"omitempty" example:"2026-06-15T00:00:00Z"`
	TermsConditions string             `json:"terms_conditions" validate:"omitempty" example:"DP Minimal 50%, Waktu Pengerjaan 14 Hari"`
	Notes           string             `json:"notes" validate:"omitempty" example:"Tolong packing kayu"`
	OrderStatus     string             `json:"order_status" validate:"omitempty" example:"quotation"`
	Items           []OrderItemRequest `json:"items" validate:"required,min=1,dive"`
}

type OrderUpdateRequest struct {
	ShippingCost    float64    `json:"shipping_cost" validate:"omitempty,gte=0" example:"75000"`
	CourierName     string     `json:"courier_name" validate:"omitempty" example:"SiCepat Gokil"`
	ShippingAddress string     `json:"shipping_address" validate:"omitempty" example:"Jl. Sudirman No. 123, Jakarta Selatan"`
	ValidUntil      *time.Time `json:"valid_until" validate:"omitempty" example:"2026-06-15T00:00:00Z"`
	TermsConditions string     `json:"terms_conditions" validate:"omitempty" example:"DP Minimal 50%, Waktu Pengerjaan 14 Hari"`
	Notes           string     `json:"notes" validate:"omitempty" example:"Tambahan resi otomatis"`
}

type OrderStatusUpdateRequest struct {
	OrderStatus string `json:"order_status" validate:"required,oneof=quotation pending production ready completed canceled"`
}

type PaymentStatusUpdateRequest struct {
	PaymentStatus string `json:"payment_status" validate:"required,oneof=unpaid partial paid"`
}

// --- RESPONSE ---
type OrderItemResponse struct {
	ID        string           `json:"id" example:"item-uuid"`
	ProductID string           `json:"product_id" example:"999e4567-e89b-12d3-a456-426614174000"`
	Qty       int              `json:"qty" example:"100"`
	Price     float64          `json:"price" example:"35000"`
	Details   map[string]any   `json:"details"`
	Product   *ProductResponse `json:"product,omitempty"`
}

type OrderResponse struct {
	ID              string     `json:"id" example:"ord-uuid"`
	OrderNumber     string     `json:"order_number" example:"ORD-20231001-1234"`
	CustomerID      string     `json:"customer_id" example:"333e4567-e89b-12d3-a456-426614174000"`
	SalesID         string     `json:"sales_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Subtotal        float64    `json:"subtotal" example:"3500000"`
	DiscountAmount  float64    `json:"discount_amount" example:"0"`
	TaxPpn          float64    `json:"tax_ppn" example:"0"`
	TaxPph          float64    `json:"tax_ph" example:"0"`
	TotalAmount     float64    `json:"total_amount" example:"3550000"`
	ShippingCost    float64    `json:"shipping_cost" example:"50000"`
	CourierName     string     `json:"courier_name" example:"JNE Trucking"`
	ShippingAddress string     `json:"shipping_address" example:"Jl. Sudirman No. 123, Jakarta"`
	OrderStatus     string     `json:"order_status" example:"pending"`
	PaymentStatus   string     `json:"payment_status" example:"unpaid"`
	ValidUntil      *time.Time `json:"valid_until,omitempty" example:"2026-06-15T00:00:00Z"`
	TermsConditions string     `json:"terms_conditions,omitempty" example:"DP Minimal 50%"`
	Notes           string     `json:"notes" example:"Tolong packing kayu"`
	CreatedAt       time.Time  `json:"created_at" example:"2023-10-01T15:00:00Z"`
	UpdatedAt       time.Time  `json:"updated_at" example:"2023-10-01T15:00:00Z"`

	Items    []OrderItemResponse `json:"items,omitempty"`
	Customer *CustomerResponse   `json:"customer,omitempty"`
	Sales    *UserResponse       `json:"sales,omitempty"`
}

func ToOrderResponse(o *domain.Order) OrderResponse {
	// ... (Logika ToOrderResponse tetap sama persis seperti sebelumnya) ...
	resp := OrderResponse{
		ID:              o.ID,
		OrderNumber:     o.OrderNumber,
		CustomerID:      o.CustomerID,
		SalesID:         o.SalesID,
		Subtotal:        o.Subtotal,
		DiscountAmount:  o.DiscountAmount,
		TaxPpn:          o.TaxPpn,
		TaxPph:          o.TaxPph,
		TotalAmount:     o.TotalAmount,
		ShippingCost:    o.ShippingCost,
		CourierName:     o.CourierName,
		ShippingAddress: o.ShippingAddress,
		OrderStatus:     string(o.OrderStatus),
		PaymentStatus:   string(o.PaymentStatus),
		ValidUntil:      o.ValidUntil,
		TermsConditions: o.TermsConditions,
		Notes:           o.Notes,
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
	}

	if len(o.Items) > 0 {
		resp.Items = make([]OrderItemResponse, 0, len(o.Items))
		for _, item := range o.Items {
			itemResp := OrderItemResponse{
				ID:        item.ID,
				ProductID: item.ProductID,
				Qty:       item.Qty,
				Price:     item.Price,
				Details:   item.Details,
			}
			if item.Product != nil {
				prodResp := ToProductResponse(item.Product)
				itemResp.Product = &prodResp
			}
			resp.Items = append(resp.Items, itemResp)
		}
	}

	if o.Customer != nil {
		custResp := ToCustomerResponse(o.Customer)
		resp.Customer = &custResp
	}

	if o.Sales != nil {
		salesResp := ToUserResponse(o.Sales)
		resp.Sales = &salesResp
	}

	return resp
}

func ToOrderResponseList(orders []domain.Order) []OrderResponse {
	var responses []OrderResponse
	for _, o := range orders {
		responses = append(responses, ToOrderResponse(&o))
	}
	if responses == nil {
		return []OrderResponse{}
	}
	return responses
}
