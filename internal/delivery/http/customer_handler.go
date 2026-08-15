package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type CustomerHandler interface {
	CreateCustomer(c *fiber.Ctx) error
	GetCustomer(c *fiber.Ctx) error
	ListCustomers(c *fiber.Ctx) error
	UpdateCustomer(c *fiber.Ctx) error
	DeleteCustomer(c *fiber.Ctx) error
}

type customerHandler struct {
	customerUsecase domain.CustomerUsecase
}

func NewCustomerHandler(cu domain.CustomerUsecase) CustomerHandler {
	return &customerHandler{
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
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /customers [post]
func (h *customerHandler) CreateCustomer(c *fiber.Ctx) error {
	var req dto.CustomerCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	// Ekstrak User ID dan Role dari JWT Claims
	userID, _ := c.Locals("userID").(string)
	userRole, _ := c.Locals("userRole").(domain.Role)

	// 🚨 ATURAN BISNIS: Jika yang login adalah Sales, paksa SalesID ke ID miliknya sendiri
	salesID := req.SalesID
	if userRole == domain.RoleSales {
		salesID = userID
	}

	domainReq := domain.CustomerCreateInput{
		Name:      req.Name,
		Phone:     req.Phone,
		Address:   req.Address,
		CreatedBy: userID,
		SalesID:   salesID, // Gunakan ID yang sudah divalidasi
	}

	customer, err := h.customerUsecase.CreateCustomer(c.Context(), domainReq)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil mendaftarkan customer", dto.ToCustomerResponse(customer))
}

// @Summary Get Customer
// @Description Mengambil detail data pelanggan berdasarkan ID (Mendukung Data Isolation per Role)
// @Tags Customers
// @Produce json
// @Param id path string true "Customer ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.CustomerResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 401 {object} utils.ErrorResponse
// @Failure 403 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /customers/{id} [get]
func (h *customerHandler) GetCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	// Ambil data identitas login dari context JWT
	operatorID, _ := c.Locals("userID").(string)
	operatorRole, _ := c.Locals("userRole").(domain.Role)

	customer, err := h.customerUsecase.GetCustomer(c.Context(), id, operatorID, operatorRole)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil data customer", dto.ToCustomerResponse(customer))
}

// @Summary List Customers
// @Description Mengambil daftar seluruh pelanggan (terfilter otomatis untuk role Sales)
// @Tags Customers
// @Produce json
// @Param page query int false "Nomor Halaman" default(1)
// @Param limit query int false "Batas Data per Halaman" default(10)
// @Param search query string false "Cari berdasarkan Nama atau Telepon"
// @Param sales_id query string false "Filter berdasarkan Sales ID (Owner/Accounting Only)"
// @Success 200 {object} utils.PaginatedResponse[dto.CustomerResponse]
// @Failure 401 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /customers [get]
func (h *customerHandler) ListCustomers(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	query := domain.PaginationQuery{Page: page, Limit: limit}

	filter := domain.CustomerFilter{
		Search:  c.Query("search"),
		SalesID: c.Query("sales_id"),
	}

	// Ambil data identitas login dari context JWT
	operatorID, _ := c.Locals("userID").(string)
	operatorRole, _ := c.Locals("userRole").(domain.Role)

	customers, meta, err := h.customerUsecase.ListCustomers(c.Context(), query, filter, operatorID, operatorRole)
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
// @Failure 401 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /customers/{id} [put]
func (h *customerHandler) UpdateCustomer(c *fiber.Ctx) error {
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

	// Ekstrak User ID dan Role dari JWT Claims
	operatorID, _ := c.Locals("userID").(string)
	operatorRole, _ := c.Locals("userRole").(domain.Role)

	// 🚨 ATURAN BISNIS: Jika yang login adalah Sales, paksa SalesID ke ID miliknya sendiri
	salesID := req.SalesID
	if operatorRole == domain.RoleSales {
		salesID = operatorID
	}

	domainReq := domain.CustomerUpdateInput{
		Name:    req.Name,
		Phone:   req.Phone,
		Address: req.Address,
		SalesID: salesID, // Gunakan ID yang sudah divalidasi
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
// @Failure 401 {object} utils.ErrorResponse
// @Failure 403 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Security BearerAuth
// @Router /customers/{id} [delete]
func (h *customerHandler) DeleteCustomer(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.customerUsecase.DeleteCustomer(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus customer", nil)
}
