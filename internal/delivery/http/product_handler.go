package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type ProductHandler interface {
	CreateProduct(c *fiber.Ctx) error
	GetProduct(c *fiber.Ctx) error
	ListProducts(c *fiber.Ctx) error
	UpdateProduct(c *fiber.Ctx) error
	DeleteProduct(c *fiber.Ctx) error
}

type productHandler struct {
	productUsecase domain.ProductUsecase
}

func NewProductHandler(pu domain.ProductUsecase) ProductHandler {
	return &productHandler{
		productUsecase: pu,
	}
}

// @Summary Create Product
// @Description Menambahkan produk baru ke dalam katalog
// @Tags Products
// @Accept json
// @Produce json
// @Param request body dto.ProductCreateRequest true "Data Produk Baru"
// @Success 201 {object} utils.SuccessResponse[dto.ProductResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /products [post]
func (h *productHandler) CreateProduct(c *fiber.Ctx) error {
	var req dto.ProductCreateRequest

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}

	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	domainReq := domain.ProductCreateInput{
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Description: req.Description,
		BasePrice:   req.BasePrice,
		ImageURL:    req.ImageURL,
	}

	product, err := h.productUsecase.CreateProduct(c.Context(), domainReq)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil membuat produk", dto.ToProductResponse(product))
}

// @Summary Get Product
// @Description Mengambil detail data produk berdasarkan ID
// @Tags Products
// @Produce json
// @Param id path string true "Product ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.ProductResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /products/{id} [get]
func (h *productHandler) GetProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	product, err := h.productUsecase.GetProduct(c.Context(), id)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil data produk", dto.ToProductResponse(product))
}

// @Summary List Products
// @Description Mengambil daftar seluruh produk dengan pagination
// @Tags Products
// @Produce json
// @Param page query int false "Nomor Halaman" default(1)
// @Param limit query int false "Batas Data per Halaman" default(10)
// @Success 200 {object} utils.PaginatedResponse[dto.ProductResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Router /products [get]
func (h *productHandler) ListProducts(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	filter := domain.ProductFilter{
		Search:     c.Query("search"),
		CategoryID: c.Query("category_id"),
	}

	query := domain.PaginationQuery{Page: page, Limit: limit}

	products, meta, err := h.productUsecase.ListProducts(c.Context(), filter, query)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccessPaginated(c, "Berhasil mengambil daftar produk", dto.ToProductResponseList(products), meta)
}

// @Summary Update Product
// @Description Memperbarui data produk (nama, harga, kategori)
// @Tags Products
// @Accept json
// @Produce json
// @Param id path string true "Product ID (UUID)"
// @Param request body dto.ProductUpdateRequest true "Data Update Produk"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /products/{id} [put]
func (h *productHandler) UpdateProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.ProductUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}

	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	domainReq := domain.ProductUpdateInput{
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Description: req.Description,
		BasePrice:   req.BasePrice,
		ImageURL:    req.ImageURL,
	}

	product, err := h.productUsecase.UpdateProduct(c.Context(), id, domainReq)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memperbarui produk", dto.ToProductResponse(product))
}

// @Summary Delete Product
// @Description Menghapus data produk secara permanen
// @Tags Products
// @Produce json
// @Param id path string true "Product ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /products/{id} [delete]
func (h *productHandler) DeleteProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.productUsecase.DeleteProduct(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus produk", nil)
}
