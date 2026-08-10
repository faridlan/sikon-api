package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type ExpenseHandler interface {
	// Kategori
	CreateCategory(c *fiber.Ctx) error
	ListCategories(c *fiber.Ctx) error

	// Transaksi
	CreateExpense(c *fiber.Ctx) error
	GetExpense(c *fiber.Ctx) error
	ListExpenses(c *fiber.Ctx) error
	DeleteExpense(c *fiber.Ctx) error
}

type expenseHandler struct {
	expenseUsecase domain.ExpenseUsecase
}

func NewExpenseHandler(eu domain.ExpenseUsecase) ExpenseHandler {
	return &expenseHandler{expenseUsecase: eu}
}

// ==========================================
// HANDLERS: EXPENSE CATEGORY
// ==========================================

// @Summary Create Expense Category
// @Tags Expense Categories
// @Accept json
// @Produce json
// @Param request body dto.ExpenseCategoryCreateRequest true "Data Kategori Baru"
// @Success 201 {object} utils.SuccessResponse[dto.ExpenseCategoryResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /expenses/categories [post]
func (h *expenseHandler) CreateCategory(c *fiber.Ctx) error {
	var req dto.ExpenseCategoryCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	input := domain.ExpenseCategoryCreateInput{
		Name:        req.Name,
		Type:        req.Type,
		Description: req.Description,
	}

	cat, err := h.expenseUsecase.CreateCategory(c.Context(), input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil membuat Kategori Pengeluaran", dto.ToExpenseCategoryResponse(cat))
}

// @Summary List All Expense Categories
// @Tags Expense Categories
// @Produce json
// @Success 200 {object} utils.SuccessResponse[[]dto.ExpenseCategoryResponse]
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /expenses/categories [get]
func (h *expenseHandler) ListCategories(c *fiber.Ctx) error {
	cats, err := h.expenseUsecase.ListCategories(c.Context())
	if err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil daftar kategori pengeluaran", dto.ToExpenseCategoryResponseList(cats))
}

// ==========================================
// HANDLERS: EXPENSE TRANSACTION
// ==========================================

// @Summary Create Expense
// @Tags Expenses
// @Accept json
// @Produce json
// @Param request body dto.ExpenseCreateRequest true "Data Pengeluaran Baru"
// @Success 201 {object} utils.SuccessResponse[dto.ExpenseResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /expenses [post]
func (h *expenseHandler) CreateExpense(c *fiber.Ctx) error {
	var req dto.ExpenseCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	// Ekstrak User ID dari JWT Claims yang disimpan oleh JWTMiddleware
	userID := req.CreatedByID
	if userID == "" {
		userID, _ = c.Locals("userID").(string)
	}

	input := domain.ExpenseCreateInput{
		ExpenseCategoryID: req.ExpenseCategoryID,
		BatchPoID:         req.BatchPoID,
		Title:             req.Title,
		Amount:            req.Amount,
		ExpenseDate:       req.ExpenseDate,
		Notes:             req.Notes,
		CreatedByID:       userID,
	}

	exp, err := h.expenseUsecase.CreateExpense(c.Context(), input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil mencatat pengeluaran", dto.ToExpenseResponse(exp))
}

// @Summary Get Expense Detail
// @Tags Expenses
// @Produce json
// @Param id path string true "Expense ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.ExpenseResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /expenses/{id} [get]
func (h *expenseHandler) GetExpense(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	exp, err := h.expenseUsecase.GetExpense(c.Context(), id)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil detail pengeluaran", dto.ToExpenseResponse(exp))
}

// @Summary List All Expenses
// @Tags Expenses
// @Produce json
// @Param page query int false "Halaman" default(1)
// @Param limit query int false "Limit" default(10)
// @Param po_id query string false "Filter berdasarkan Batch PO ID (UUID)"
// @Param start_date query string false "Filter Tanggal Awal (YYYY-MM-DD)"
// @Param end_date query string false "Filter Tanggal Akhir (YYYY-MM-DD)"
// @Success 200 {object} utils.PaginatedResponse[dto.ExpenseResponse]
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /expenses [get]
func (h *expenseHandler) ListExpenses(c *fiber.Ctx) error {
	query := domain.PaginationQuery{
		Page:  c.QueryInt("page", 1),
		Limit: c.QueryInt("limit", 10),
	}

	// Parsing parameter opsional dari URL Query
	var poID, startDate, endDate *string

	if p := c.Query("po_id"); p != "" {
		poID = &p
	}
	if s := c.Query("start_date"); s != "" {
		startDate = &s
	}
	if e := c.Query("end_date"); e != "" {
		endDate = &e
	}

	expenses, meta, err := h.expenseUsecase.ListExpenses(c.Context(), query, poID, startDate, endDate)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccessPaginated(c, "Berhasil mengambil daftar pengeluaran", dto.ToExpenseResponseList(expenses), meta)
}

// @Summary Delete Expense
// @Tags Expenses
// @Produce json
// @Param id path string true "Expense ID"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /expenses/{id} [delete]
func (h *expenseHandler) DeleteExpense(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.expenseUsecase.DeleteExpense(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus catatan pengeluaran", nil)
}
