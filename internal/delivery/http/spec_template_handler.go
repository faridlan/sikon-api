package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type SpecTemplateHandler interface {
	CreateSpecTemplate(c *fiber.Ctx) error
	GetSpecTemplate(c *fiber.Ctx) error
	ListSpecTemplates(c *fiber.Ctx) error
	UpdateSpecTemplate(c *fiber.Ctx) error
	DeleteSpecTemplate(c *fiber.Ctx) error
}

type specTemplateHandler struct {
	specTemplateUsecase domain.SpecTemplateUsecase
}

func NewSpecTemplateHandler(su domain.SpecTemplateUsecase) SpecTemplateHandler {
	return &specTemplateHandler{
		specTemplateUsecase: su,
	}
}

// @Summary Create Spec Template
// @Description Membuat master kain global / template spesifikasi baru
// @Tags SpecTemplates
// @Accept json
// @Produce json
// @Param request body dto.SpecTemplateRequest true "Data Template Spesifikasi Baru"
// @Success 201 {object} utils.SuccessResponse[dto.SpecTemplateResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /spec-templates [post]
func (h *specTemplateHandler) CreateSpecTemplate(c *fiber.Ctx) error {
	var req dto.SpecTemplateRequest

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}

	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}
	var colors []domain.FabricColorInput
	for _, c := range req.Colors {
		colors = append(colors, domain.FabricColorInput{
			Name:    c.Name,
			HexCode: c.HexCode,
		})
	}

	domainReq := domain.SpecTemplateCreateInput{
		Name:            req.Name,
		Spec:            req.Spec,
		Description:     req.Description,
		Composition:     req.Composition,
		CareInstruction: req.CareInstruction,
		Colors:          colors,
	}

	specTemplate, err := h.specTemplateUsecase.CreateSpecTemplate(c.Context(), domainReq)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil membuat template spesifikasi", dto.ToSpecTemplateResponse(specTemplate))
}

// @Summary Get Spec Template
// @Description Mengambil detail template spesifikasi berdasarkan ID
// @Tags SpecTemplates
// @Produce json
// @Param id path string true "SpecTemplate ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.SpecTemplateResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /spec-templates/{id} [get]
func (h *specTemplateHandler) GetSpecTemplate(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	specTemplate, err := h.specTemplateUsecase.GetSpecTemplate(c.Context(), id)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil template spesifikasi", dto.ToSpecTemplateResponse(specTemplate))
}

// @Summary List Spec Templates
// @Description Mengambil daftar seluruh template spesifikasi dengan pagination
// @Tags SpecTemplates
// @Produce json
// @Param page query int false "Nomor Halaman" default(1)
// @Param limit query int false "Batas Data per Halaman" default(10)
// @Success 200 {object} utils.PaginatedResponse[dto.SpecTemplateResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /spec-templates [get]
func (h *specTemplateHandler) ListSpecTemplates(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	query := domain.PaginationQuery{
		Page:  page,
		Limit: limit,
	}

	specTemplates, meta, err := h.specTemplateUsecase.ListSpecTemplates(c.Context(), query)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccessPaginated(c, "Berhasil mengambil daftar template spesifikasi", dto.ToSpecTemplateResponseList(specTemplates), meta)
}

// @Summary Update Spec Template
// @Description Memperbarui nama dan detail spesifikasi master kain global
// @Tags SpecTemplates
// @Accept json
// @Produce json
// @Param id path string true "SpecTemplate ID (UUID)"
// @Param request body dto.SpecTemplateRequest true "Data Update Spec Template"
// @Success 200 {object} utils.SuccessResponse[dto.SpecTemplateResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /spec-templates/{id} [put]
func (h *specTemplateHandler) UpdateSpecTemplate(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.SpecTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}

	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var colors []domain.FabricColorInput
	for _, c := range req.Colors {
		colors = append(colors, domain.FabricColorInput{
			Name:    c.Name,
			HexCode: c.HexCode,
		})
	}

	domainReq := domain.SpecTemplateUpdateInput{
		Name:            req.Name,
		Spec:            req.Spec,
		Description:     req.Description,
		Composition:     req.Composition,
		CareInstruction: req.CareInstruction,
		Colors:          colors,
	}

	specTemplate, err := h.specTemplateUsecase.UpdateSpecTemplate(c.Context(), id, domainReq)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memperbarui template spesifikasi", dto.ToSpecTemplateResponse(specTemplate))
}

// @Summary Delete Spec Template
// @Description Menghapus template spesifikasi (Soft Delete / Arsip)
// @Tags SpecTemplates
// @Produce json
// @Param id path string true "SpecTemplate ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /spec-templates/{id} [delete]
func (h *specTemplateHandler) DeleteSpecTemplate(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.specTemplateUsecase.DeleteSpecTemplate(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus template spesifikasi", nil)
}
