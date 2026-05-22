package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type CustomerCreateRequest struct {
	Name    string `json:"name" validate:"required"`
	Phone   string `json:"phone" validate:"required"`
	Address string `json:"address"`
	// CreatedBy TIDAK ada di sini, karena akan diambil dari token JWT (User yang login)
}

type CustomerUpdateRequest struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}

// --- RESPONSE ---
type CustomerResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Address   string    `json:"address"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Preload relasi
	Creator *UserResponse `json:"creator,omitempty"`
}

func ToCustomerResponse(customer *domain.Customer) CustomerResponse {
	resp := CustomerResponse{
		ID:        customer.ID,
		Name:      customer.Name,
		Phone:     customer.Phone,
		Address:   customer.Address,
		CreatedBy: customer.CreatedBy,
		CreatedAt: customer.CreatedAt,
		UpdatedAt: customer.UpdatedAt,
	}

	if customer.Creator != nil {
		creatorResp := ToUserResponse(customer.Creator)
		resp.Creator = &creatorResp
	}

	return resp
}

func ToCustomerResponseList(customers []domain.Customer) []CustomerResponse {
	var responses []CustomerResponse
	for _, c := range customers {
		responses = append(responses, ToCustomerResponse(&c))
	}
	if responses == nil {
		return []CustomerResponse{}
	}
	return responses
}
