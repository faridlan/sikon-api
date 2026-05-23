package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type CategoryHandler struct {
	categoryUsecase domain.CategoryUsecase
}

func NewCategoryHandler(cu domain.CategoryUsecase) *CategoryHandler {
	return &CategoryHandler{
		categoryUsecase: cu,
	}
}

// @Summary Create Category
// @Description Membuat kategori produk baru
// @Tags Categories
// @Accept json
// @Produce json
// @Param request body dto.CategoryRequest true "Data Kategori Baru"
// @Success 201 {object} utils.SuccessResponse[dto.CategoryResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /categories [post]
func (h *CategoryHandler) CreateCategory(c *fiber.Ctx) error {
	var req dto.CategoryRequest

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}

	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	category := &domain.Category{
		Name: req.Name,
	}

	if err := h.categoryUsecase.CreateCategory(c.Context(), category); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil membuat kategori", dto.ToCategoryResponse(category))
}

// @Summary Get Category
// @Description Mengambil detail kategori berdasarkan ID
// @Tags Categories
// @Produce json
// @Param id path string true "Category ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.CategoryResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /categories/{id} [get]
func (h *CategoryHandler) GetCategory(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	category, err := h.categoryUsecase.GetCategory(c.Context(), id)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil kategori", dto.ToCategoryResponse(category))
}

// @Summary List Categories
// @Description Mengambil daftar seluruh kategori produk dengan pagination
// @Tags Categories
// @Produce json
// @Param page query int false "Nomor Halaman" default(1)
// @Param limit query int false "Batas Data per Halaman" default(10)
// @Success 200 {object} utils.PaginatedResponse[dto.CategoryResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Router /categories [get]
func (h *CategoryHandler) ListCategories(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	query := domain.PaginationQuery{
		Page:  page,
		Limit: limit,
	}

	categories, meta, err := h.categoryUsecase.ListCategories(c.Context(), query)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccessPaginated(c, "Berhasil mengambil daftar kategori", dto.ToCategoryResponseList(categories), meta)
}

// @Summary Update Category
// @Description Memperbarui nama kategori
// @Tags Categories
// @Accept json
// @Produce json
// @Param id path string true "Category ID (UUID)"
// @Param request body dto.CategoryRequest true "Data Update Kategori"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.CategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}

	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	category := &domain.Category{
		ID:   id,
		Name: req.Name,
	}

	if err := h.categoryUsecase.UpdateCategory(c.Context(), category); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memperbarui kategori", nil)
}

// @Summary Delete Category
// @Description Menghapus kategori secara permanen
// @Tags Categories
// @Produce json
// @Param id path string true "Category ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.categoryUsecase.DeleteCategory(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus kategori", nil)
}
