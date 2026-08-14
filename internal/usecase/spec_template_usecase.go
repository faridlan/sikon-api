package usecase

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type specTemplateUsecase struct {
	specTemplateRepo domain.SpecTemplateRepository
	contextTimeout   time.Duration
}

func NewSpecTemplateUsecase(str domain.SpecTemplateRepository, timeout time.Duration) domain.SpecTemplateUsecase {
	return &specTemplateUsecase{
		specTemplateRepo: str,
		contextTimeout:   timeout,
	}
}

func (u *specTemplateUsecase) CreateSpecTemplate(c context.Context, input domain.SpecTemplateCreateInput) (*domain.SpecTemplate, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	specTemplate := &domain.SpecTemplate{
		Name:            input.Name,
		Spec:            input.Spec,
		Description:     input.Description,
		Composition:     input.Composition,
		CareInstruction: input.CareInstruction,
	}

	for _, c := range input.Colors {
		specTemplate.Colors = append(specTemplate.Colors, domain.FabricColor{
			Name:    c.Name,
			HexCode: c.HexCode,
		})
	}

	if err := u.specTemplateRepo.Create(ctx, specTemplate); err != nil {
		return nil, err
	}

	return specTemplate, nil
}

func (u *specTemplateUsecase) GetSpecTemplate(c context.Context, id string) (*domain.SpecTemplate, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	specTemplate, err := u.specTemplateRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Template spesifikasi tidak ditemukan")
		}
		return nil, err
	}

	return specTemplate, nil
}

func (u *specTemplateUsecase) ListSpecTemplates(c context.Context, query domain.PaginationQuery) ([]domain.SpecTemplate, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	offset := query.GetOffset()
	limit := query.Limit

	specTemplates, totalItems, err := u.specTemplateRepo.Fetch(ctx, limit, offset)
	if err != nil {
		return nil, domain.PaginationMeta{}, err
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))
	meta := domain.PaginationMeta{
		CurrentPage: query.Page,
		Limit:       limit,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}

	return specTemplates, meta, nil
}

func (u *specTemplateUsecase) UpdateSpecTemplate(c context.Context, id string, input domain.SpecTemplateUpdateInput) (*domain.SpecTemplate, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingSpecTemplate, err := u.specTemplateRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Template spesifikasi tidak ditemukan")
		}
		return nil, err
	}

	if input.Name != "" {
		existingSpecTemplate.Name = input.Name
	}
	if input.Spec != "" {
		existingSpecTemplate.Spec = input.Spec
	}
	if input.Description != "" {
		existingSpecTemplate.Description = input.Description
	}
	if input.Composition != "" {
		existingSpecTemplate.Composition = input.Composition
	}
	if input.CareInstruction != "" {
		existingSpecTemplate.CareInstruction = input.CareInstruction
	}

	if input.Colors != nil {
		var newColors []domain.FabricColor
		for _, c := range input.Colors {
			newColors = append(newColors, domain.FabricColor{
				SpecTemplateID: &id,
				Name:           c.Name,
				HexCode:        c.HexCode,
			})
		}
		existingSpecTemplate.Colors = newColors
	}

	if err := u.specTemplateRepo.Update(ctx, existingSpecTemplate); err != nil {
		return nil, err
	}

	return existingSpecTemplate, nil
}

func (u *specTemplateUsecase) DeleteSpecTemplate(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	_, err := u.specTemplateRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Template spesifikasi tidak ditemukan")
		}
		return err
	}

	return u.specTemplateRepo.Delete(ctx, id)
}
