package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type CustomerHandler struct {
	customerUsecase domain.CustomerUsecase
}

func NewCustomerHandler(cu domain.CustomerUsecase) *CustomerHandler {
	return &CustomerHandler{
		customerUsecase: cu,
	}
}

// @Summary Create Customer
// @Description Mendaftarkan pelanggan (customer) baru ke dalam sistem
// @Tags Customers
// @Accept json
// @Produce json
// @Param request body dto.CustomerCreateRequest true "Data Customer Baru"
// @Success 201 {object} utils.SuccessResponse[dto.CustomerResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /customers [post]
func (h *CustomerHandler) CreateCustomer(c *fiber.Ctx) error {
	var req dto.CustomerCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	// TODO: Nanti diisi dari ID User (JWT) saat fitur Auth siap
	domainReq := domain.CustomerCreateInput{
		Name:    req.Name,
		Phone:   req.Phone,
		Address: req.Address,
	}

	customer, err := h.customerUsecase.CreateCustomer(c.Context(), domainReq)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil mendaftarkan customer", dto.ToCustomerResponse(customer))
}

// @Summary Get Customer
// @Description Mengambil detail data pelanggan berdasarkan ID
// @Tags Customers
// @Produce json
// @Param id path string true "Customer ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.CustomerResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /customers/{id} [get]
func (h *CustomerHandler) GetCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	customer, err := h.customerUsecase.GetCustomer(c.Context(), id)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil data customer", dto.ToCustomerResponse(customer))
}

// @Summary List Customers
// @Description Mengambil daftar seluruh pelanggan dengan pagination
// @Tags Customers
// @Produce json
// @Param page query int false "Nomor Halaman" default(1)
// @Param limit query int false "Batas Data per Halaman" default(10)
// @Success 200 {object} utils.PaginatedResponse[dto.CustomerResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Router /customers [get]
func (h *CustomerHandler) ListCustomers(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	query := domain.PaginationQuery{Page: page, Limit: limit}

	customers, meta, err := h.customerUsecase.ListCustomers(c.Context(), query)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccessPaginated(c, "Berhasil mengambil daftar customer", dto.ToCustomerResponseList(customers), meta)
}

// @Summary Update Customer
// @Description Memperbarui data pelanggan
// @Tags Customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID (UUID)"
// @Param request body dto.CustomerUpdateRequest true "Data Update Customer"
// @Success 200 {object} utils.SuccessResponse[dto.CustomerResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /customers/{id} [put]
func (h *CustomerHandler) UpdateCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.CustomerUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	domainReq := domain.CustomerUpdateInput{
		Name:    req.Name,
		Phone:   req.Phone,
		Address: req.Address,
	}

	customer, err := h.customerUsecase.UpdateCustomer(c.Context(), id, domainReq)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memperbarui data customer", dto.ToCustomerResponse(customer))
}

// @Summary Delete Customer
// @Description Menghapus data pelanggan secara permanen
// @Tags Customers
// @Produce json
// @Param id path string true "Customer ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /customers/{id} [delete]
func (h *CustomerHandler) DeleteCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.customerUsecase.DeleteCustomer(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus customer", nil)
}
