package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type CustomerCreateRequest struct {
	Name    string `json:"name" validate:"required" example:"PT Maju Jaya"`
	Phone   string `json:"phone" validate:"required" example:"081234567890"`
	Address string `json:"address" example:"Jl. Sudirman No. 123, Jakarta"`
}

type CustomerUpdateRequest struct {
	Name    string `json:"name" validate:"omitempty" example:"PT Maju Jaya Abadi"`
	Phone   string `json:"phone" validate:"omitempty" example:"081299998888"`
	Address string `json:"address" validate:"omitempty" example:"Jl. Sudirman No. 123, Jakarta Selatan"`
}

// --- RESPONSE ---
type CustomerResponse struct {
	ID        string    `json:"id" example:"333e4567-e89b-12d3-a456-426614174000"`
	Name      string    `json:"name" example:"PT Maju Jaya"`
	Phone     string    `json:"phone" example:"081234567890"`
	Address   string    `json:"address" example:"Jl. Sudirman No. 123, Jakarta"`
	CreatedBy string    `json:"created_by" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedAt time.Time `json:"created_at" example:"2023-10-01T15:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2023-10-01T15:00:00Z"`

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
