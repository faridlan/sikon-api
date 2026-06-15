package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type SpecTemplateRequest struct {
	Name string `json:"name" validate:"required" example:"Bahan Rompi Standar"`
	Spec string `json:"spec" validate:"required" example:"Drill Halus, Furing Peles, Resleting YKK"`
}

// --- RESPONSE ---
type SpecTemplateResponse struct {
	ID        string    `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name      string    `json:"name" example:"Bahan Rompi Standar"`
	Spec      string    `json:"spec" example:"Drill Halus, Furing Peles, Resleting YKK"`
	CreatedAt time.Time `json:"created_at" example:"2023-10-01T15:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2023-10-01T15:00:00Z"`
}

func ToSpecTemplateResponse(specTemplate *domain.SpecTemplate) SpecTemplateResponse {
	return SpecTemplateResponse{
		ID:        specTemplate.ID,
		Name:      specTemplate.Name,
		Spec:      specTemplate.Spec,
		CreatedAt: specTemplate.CreatedAt,
		UpdatedAt: specTemplate.UpdatedAt,
	}
}

func ToSpecTemplateResponseList(specTemplates []domain.SpecTemplate) []SpecTemplateResponse {
	var responses []SpecTemplateResponse
	for _, s := range specTemplates {
		responses = append(responses, ToSpecTemplateResponse(&s))
	}
	if responses == nil {
		return []SpecTemplateResponse{}
	}
	return responses
}
