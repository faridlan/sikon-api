package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// ==========================================
// REQUEST DTO: MATERIAL
// ==========================================

type MaterialCreateRequest struct {
	Name            string               `json:"name" validate:"required" example:"Kain Ripstop"`
	Unit            string               `json:"unit" validate:"required" example:"meter"`
	UnitPrice       float64              `json:"unit_price" validate:"required,gt=0" example:"25000"`
	Category        string               `json:"category" validate:"omitempty" example:"kain"`
	Description     string               `json:"description" validate:"omitempty" example:"Kain ripstop tahan robek"`
	Composition     string               `json:"composition" validate:"omitempty" example:"65% Cotton / 35% Polyester"`
	CareInstruction string               `json:"care_instruction" validate:"omitempty" example:"Cuci air dingin"`
	GSMInfo         string               `json:"gsm_info" validate:"omitempty" example:"210gsm"`
	Colors          []FabricColorRequest `json:"colors" validate:"omitempty,dive"`
}

type MaterialUpdateRequest struct {
	Name            string               `json:"name" validate:"omitempty" example:"Kain Ripstop"`
	Unit            string               `json:"unit" validate:"omitempty" example:"meter"`
	UnitPrice       float64              `json:"unit_price" validate:"omitempty,gt=0" example:"27000"`
	Category        string               `json:"category" validate:"omitempty" example:"kain"`
	Description     string               `json:"description" validate:"omitempty" example:"Kain ripstop tahan robek"`
	Composition     string               `json:"composition" validate:"omitempty" example:"65% Cotton / 35% Polyester"`
	CareInstruction string               `json:"care_instruction" validate:"omitempty" example:"Cuci air dingin"`
	GSMInfo         string               `json:"gsm_info" validate:"omitempty" example:"210gsm"`
	Colors          []FabricColorRequest `json:"colors" validate:"omitempty,dive"`
}

// ==========================================
// RESPONSE DTO: MATERIAL
// ==========================================

type MaterialResponse struct {
	ID              string                `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name            string                `json:"name" example:"Kain Ripstop"`
	Unit            string                `json:"unit" example:"meter"`
	UnitPrice       float64               `json:"unit_price" example:"25000"`
	Category        string                `json:"category" example:"kain"`
	Description     string                `json:"description,omitempty" example:"Kain ripstop tahan robek"`
	Composition     string                `json:"composition,omitempty" example:"65% Cotton / 35% Polyester"`
	CareInstruction string                `json:"care_instruction,omitempty" example:"Cuci air dingin"`
	GSMInfo         string                `json:"gsm_info,omitempty" example:"210gsm"`
	CreatedAt       time.Time             `json:"created_at" example:"2026-09-09T10:00:00Z"`
	Colors          []FabricColorResponse `json:"colors,omitempty"`
}

func ToMaterialResponse(m *domain.Material) MaterialResponse {
	resp := MaterialResponse{
		ID:              m.ID,
		Name:            m.Name,
		Unit:            m.Unit,
		UnitPrice:       m.UnitPrice,
		Category:        m.Category,
		Description:     m.Description,
		Composition:     m.Composition,
		CareInstruction: m.CareInstruction,
		GSMInfo:         m.GSMInfo,
		CreatedAt:       m.CreatedAt,
	}
	if len(m.Colors) > 0 {
		resp.Colors = make([]FabricColorResponse, len(m.Colors))
		for i, c := range m.Colors {
			resp.Colors[i] = FabricColorResponse{
				ID:      c.ID,
				Name:    c.Name,
				HexCode: c.HexCode,
			}
		}
	}
	return resp
}

func ToMaterialResponseList(list []domain.Material) []MaterialResponse {
	if len(list) == 0 {
		return []MaterialResponse{}
	}
	responses := make([]MaterialResponse, len(list))
	for i, m := range list {
		responses[i] = ToMaterialResponse(&m)
	}
	return responses
}

// ==========================================
// REQUEST DTO: PRODUCT MATERIAL (Resep / BOM)
// ==========================================

type ProductMaterialItemRequest struct {
	MaterialID string  `json:"material_id" validate:"required,uuid4" example:"123e4567-e89b-12d3-a456-426614174000"`
	QtyPerUnit float64 `json:"qty_per_unit" validate:"required,gt=0" example:"1.2"`
}

type SetProductMaterialsRequest struct {
	Items []ProductMaterialItemRequest `json:"items" validate:"required,dive"`
}

// ==========================================
// RESPONSE DTO: PRODUCT MATERIAL (Resep / BOM)
// ==========================================

type ProductMaterialResponse struct {
	ID         string             `json:"id" example:"123e4567-e89b-12d3-a456-426614174001"`
	MaterialID string             `json:"material_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	QtyPerUnit float64            `json:"qty_per_unit" example:"1.2"`
	Material   *MaterialResponse  `json:"material,omitempty"`
}

func ToProductMaterialResponse(pm *domain.ProductMaterial) ProductMaterialResponse {
	resp := ProductMaterialResponse{
		ID:         pm.ID,
		MaterialID: pm.MaterialID,
		QtyPerUnit: pm.QtyPerUnit,
	}
	if pm.Material != nil {
		matResp := ToMaterialResponse(pm.Material)
		resp.Material = &matResp
	}
	return resp
}

func ToProductMaterialResponseList(list []domain.ProductMaterial) []ProductMaterialResponse {
	if len(list) == 0 {
		return []ProductMaterialResponse{}
	}
	responses := make([]ProductMaterialResponse, len(list))
	for i, pm := range list {
		responses[i] = ToProductMaterialResponse(&pm)
	}
	return responses
}
