package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type BankAccountCreateRequest struct {
	UserID        *string `json:"user_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	BankName      string  `json:"bank_name" validate:"required" example:"BCA"`
	AccountNumber string  `json:"account_number" validate:"required" example:"1234567890"`
	AccountName   string  `json:"account_name" validate:"required" example:"PT SIKOn Konveksi"`
}

type BankAccountUpdateRequest struct {
	BankName      string `json:"bank_name" validate:"omitempty" example:"Mandiri"`
	AccountNumber string `json:"account_number" validate:"omitempty" example:"0987654321"`
	AccountName   string `json:"account_name" validate:"omitempty" example:"Budi Haryanto"`
}

// --- RESPONSE ---
type BankAccountResponse struct {
	ID            string    `json:"id" example:"777e4567-e89b-12d3-a456-426614174000"`
	UserID        *string   `json:"user_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	BankName      string    `json:"bank_name" example:"BCA"`
	AccountNumber string    `json:"account_number" example:"1234567890"`
	AccountName   string    `json:"account_name" example:"PT SIKOn Konveksi"`
	CreatedAt     time.Time `json:"created_at" example:"2023-10-01T15:00:00Z"`
	UpdatedAt     time.Time `json:"updated_at" example:"2023-10-01T15:00:00Z"`
}

func ToBankAccountResponse(ba *domain.BankAccount) BankAccountResponse {
	return BankAccountResponse{
		ID:            ba.ID,
		UserID:        ba.UserID,
		BankName:      ba.BankName,
		AccountNumber: ba.AccountNumber,
		AccountName:   ba.AccountName,
		CreatedAt:     ba.CreatedAt,
		UpdatedAt:     ba.UpdatedAt,
	}
}

func ToBankAccountResponseList(accounts []domain.BankAccount) []BankAccountResponse {
	var responses []BankAccountResponse
	for _, ba := range accounts {
		responses = append(responses, ToBankAccountResponse(&ba))
	}
	if responses == nil {
		return []BankAccountResponse{}
	}
	return responses
}
