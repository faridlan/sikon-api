package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type PaymentRequest struct {
	OrderID         string  `json:"order_id" validate:"required,uuid"`
	BankAccountID   string  `json:"bank_account_id" validate:"required,uuid"`
	Amount          float64 `json:"amount" validate:"required,gt=0"`
	ReferenceNumber string  `json:"reference_number" validate:"required"`
	PaymentType     string  `json:"payment_type" validate:"required,oneof=dp settlement installment"`
}

type PaymentResponse struct {
	ID              string    `json:"id"`
	OrderID         string    `json:"order_id"`
	Amount          float64   `json:"amount"`
	PaymentDate     time.Time `json:"payment_date"`
	PaymentType     string    `json:"payment_type"`
	ReferenceNumber string    `json:"reference_number"`
}

func ToPaymentResponse(p *domain.Payment) PaymentResponse {
	return PaymentResponse{
		ID:              p.ID,
		OrderID:         p.OrderID,
		Amount:          p.Amount,
		PaymentDate:     p.PaymentDate,
		PaymentType:     string(p.PaymentType),
		ReferenceNumber: p.ReferenceNumber,
	}
}
