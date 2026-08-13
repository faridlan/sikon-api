package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type SpecTemplateRequest struct {
	Name            string `json:"name" validate:"required" example:"American Drill"`
	Spec            string `json:"spec" validate:"required" example:"Tekstur miring sedang, adem dan tidak gampang kusut"`
	Description     string `json:"description" example:"Kain drill serbaguna untuk kemeja taktikal & seragam kerja"`
	Composition     string `json:"composition" example:"65% Polyester / 35% Viscose"`
	CareInstruction string `json:"care_instruction" example:"Setrika suhu sedang, jangan gunakan pemutih"`
}

// --- RESPONSE ---
type SpecTemplateResponse struct {
	ID              string    `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name            string    `json:"name" example:"American Drill"`
	Spec            string    `json:"spec" example:"Tekstur miring sedang, adem dan tidak gampang kusut"`
	Description     string    `json:"description" example:"Kain drill serbaguna untuk kemeja taktikal & seragam kerja"`
	Composition     string    `json:"composition" example:"65% Polyester / 35% Viscose"`
	CareInstruction string    `json:"care_instruction" example:"Setrika suhu sedang, jangan gunakan pemutih"`
	CreatedAt       time.Time `json:"created_at" example:"2023-10-01T15:00:00Z"`
	UpdatedAt       time.Time `json:"updated_at" example:"2023-10-01T15:00:00Z"`
}

func ToSpecTemplateResponse(specTemplate *domain.SpecTemplate) SpecTemplateResponse {
	return SpecTemplateResponse{
		ID:              specTemplate.ID,
		Name:            specTemplate.Name,
		Spec:            specTemplate.Spec,
		Description:     specTemplate.Description,
		Composition:     specTemplate.Composition,
		CareInstruction: specTemplate.CareInstruction,
		CreatedAt:       specTemplate.CreatedAt,
		UpdatedAt:       specTemplate.UpdatedAt,
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
