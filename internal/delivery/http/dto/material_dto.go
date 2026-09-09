package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// ==========================================
// REQUEST DTO: MATERIAL
// ==========================================

type MaterialCreateRequest struct {
	Name      string  `json:"name" validate:"required" example:"Kain Ripstop"`
	Unit      string  `json:"unit" validate:"required" example:"meter"`
	UnitPrice float64 `json:"unit_price" validate:"required,gt=0" example:"25000"`
	Category  string  `json:"category" validate:"omitempty" example:"kain"`
}

type MaterialUpdateRequest struct {
	Name      string  `json:"name" validate:"omitempty" example:"Kain Ripstop"`
	Unit      string  `json:"unit" validate:"omitempty" example:"meter"`
	UnitPrice float64 `json:"unit_price" validate:"omitempty,gt=0" example:"27000"`
	Category  string  `json:"category" validate:"omitempty" example:"kain"`
}

// ==========================================
// RESPONSE DTO: MATERIAL
// ==========================================

type MaterialResponse struct {
	ID        string    `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name      string    `json:"name" example:"Kain Ripstop"`
	Unit      string    `json:"unit" example:"meter"`
	UnitPrice float64   `json:"unit_price" example:"25000"`
	Category  string    `json:"category" example:"kain"`
	CreatedAt time.Time `json:"created_at" example:"2026-09-09T10:00:00Z"`
}

func ToMaterialResponse(m *domain.Material) MaterialResponse {
	return MaterialResponse{
		ID:        m.ID,
		Name:      m.Name,
		Unit:      m.Unit,
		UnitPrice: m.UnitPrice,
		Category:  m.Category,
		CreatedAt: m.CreatedAt,
	}
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
