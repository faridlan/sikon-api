package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// --- REQUEST SUB-STRUCTS ---

type FabricColorRequest struct {
	Name    string `json:"name" validate:"required" example:"Olive"`
	HexCode string `json:"hex_code" validate:"required" example:"#4b5320"`
}

type ProductFabricRequest struct {
	SpecTemplateID  *string              `json:"spec_template_id" validate:"omitempty,uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name            string               `json:"name" example:"Ripstop Cotton 65/35"`
	Description     string               `json:"description" example:"Kuat & anti robek, 210gsm"`
	Composition     string               `json:"composition" example:"65% Cotton / 35% Polyester"`
	CareInstruction string               `json:"care_instruction" example:"Cuci mesin air dingin"`
	BasePrice       float64              `json:"base_price" validate:"gte=0" example:"185000"`
	PriceAdjustment float64              `json:"price_adjustment" example:"0"`
	IsDefault       bool                 `json:"is_default" example:"true"`
	Colors          []FabricColorRequest `json:"colors" validate:"omitempty,dive"`
}

type WholesalePriceRequest struct {
	FabricID  *string `json:"fabric_id" validate:"omitempty,uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	MinQty    int     `json:"min_qty" validate:"required,gt=0" example:"6"`
	MaxQty    *int    `json:"max_qty" validate:"omitempty,gt=0" example:"24"`
	UnitPrice float64 `json:"unit_price" validate:"required,gt=0" example:"175000"`
}

type ProductModelViewRequest struct {
	Side    string `json:"side" validate:"required,oneof=front back" example:"front"`
	ArtURL  string `json:"art_url" validate:"required,url" example:"https://supa.../front-art.png"`
	MaskURL string `json:"mask_url" validate:"required,url" example:"https://supa.../front-mask.png"`
	Width   int    `json:"width" example:"1756"`
	Height  int    `json:"height" example:"1920"`
}

type ProductModelRequest struct {
	Name        string                    `json:"name" validate:"required" example:"Series 1 — Lengan Panjang"`
	Type        string                    `json:"type" validate:"required" example:"long_sleeve"`
	Description string                    `json:"description" example:"Template kemeja taktikal 2 saku"`
	Views       []ProductModelViewRequest `json:"views" validate:"required,dive"`
}

// --- MAIN REQUESTS ---

type ProductCreateRequest struct {
	CategoryID    string                  `json:"category_id" validate:"required,uuid" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name          string                  `json:"name" validate:"required" example:"Kemeja Taktikal Premium 7200"`
	Description   string                  `json:"description" example:"Kemeja taktikal bahan ripstop cotton 65/35"`
	BasePrice     float64                 `json:"base_price" validate:"required,gt=0" example:"185000"`
	Slug          string                  `json:"slug" example:"kemeja-taktikal-premium-7200"`
	GSMInfo       string                  `json:"gsm_info" example:"210gsm"`
	FabricSummary string                  `json:"fabric_summary" example:"Ripstop"`
	KeyFeatures   []string                `json:"key_features" example:"[\"Bahan ripstop anti robek\"]"`
	ImageURLs     []string                `json:"image_urls" validate:"omitempty,dive,url"`
	Fabrics       []ProductFabricRequest  `json:"fabrics" validate:"omitempty,dive"`
	Wholesale     []WholesalePriceRequest `json:"wholesale" validate:"omitempty,dive"`
	DesignModel   *ProductModelRequest    `json:"design_model" validate:"omitempty"`
}

type ProductUpdateRequest struct {
	CategoryID    string                  `json:"category_id" validate:"omitempty,uuid"`
	Name          string                  `json:"name" validate:"omitempty"`
	Description   string                  `json:"description" validate:"omitempty"`
	BasePrice     float64                 `json:"base_price" validate:"omitempty,gt=0"`
	Slug          string                  `json:"slug" validate:"omitempty"`
	GSMInfo       string                  `json:"gsm_info" validate:"omitempty"`
	FabricSummary string                  `json:"fabric_summary" validate:"omitempty"`
	KeyFeatures   []string                `json:"key_features" validate:"omitempty"`
	ImageURLs     []string                `json:"image_urls" validate:"omitempty,dive,url"`
	Fabrics       []ProductFabricRequest  `json:"fabrics" validate:"omitempty,dive"`
	Wholesale     []WholesalePriceRequest `json:"wholesale" validate:"omitempty,dive"`
	DesignModel   *ProductModelRequest    `json:"design_model" validate:"omitempty"`
}

// --- RESPONSE SUB-STRUCTS ---

type ProductImageResponse struct {
	ID        string `json:"id"`
	ImageURL  string `json:"image_url"`
	IsPrimary bool   `json:"is_primary"`
}

type FabricColorResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	HexCode string `json:"hex_code"`
}

type ProductFabricResponse struct {
	ID              string                `json:"id"`
	SpecTemplateID  *string               `json:"spec_template_id,omitempty"`
	Name            string                `json:"name"`
	Description     string                `json:"description"`
	Composition     string                `json:"composition"`
	CareInstruction string                `json:"care_instruction"`
	BasePrice       float64               `json:"base_price"`
	PriceAdjustment float64               `json:"price_adjustment"`
	IsDefault       bool                  `json:"is_default"`
	Colors          []FabricColorResponse `json:"colors,omitempty"`
}

type WholesalePriceResponse struct {
	ID        string  `json:"id"`
	FabricID  *string `json:"fabric_id,omitempty"`
	MinQty    int     `json:"min_qty"`
	MaxQty    *int    `json:"max_qty,omitempty"`
	UnitPrice float64 `json:"unit_price"`
}

type ProductModelViewResponse struct {
	ID      string `json:"id"`
	Side    string `json:"side"`
	ArtURL  string `json:"art_url"`
	MaskURL string `json:"mask_url"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

type ProductModelResponse struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name"`
	Type        string                     `json:"type"`
	Description string                     `json:"description"`
	Views       []ProductModelViewResponse `json:"views,omitempty"`
}

// --- MAIN RESPONSE ---

type ProductResponse struct {
	ID            string                   `json:"id"`
	CategoryID    string                   `json:"category_id"`
	Name          string                   `json:"name"`
	Description   string                   `json:"description"`
	BasePrice     float64                  `json:"base_price"`
	Slug          string                   `json:"slug"`
	GSMInfo       string                   `json:"gsm_info"`
	FabricSummary string                   `json:"fabric_summary"`
	Rating        float64                  `json:"rating"`
	SoldCount     int                      `json:"sold_count"`
	ReviewCount   int                      `json:"review_count"`
	KeyFeatures   []string                 `json:"key_features,omitempty"`
	CreatedAt     time.Time                `json:"created_at"`
	UpdatedAt     time.Time                `json:"updated_at"`
	Images        []ProductImageResponse   `json:"images,omitempty"`
	Category      *CategoryResponse        `json:"category,omitempty"`
	Fabrics       []ProductFabricResponse  `json:"fabrics,omitempty"`
	Wholesale     []WholesalePriceResponse `json:"wholesale,omitempty"`
	DesignModel   *ProductModelResponse    `json:"design_model,omitempty"`
}

// --- MAPPERS ---

func (r *ProductCreateRequest) ToDomainCreateInput() domain.ProductCreateInput {
	input := domain.ProductCreateInput{
		CategoryID:    r.CategoryID,
		Name:          r.Name,
		Description:   r.Description,
		BasePrice:     r.BasePrice,
		Slug:          r.Slug,
		GSMInfo:       r.GSMInfo,
		FabricSummary: r.FabricSummary,
		KeyFeatures:   r.KeyFeatures,
		ImageURLs:     r.ImageURLs,
	}

	for _, fab := range r.Fabrics {
		var colors []domain.FabricColorInput
		for _, c := range fab.Colors {
			colors = append(colors, domain.FabricColorInput{
				Name:    c.Name,
				HexCode: c.HexCode,
			})
		}
		input.Fabrics = append(input.Fabrics, domain.ProductFabricInput{
			SpecTemplateID:  fab.SpecTemplateID,
			Name:            fab.Name,
			Description:     fab.Description,
			Composition:     fab.Composition,
			CareInstruction: fab.CareInstruction,
			BasePrice:       fab.BasePrice,
			PriceAdjustment: fab.PriceAdjustment,
			IsDefault:       fab.IsDefault,
			Colors:          colors,
		})
	}

	for _, w := range r.Wholesale {
		input.Wholesale = append(input.Wholesale, domain.WholesalePriceInput{
			FabricID:  w.FabricID,
			MinQty:    w.MinQty,
			MaxQty:    w.MaxQty,
			UnitPrice: w.UnitPrice,
		})
	}

	if r.DesignModel != nil {
		var views []domain.ProductModelViewInput
		for _, v := range r.DesignModel.Views {
			views = append(views, domain.ProductModelViewInput{
				Side:    v.Side,
				ArtURL:  v.ArtURL,
				MaskURL: v.MaskURL,
				Width:   v.Width,
				Height:  v.Height,
			})
		}
		input.DesignModel = &domain.ProductModelInput{
			Name:        r.DesignModel.Name,
			Type:        r.DesignModel.Type,
			Description: r.DesignModel.Description,
			Views:       views,
		}
	}

	return input
}

func (r *ProductUpdateRequest) ToDomainUpdateInput() domain.ProductUpdateInput {
	input := domain.ProductUpdateInput{
		CategoryID:    r.CategoryID,
		Name:          r.Name,
		Description:   r.Description,
		BasePrice:     r.BasePrice,
		Slug:          r.Slug,
		GSMInfo:       r.GSMInfo,
		FabricSummary: r.FabricSummary,
		KeyFeatures:   r.KeyFeatures,
		ImageURLs:     r.ImageURLs,
	}

	if r.Fabrics != nil {
		for _, fab := range r.Fabrics {
			var colors []domain.FabricColorInput
			for _, c := range fab.Colors {
				colors = append(colors, domain.FabricColorInput{
					Name:    c.Name,
					HexCode: c.HexCode,
				})
			}
			input.Fabrics = append(input.Fabrics, domain.ProductFabricInput{
				SpecTemplateID:  fab.SpecTemplateID,
				Name:            fab.Name,
				Description:     fab.Description,
				Composition:     fab.Composition,
				CareInstruction: fab.CareInstruction,
				BasePrice:       fab.BasePrice,
				PriceAdjustment: fab.PriceAdjustment,
				IsDefault:       fab.IsDefault,
				Colors:          colors,
			})
		}
	}

	if r.Wholesale != nil {
		for _, w := range r.Wholesale {
			input.Wholesale = append(input.Wholesale, domain.WholesalePriceInput{
				FabricID:  w.FabricID,
				MinQty:    w.MinQty,
				MaxQty:    w.MaxQty,
				UnitPrice: w.UnitPrice,
			})
		}
	}

	if r.DesignModel != nil {
		var views []domain.ProductModelViewInput
		for _, v := range r.DesignModel.Views {
			views = append(views, domain.ProductModelViewInput{
				Side:    v.Side,
				ArtURL:  v.ArtURL,
				MaskURL: v.MaskURL,
				Width:   v.Width,
				Height:  v.Height,
			})
		}
		input.DesignModel = &domain.ProductModelInput{
			Name:        r.DesignModel.Name,
			Type:        r.DesignModel.Type,
			Description: r.DesignModel.Description,
			Views:       views,
		}
	}

	return input
}

func ToProductResponse(p *domain.Product) ProductResponse {
	resp := ProductResponse{
		ID:            p.ID,
		CategoryID:    p.CategoryID,
		Name:          p.Name,
		Description:   p.Description,
		BasePrice:     p.BasePrice,
		Slug:          p.Slug,
		GSMInfo:       p.GSMInfo,
		FabricSummary: p.FabricSummary,
		Rating:        p.Rating,
		SoldCount:     p.SoldCount,
		ReviewCount:   p.ReviewCount,
		KeyFeatures:   p.KeyFeatures,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}

	if p.Category != nil {
		catResp := ToCategoryResponse(p.Category)
		resp.Category = &catResp
	}

	if len(p.Images) > 0 {
		for _, img := range p.Images {
			resp.Images = append(resp.Images, ProductImageResponse{
				ID:        img.ID,
				ImageURL:  img.ImageURL,
				IsPrimary: img.IsPrimary,
			})
		}
	}

	if len(p.Fabrics) > 0 {
		for _, fab := range p.Fabrics {
			var colors []FabricColorResponse
			for _, c := range fab.Colors {
				colors = append(colors, FabricColorResponse{
					ID:      c.ID,
					Name:    c.Name,
					HexCode: c.HexCode,
				})
			}
			resp.Fabrics = append(resp.Fabrics, ProductFabricResponse{
				ID:              fab.ID,
				SpecTemplateID:  fab.SpecTemplateID,
				Name:            fab.Name,
				Description:     fab.Description,
				Composition:     fab.Composition,
				CareInstruction: fab.CareInstruction,
				BasePrice:       fab.BasePrice,
				PriceAdjustment: fab.PriceAdjustment,
				IsDefault:       fab.IsDefault,
				Colors:          colors,
			})
		}
	}

	if len(p.Wholesale) > 0 {
		for _, w := range p.Wholesale {
			resp.Wholesale = append(resp.Wholesale, WholesalePriceResponse{
				ID:        w.ID,
				FabricID:  w.FabricID,
				MinQty:    w.MinQty,
				MaxQty:    w.MaxQty,
				UnitPrice: w.UnitPrice,
			})
		}
	}

	if p.DesignModel != nil {
		var views []ProductModelViewResponse
		for _, v := range p.DesignModel.Views {
			views = append(views, ProductModelViewResponse{
				ID:      v.ID,
				Side:    v.Side,
				ArtURL:  v.ArtURL,
				MaskURL: v.MaskURL,
				Width:   v.Width,
				Height:  v.Height,
			})
		}
		resp.DesignModel = &ProductModelResponse{
			ID:          p.DesignModel.ID,
			Name:        p.DesignModel.Name,
			Type:        p.DesignModel.Type,
			Description: p.DesignModel.Description,
			Views:       views,
		}
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
