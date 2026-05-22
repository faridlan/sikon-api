package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type CategoryRequest struct {
	Name string `json:"name" validate:"required"`
}

// --- RESPONSE ---
type CategoryResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToCategoryResponse(category *domain.Category) CategoryResponse {
	return CategoryResponse{
		ID:        category.ID,
		Name:      category.Name,
		CreatedAt: category.CreatedAt,
		UpdatedAt: category.UpdatedAt,
	}
}

func ToCategoryResponseList(categories []domain.Category) []CategoryResponse {
	var responses []CategoryResponse
	for _, c := range categories {
		responses = append(responses, ToCategoryResponse(&c))
	}
	if responses == nil {
		return []CategoryResponse{}
	}
	return responses
}
