package domain

import (
	"context"
	"time"
)

type SpecTemplate struct {
	ID              string
	Name            string
	Spec            string
	Description     string
	Composition     string
	CareInstruction string
	CreatedAt       time.Time
	UpdatedAt       time.Time

	// Penambahan field baru (aman)
	Colors []FabricColor
}

type SpecTemplateCreateInput struct {
	Name            string
	Spec            string
	Description     string
	Composition     string
	CareInstruction string

	// Penambahan field baru (aman)
	Colors []FabricColorInput
}

type SpecTemplateUpdateInput struct {
	Name            string
	Spec            string
	Description     string
	Composition     string
	CareInstruction string

	// Penambahan field baru (aman)
	Colors []FabricColorInput
}

type SpecTemplateRepository interface {
	Create(ctx context.Context, specTemplate *SpecTemplate) error
	GetByID(ctx context.Context, id string) (*SpecTemplate, error)
	Fetch(ctx context.Context, limit, offset int) ([]SpecTemplate, int64, error)
	Update(ctx context.Context, specTemplate *SpecTemplate) error
	Delete(ctx context.Context, id string) error
}

type SpecTemplateUsecase interface {
	CreateSpecTemplate(ctx context.Context, input SpecTemplateCreateInput) (*SpecTemplate, error)
	GetSpecTemplate(ctx context.Context, id string) (*SpecTemplate, error)
	ListSpecTemplates(c context.Context, query PaginationQuery) ([]SpecTemplate, PaginationMeta, error)
	UpdateSpecTemplate(ctx context.Context, id string, input SpecTemplateUpdateInput) (*SpecTemplate, error)
	DeleteSpecTemplate(ctx context.Context, id string) error
}
