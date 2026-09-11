package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type MaterialHandler interface {
	CreateMaterial(c *fiber.Ctx) error
	GetMaterial(c *fiber.Ctx) error
	ListMaterials(c *fiber.Ctx) error
	UpdateMaterial(c *fiber.Ctx) error
	DeleteMaterial(c *fiber.Ctx) error

	// Resep / BOM (nested di bawah Product)
	SetProductMaterials(c *fiber.Ctx) error
	GetProductMaterials(c *fiber.Ctx) error
}

type materialHandler struct {
	materialUsecase        domain.MaterialUsecase
	productMaterialUsecase domain.ProductMaterialUsecase
}

func NewMaterialHandler(mu domain.MaterialUsecase, pmu domain.ProductMaterialUsecase) MaterialHandler {
	return &materialHandler{materialUsecase: mu, productMaterialUsecase: pmu}
}

// ==========================================
// HANDLERS: MATERIAL
// ==========================================

// @Summary Create Material
// @Tags Materials
// @Accept json
// @Produce json
// @Param request body dto.MaterialCreateRequest true "Data Material Baru"
// @Success 201 {object} utils.SuccessResponse[dto.MaterialResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /materials [post]
func (h *materialHandler) CreateMaterial(c *fiber.Ctx) error {
	var req dto.MaterialCreateRequest
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

	input := domain.MaterialCreateInput{
		Name:            req.Name,
		Unit:            req.Unit,
		UnitPrice:       req.UnitPrice,
		Category:        req.Category,
		Description:     req.Description,
		Composition:     req.Composition,
		CareInstruction: req.CareInstruction,
		GSMInfo:         req.GSMInfo,
		Colors:          colors,
	}

	material, err := h.materialUsecase.CreateMaterial(c.Context(), input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil membuat material", dto.ToMaterialResponse(material))
}

// @Summary Get Material Detail
// @Tags Materials
// @Produce json
// @Param id path string true "Material ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.MaterialResponse]
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Security BearerAuth
// @Router /materials/{id} [get]
func (h *materialHandler) GetMaterial(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	material, err := h.materialUsecase.GetMaterial(c.Context(), id)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil detail material", dto.ToMaterialResponse(material))
}

// @Summary List All Materials
// @Tags Materials
// @Produce json
// @Param page query int false "Halaman" default(1)
// @Param limit query int false "Limit" default(10)
// @Success 200 {object} utils.PaginatedResponse[dto.MaterialResponse]
// @Security BearerAuth
// @Router /materials [get]
func (h *materialHandler) ListMaterials(c *fiber.Ctx) error {
	query := domain.PaginationQuery{
		Page:  c.QueryInt("page", 1),
		Limit: c.QueryInt("limit", 10),
	}

	materials, meta, err := h.materialUsecase.ListMaterials(c.Context(), query)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccessPaginated(c, "Berhasil mengambil daftar material", dto.ToMaterialResponseList(materials), meta)
}

// @Summary Update Material
// @Tags Materials
// @Accept json
// @Produce json
// @Param id path string true "Material ID (UUID)"
// @Param request body dto.MaterialUpdateRequest true "Data Material yang diubah"
// @Success 200 {object} utils.SuccessResponse[dto.MaterialResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Security BearerAuth
// @Router /materials/{id} [put]
func (h *materialHandler) UpdateMaterial(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.MaterialUpdateRequest
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

	input := domain.MaterialUpdateInput{
		Name:            req.Name,
		Unit:            req.Unit,
		UnitPrice:       req.UnitPrice,
		Category:        req.Category,
		Description:     req.Description,
		Composition:     req.Composition,
		CareInstruction: req.CareInstruction,
		GSMInfo:         req.GSMInfo,
		Colors:          colors,
	}

	material, err := h.materialUsecase.UpdateMaterial(c.Context(), id, input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memperbarui material", dto.ToMaterialResponse(material))
}

// @Summary Delete Material
// @Tags Materials
// @Produce json
// @Param id path string true "Material ID"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Security BearerAuth
// @Router /materials/{id} [delete]
func (h *materialHandler) DeleteMaterial(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.materialUsecase.DeleteMaterial(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus material", nil)
}

// ==========================================
// HANDLERS: PRODUCT MATERIAL (Resep / BOM)
// ==========================================

// @Summary Set Product Materials (Resep Produk)
// @Description Menyimpan seluruh resep bahan untuk sebuah produk (replace-all, bukan tambah satu-satu)
// @Tags Materials
// @Accept json
// @Produce json
// @Param product_id path string true "Product ID (UUID)"
// @Param request body dto.SetProductMaterialsRequest true "Daftar resep bahan"
// @Success 200 {object} utils.SuccessResponse[[]dto.ProductMaterialResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Security BearerAuth
// @Router /products/{product_id}/materials [put]
func (h *materialHandler) SetProductMaterials(c *fiber.Ctx) error {
	productID := c.Params("id")
	if err := utils.ValidateUUID(productID, "product_id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.SetProductMaterialsRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	items := make([]domain.ProductMaterialItemInput, len(req.Items))
	for i, it := range req.Items {
		items[i] = domain.ProductMaterialItemInput{
			MaterialID: it.MaterialID,
			QtyPerUnit: it.QtyPerUnit,
		}
	}

	input := domain.SetProductMaterialsInput{
		ProductID: productID,
		Items:     items,
	}

	result, err := h.productMaterialUsecase.SetProductMaterials(c.Context(), input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menyimpan resep produk", dto.ToProductMaterialResponseList(result))
}

// @Summary Get Product Materials (Resep Produk)
// @Tags Materials
// @Produce json
// @Param product_id path string true "Product ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[[]dto.ProductMaterialResponse]
// @Security BearerAuth
// @Router /products/{product_id}/materials [get]
func (h *materialHandler) GetProductMaterials(c *fiber.Ctx) error {
	productID := c.Params("id")
	if err := utils.ValidateUUID(productID, "product_id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	result, err := h.productMaterialUsecase.GetProductMaterials(c.Context(), productID)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil resep produk", dto.ToProductMaterialResponseList(result))
}
