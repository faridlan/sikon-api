package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type OrderItemRequest struct {
	ProductID     string  `json:"product_id" validate:"required,uuid" example:"999e4567-e89b-12d3-a456-426614174000"`
	FabricID      *string `json:"fabric_id" validate:"omitempty,uuid" example:"111e4567-e89b-12d3-a456-426614174000"`
	FabricColorID *string `json:"fabric_color_id" validate:"omitempty,uuid" example:"222e4567-e89b-12d3-a456-426614174000"`
	CustomName    string  `json:"custom_name" validate:"omitempty" example:"Custom Name"`
	Qty           int     `json:"qty" validate:"required,gt=0" example:"100"`
	Price         float64 `json:"price" validate:"omitempty" example:"35000"`

	// @Schema type object
	// @Schema example {"ukuran": "L", "warna": "Hitam"}
	Details any `json:"details"`
}

type OrderCreateRequest struct {
	BatchPoID       string             `json:"batch_po_id" validate:"required,uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	CustomerID      string             `json:"customer_id" validate:"required,uuid" example:"333e4567-e89b-12d3-a456-426614174000"`
	SalesID         string             `json:"sales_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	IsTaxable       bool               `json:"is_taxable" example:"true"`
	TaxPpnRate      float64            `json:"tax_ppn_rate" validate:"gte=0" example:"12.00"`
	TaxPph22Rate    float64            `json:"tax_pph22_rate" validate:"gte=0" example:"1.50"`
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
	IsTaxable       bool       `json:"is_taxable" example:"true"`
	TaxPpnRate      float64    `json:"tax_ppn_rate" validate:"gte=0" example:"12.00"`
	TaxPph22Rate    float64    `json:"tax_pph22_rate" validate:"gte=0" example:"1.50"`
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
	ID              string           `json:"id" example:"item-uuid"`
	ProductID       string           `json:"product_id" example:"999e4567-e89b-12d3-a456-426614174000"`
	FabricID        *string          `json:"fabric_id,omitempty" example:"111e4567-e89b-12d3-a456-426614174000"`
	FabricColorID   *string          `json:"fabric_color_id,omitempty" example:"222e4567-e89b-12d3-a456-426614174000"`
	FabricName      string           `json:"fabric_name,omitempty" example:"American Drill"`
	FabricColorName string           `json:"fabric_color_name,omitempty" example:"Navy Blue"`
	FabricHexCode   string           `json:"fabric_hex_code,omitempty" example:"#000080"`
	CustomName      string           `json:"custom_name" example:"Custom Name"`
	Qty             int              `json:"qty" example:"100"`
	Price           float64          `json:"price" example:"35000"`
	Details         any              `json:"details"`
	Product         *ProductResponse `json:"product,omitempty"`
}

type OrderResponse struct {
	ID                string     `json:"id" example:"ord-uuid"`
	OrderNumber       string     `json:"order_number" example:"ORD-20231001-1234"`
	BatchPoID         string     `json:"batch_po_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	CustomerID        string     `json:"customer_id" example:"333e4567-e89b-12d3-a456-426614174000"`
	SalesID           string     `json:"sales_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Subtotal          float64    `json:"subtotal" example:"5325000"`
	DiscountAmount    float64    `json:"discount_amount" example:"0"`
	IsTaxable         bool       `json:"is_taxable" example:"true"`
	TaxPpnRate        float64    `json:"tax_ppn_rate" example:"12.00"`
	TaxPph22Rate      float64    `json:"tax_pph22_rate" example:"1.50"`
	DppPpn            float64    `json:"dpp_ppn" example:"4885321.10"`
	DppPph            float64    `json:"dpp_pph" example:"5325000"`
	TaxPpn            float64    `json:"tax_ppn" example:"586238.53"`
	TaxPph            float64    `json:"tax_pph" example:"73279.82"`
	TotalAmount       float64    `json:"total_amount" example:"5325000"`
	PaguAmount        float64    `json:"pagu_amount" example:"5911238.53"`
	NetReceivedAmount float64    `json:"net_received_amount" example:"5251719.53"`
	TotalQty          int        `json:"total_qty" example:"15"`
	ShippingCost      float64    `json:"shipping_cost" example:"0"`
	CourierName       string     `json:"courier_name" example:"Self Pickup"`
	ShippingAddress   string     `json:"shipping_address" example:"Kantor Dinas"`
	OrderStatus       string     `json:"order_status" example:"pending"`
	PaymentStatus     string     `json:"payment_status" example:"unpaid"`
	ValidUntil        *time.Time `json:"valid_until,omitempty" example:"2026-06-15T00:00:00Z"`
	TermsConditions   string     `json:"terms_conditions,omitempty" example:"DP Minimal 50%"`
	Notes             string     `json:"notes" example:"Pengadaan Baju Dinas"`
	CreatedAt         time.Time  `json:"created_at" example:"2026-08-31T15:00:00Z"`
	UpdatedAt         time.Time  `json:"updated_at" example:"2026-08-31T15:00:00Z"`
	ApprovedAt        *time.Time `json:"approved_at,omitempty" example:"2023-10-01T16:00:00Z"`
	HPPMaterialCost   float64    `json:"hpp_material_cost,omitempty" example:"600000"`
	HPPCalculatedAt   *time.Time `json:"hpp_calculated_at,omitempty"`

	Items    []OrderItemResponse `json:"items,omitempty"`
	Customer *CustomerResponse   `json:"customer,omitempty"`
	Sales    *UserResponse       `json:"sales,omitempty"`
	BatchPO  *BatchPOResponse    `json:"batch_po,omitempty"`
}

type OrderHPPResponse struct {
	OrderID              string     `json:"order_id" example:"ord-uuid"`
	MaterialCost         float64    `json:"material_cost" example:"600000"`
	MaterialCalculatedAt *time.Time `json:"material_calculated_at,omitempty"`
	LaborCost            float64    `json:"labor_cost" example:"350000"`
	TotalCost            float64    `json:"total_cost" example:"950000"`
}

func ToOrderResponse(o *domain.Order) OrderResponse {

	if o.TotalQty == 0 && len(o.Items) > 0 {
		o.CalculateTotals()
	}

	resp := OrderResponse{
		ID:                o.ID,
		OrderNumber:       o.OrderNumber,
		BatchPoID:         o.BatchPoID,
		CustomerID:        o.CustomerID,
		SalesID:           o.SalesID,
		Subtotal:          o.Subtotal,
		DiscountAmount:    o.DiscountAmount,
		IsTaxable:         o.IsTaxable,
		TaxPpnRate:        o.TaxPpnRate,
		TaxPph22Rate:      o.TaxPph22Rate,
		DppPpn:            o.DppPpn,
		DppPph:            o.DppPph,
		TaxPpn:            o.TaxPpn,
		TaxPph:            o.TaxPph,
		TotalAmount:       o.TotalAmount,
		PaguAmount:        o.PaguAmount,
		NetReceivedAmount: o.NetReceivedAmount,
		TotalQty:          o.TotalQty,
		ShippingCost:      o.ShippingCost,
		CourierName:       o.CourierName,
		ShippingAddress:   o.ShippingAddress,
		OrderStatus:       string(o.OrderStatus),
		PaymentStatus:     string(o.PaymentStatus),
		ValidUntil:        o.ValidUntil,
		TermsConditions:   o.TermsConditions,
		Notes:             o.Notes,
		CreatedAt:         o.CreatedAt,
		UpdatedAt:         o.UpdatedAt,
		ApprovedAt:        o.ApprovedAt,
		HPPMaterialCost:   o.HPPMaterialCost,
		HPPCalculatedAt:   o.HPPCalculatedAt,
	}

	if len(o.Items) > 0 {
		resp.Items = make([]OrderItemResponse, 0, len(o.Items))
		for _, item := range o.Items {
			itemResp := OrderItemResponse{
				ID:            item.ID,
				ProductID:     item.ProductID,
				FabricID:      item.FabricID,
				FabricColorID: item.FabricColorID,
				CustomName:    item.CustomName,
				Qty:           item.Qty,
				Price:         item.Price,
				Details:       item.Details,
			}
			if item.Fabric != nil {
				itemResp.FabricName = item.Fabric.Name
			}
			if item.FabricColor != nil {
				itemResp.FabricColorName = item.FabricColor.Name
				itemResp.FabricHexCode = item.FabricColor.HexCode
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

	if o.BatchPO != nil {
		batchPOResp := ToBatchPOResponse(o.BatchPO)
		resp.BatchPO = &batchPOResp
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

func ToOrderHPPResponse(h *domain.OrderHPPBreakdown) OrderHPPResponse {
	return OrderHPPResponse{
		OrderID:              h.OrderID,
		MaterialCost:         h.MaterialCost,
		MaterialCalculatedAt: h.MaterialCalculatedAt,
		LaborCost:            h.LaborCost,
		TotalCost:            h.TotalCost,
	}
}
