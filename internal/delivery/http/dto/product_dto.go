package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type ProductCreateRequest struct {
	CategoryID  string  `json:"category_id" validate:"required,uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string  `json:"name" validate:"required" example:"Kaos Polos Cotton Combed 30s"`
	Description string  `json:"description" example:"Kaos polos bahan cotton combed 30s kualitas premium"`
	BasePrice   float64 `json:"base_price" validate:"required,gt=0" example:"35000"`
}

type ProductUpdateRequest struct {
	CategoryID  string  `json:"category_id" validate:"omitempty,uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string  `json:"name" validate:"omitempty" example:"Kaos Polos Lengan Panjang"`
	Description string  `json:"description" validate:"omitempty" example:"Versi lengan panjang"`
	BasePrice   float64 `json:"base_price" validate:"omitempty,gt=0" example:"45000"`
}

// --- RESPONSE ---
type ProductResponse struct {
	ID          string    `json:"id" example:"999e4567-e89b-12d3-a456-426614174000"`
	CategoryID  string    `json:"category_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string    `json:"name" example:"Kaos Polos Cotton Combed 30s"`
	Description string    `json:"description" example:"Kaos polos bahan cotton combed 30s kualitas premium"`
	BasePrice   float64   `json:"base_price" example:"35000"`
	CreatedAt   time.Time `json:"created_at" example:"2023-10-01T15:00:00Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2023-10-01T15:00:00Z"`

	Category *CategoryResponse `json:"category,omitempty"`
}

func ToProductResponse(p *domain.Product) ProductResponse {
	resp := ProductResponse{
		ID:          p.ID,
		CategoryID:  p.CategoryID,
		Name:        p.Name,
		Description: p.Description,
		BasePrice:   p.BasePrice,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}

	if p.Category != nil {
		catResp := ToCategoryResponse(p.Category)
		resp.Category = &catResp
	}

	return resp
}

func ToProductResponseList(products []domain.Product) []ProductResponse {
	var responses []ProductResponse
	for _, p := range products {
		responses = append(responses, ToProductResponse(&p))
	}
	if responses == nil {
		return []ProductResponse{}
	}
	return responses
}
