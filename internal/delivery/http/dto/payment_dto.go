package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type PaymentCreateRequest struct {
	OrderID         string    `json:"order_id" validate:"required,uuid" example:"ord-uuid"`
	BankAccountID   string    `json:"bank_account_id" validate:"required,uuid" example:"777e4567-e89b-12d3-a456-426614174000"`
	Amount          float64   `json:"amount" validate:"required,gt=0" example:"1000000"`
	PaymentDate     time.Time `json:"payment_date" validate:"omitempty" example:"2023-10-02T10:00:00Z"`
	ReferenceNumber string    `json:"reference_number" validate:"required" example:"TRX-0987654321"`
	PaymentType     string    `json:"payment_type" validate:"required,oneof=dp settlement installment" example:"dp"`
}

type PaymentUpdateRequest struct {
	ReferenceNumber string `json:"reference_number" validate:"omitempty" example:"TRX-0987654322-REVISI"`
	PaymentType     string `json:"payment_type" validate:"omitempty,oneof=dp settlement installment" example:"settlement"`
}

// --- RESPONSE ---
type PaymentResponse struct {
	ID              string               `json:"id" example:"pay-uuid"`
	OrderID         string               `json:"order_id" example:"ord-uuid"`
	BankAccountID   string               `json:"bank_account_id" example:"777e4567-e89b-12d3-a456-426614174000"`
	Amount          float64              `json:"amount" example:"1000000"`
	PaymentDate     time.Time            `json:"payment_date" example:"2023-10-02T10:00:00Z"`
	ReferenceNumber string               `json:"reference_number" example:"TRX-0987654321"`
	PaymentType     string               `json:"payment_type" example:"dp"`
	CreatedAt       time.Time            `json:"created_at" example:"2023-10-02T10:05:00Z"`
	UpdatedAt       time.Time            `json:"updated_at" example:"2023-10-02T10:05:00Z"`
	Order           *OrderResponse       `json:"order,omitempty"`
	BankAccount     *BankAccountResponse `json:"bank_account,omitempty"`
}

func ToPaymentResponse(p *domain.Payment) PaymentResponse {
	resp := PaymentResponse{
		ID:              p.ID,
		OrderID:         p.OrderID,
		BankAccountID:   p.BankAccountID,
		Amount:          p.Amount,
		PaymentDate:     p.PaymentDate,
		ReferenceNumber: p.ReferenceNumber,
		PaymentType:     string(p.PaymentType),
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
	}

	// Map relasi Order jika tersedia
	if p.Order != nil {
		orderResp := ToOrderResponse(p.Order)
		resp.Order = &orderResp
	}

	// Map relasi BankAccount jika tersedia
	if p.BankAccount != nil {
		bankAccountResp := ToBankAccountResponse(p.BankAccount)
		resp.BankAccount = &bankAccountResp
	}

	return resp
}

func ToPaymentResponseList(payments []domain.Payment) []PaymentResponse {
	var responses []PaymentResponse
	for _, p := range payments {
		responses = append(responses, ToPaymentResponse(&p))
	}
	if responses == nil {
		return []PaymentResponse{}
	}
	return responses
}
