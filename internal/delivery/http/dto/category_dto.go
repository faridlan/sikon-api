package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type CategoryRequest struct {
	Name     string `json:"name" validate:"required" example:"Kaos Polos"`
	ImageURL string `json:"image_url" validate:"omitempty,url" example:"https://example.com/cat.jpg"`
}

// --- RESPONSE ---
type CategoryResponse struct {
	ID        string    `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name      string    `json:"name" example:"Kaos Polos"`
	ImageURL  string    `json:"image_url" example:"https://example.com/cat.jpg"`
	CreatedAt time.Time `json:"created_at" example:"2023-10-01T15:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2023-10-01T15:00:00Z"`
}

func ToCategoryResponse(category *domain.Category) CategoryResponse {
	return CategoryResponse{
		ID:        category.ID,
		Name:      category.Name,
		ImageURL:  category.ImageURL,
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
