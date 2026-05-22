package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type BankAccountRequest struct {
	UserID        *string `json:"user_id"` // Jika nil, dianggap rekening global
	BankName      string  `json:"bank_name" validate:"required"`
	AccountNumber string  `json:"account_number" validate:"required"`
	AccountName   string  `json:"account_name" validate:"required"`
}

type BankAccountResponse struct {
	ID            string    `json:"id"`
	UserID        *string   `json:"user_id"`
	BankName      string    `json:"bank_name"`
	AccountNumber string    `json:"account_number"`
	AccountName   string    `json:"account_name"`
	CreatedAt     time.Time `json:"created_at"`
}

func ToBankAccountResponse(ba *domain.BankAccount) BankAccountResponse {
	return BankAccountResponse{
		ID:            ba.ID,
		UserID:        ba.UserID,
		BankName:      ba.BankName,
		AccountNumber: ba.AccountNumber,
		AccountName:   ba.AccountName,
		CreatedAt:     ba.CreatedAt,
	}
}
