package http

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type PayrollHandler interface {
	CreatePayroll(c *fiber.Ctx) error
	GetPayroll(c *fiber.Ctx) error
	ListPayrolls(c *fiber.Ctx) error
	ProcessPayment(c *fiber.Ctx) error
	DeletePayroll(c *fiber.Ctx) error
}

type payrollHandler struct {
	payrollUsecase domain.PayrollUsecase
}

func NewPayrollHandler(pu domain.PayrollUsecase) PayrollHandler {
	return &payrollHandler{payrollUsecase: pu}
}

// @Summary Create Payroll Rekap
// @Tags Payrolls
// @Accept json
// @Produce json
// @Param request body dto.PayrollCreateRequest true "Data Rekap Gaji Baru"
// @Success 201 {object} utils.SuccessResponse[dto.PayrollResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /payrolls [post]
func (h *payrollHandler) CreatePayroll(c *fiber.Ctx) error {
	var req dto.PayrollCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	userID, _ := c.Locals("userID").(string)

	startDate, errStart := time.Parse("2006-01-02", req.StartDate)
	endDate, errEnd := time.Parse("2006-01-02", req.EndDate)

	if errStart != nil || errEnd != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Format start_date dan end_date harus YYYY-MM-DD")
	}

	input := domain.PayrollCreateInput{
		StartDate:     startDate,
		EndDate:       endDate,
		WorkLogIDs:    req.WorkLogIDs,
		AttendanceIDs: req.AttendanceIDs, // 👈 Passing AttendanceIDs ke Usecase
		CreatedByID:   userID,
	}

	payroll, err := h.payrollUsecase.CreatePayroll(c.Context(), input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil membuat rekap penggajian", dto.ToPayrollResponse(payroll))
}

// @Summary Get Payroll Detail
// @Tags Payrolls
// @Produce json
// @Param id path string true "Payroll ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.PayrollResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /payrolls/{id} [get]
func (h *payrollHandler) GetPayroll(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	payroll, err := h.payrollUsecase.GetPayroll(c.Context(), id)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil detail rekap penggajian", dto.ToPayrollResponse(payroll))
}

// @Summary List All Payrolls
// @Tags Payrolls
// @Produce json
// @Param page query int false "Halaman" default(1)
// @Param limit query int false "Limit" default(10)
// @Param status query string false "Filter Status (draft, approved, paid)"
// @Param start_date query string false "Filter Tanggal Awal (YYYY-MM-DD)"
// @Param end_date query string false "Filter Tanggal Akhir (YYYY-MM-DD)"
// @Success 200 {object} utils.PaginatedResponse[dto.PayrollResponse]
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /payrolls [get]
func (h *payrollHandler) ListPayrolls(c *fiber.Ctx) error {
	query := domain.PaginationQuery{
		Page:  c.QueryInt("page", 1),
		Limit: c.QueryInt("limit", 10),
	}

	var startDate, endDate *time.Time
	if s := c.Query("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			startDate = &t
		}
	}
	if e := c.Query("end_date"); e != "" {
		if t, err := time.Parse("2006-01-02", e); err == nil {
			endOfDay := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
			endDate = &endOfDay
		}
	}

	filter := domain.PayrollFilter{
		Status:    domain.PayrollStatus(c.Query("status")),
		StartDate: startDate,
		EndDate:   endDate,
	}

	payrolls, meta, err := h.payrollUsecase.ListPayrolls(c.Context(), query, filter)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccessPaginated(c, "Berhasil mengambil daftar rekap penggajian", dto.ToPayrollResponseList(payrolls), meta)
}

// @Summary Process Payroll Payment (Approve & Auto Create Expense)
// @Tags Payrolls
// @Produce json
// @Param id path string true "Payroll ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.PayrollResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /payrolls/{id}/pay [post]
func (h *payrollHandler) ProcessPayment(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	userID, _ := c.Locals("userID").(string)

	payroll, err := h.payrollUsecase.ProcessPayrollPayment(c.Context(), id, userID)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memproses pembayaran gaji dan mencatat pengeluaran HPP", dto.ToPayrollResponse(payroll))
}

// @Summary Delete Payroll
// @Tags Payrolls
// @Produce json
// @Param id path string true "Payroll ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /payrolls/{id} [delete]
func (h *payrollHandler) DeletePayroll(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.payrollUsecase.DeletePayroll(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus rekap penggajian", nil)
}
