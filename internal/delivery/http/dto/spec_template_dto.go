package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST ---
type SpecTemplateColorRequest struct {
	Name    string `json:"name" validate:"required" example:"Navy Blue"`
	HexCode string `json:"hex_code" validate:"required" example:"#1B263B"`
}

type SpecTemplateRequest struct {
	Name            string                     `json:"name" validate:"required" example:"American Drill"`
	Spec            string                     `json:"spec" validate:"required" example:"Tekstur miring sedang, adem dan tidak gampang kusut"`
	Description     string                     `json:"description" example:"Kain drill serbaguna untuk kemeja taktikal & seragam kerja"`
	Composition     string                     `json:"composition" example:"65% Polyester / 35% Viscose"`
	CareInstruction string                     `json:"care_instruction" example:"Setrika suhu sedang, jangan gunakan pemutih"`
	Colors          []SpecTemplateColorRequest `json:"colors"`
}

// --- RESPONSE ---
type SpecTemplateColorResponse struct {
	ID      string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name    string `json:"name" example:"Navy Blue"`
	HexCode string `json:"hex_code" example:"#1B263B"`
}

type SpecTemplateResponse struct {
	ID              string                      `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name            string                      `json:"name" example:"American Drill"`
	Spec            string                      `json:"spec" example:"Tekstur miring sedang, adem dan tidak gampang kusut"`
	Description     string                      `json:"description" example:"Kain drill serbaguna untuk kemeja taktikal & seragam kerja"`
	Composition     string                      `json:"composition" example:"65% Polyester / 35% Viscose"`
	CareInstruction string                      `json:"care_instruction" example:"Setrika suhu sedang, jangan gunakan pemutih"`
	Colors          []SpecTemplateColorResponse `json:"colors"`
	CreatedAt       time.Time                   `json:"created_at" example:"2023-10-01T15:00:00Z"`
	UpdatedAt       time.Time                   `json:"updated_at" example:"2023-10-01T15:00:00Z"`
}

func ToSpecTemplateResponse(specTemplate *domain.SpecTemplate) SpecTemplateResponse {
	var colors []SpecTemplateColorResponse
	for _, c := range specTemplate.Colors {
		colors = append(colors, SpecTemplateColorResponse{
			ID:      c.ID,
			Name:    c.Name,
			HexCode: c.HexCode,
		})
	}
	if colors == nil {
		colors = []SpecTemplateColorResponse{}
	}

	return SpecTemplateResponse{
		ID:              specTemplate.ID,
		Name:            specTemplate.Name,
		Spec:            specTemplate.Spec,
		Description:     specTemplate.Description,
		Composition:     specTemplate.Composition,
		CareInstruction: specTemplate.CareInstruction,
		Colors:          colors,
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
