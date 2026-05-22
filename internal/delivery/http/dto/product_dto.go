package dto

import "github.com/faridlan/sikon-api/internal/domain"

type ProductRequest struct {
	CategoryID  string  `json:"category_id" validate:"required,uuid"`
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description"`
	BasePrice   float64 `json:"base_price" validate:"required,gt=0"`
}

type ProductResponse struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	BasePrice   float64          `json:"base_price"`
	Category    CategoryResponse `json:"category"`
}

func ToProductResponse(p *domain.Product) ProductResponse {
	resp := ProductResponse{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		BasePrice:   p.BasePrice,
	}
	if p.Category != nil {
		resp.Category = ToCategoryResponse(p.Category)
	}
	return resp
}
