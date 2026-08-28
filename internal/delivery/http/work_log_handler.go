package http

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type WorkLogHandler interface {
	CreateWorkLog(c *fiber.Ctx) error
	DistributeWorkLoad(c *fiber.Ctx) error // 👈 Tambah Interface Distribute
	GetWorkLog(c *fiber.Ctx) error
	ListWorkLogs(c *fiber.Ctx) error
	UpdateWorkLog(c *fiber.Ctx) error
	DeleteWorkLog(c *fiber.Ctx) error
}

type workLogHandler struct {
	workLogUsecase domain.WorkLogUsecase
}

func NewWorkLogHandler(wlu domain.WorkLogUsecase) WorkLogHandler {
	return &workLogHandler{workLogUsecase: wlu}
}

// @Summary Create Work Log
// @Tags Work Logs
// @Accept json
// @Produce json
// @Param request body dto.WorkLogCreateRequest true "Data Log Borongan Baru"
// @Success 201 {object} utils.SuccessResponse[dto.WorkLogResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /work-logs [post]
func (h *workLogHandler) CreateWorkLog(c *fiber.Ctx) error {
	var req dto.WorkLogCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	userID, _ := c.Locals("userID").(string)

	workDate, err := time.Parse("2006-01-02", req.WorkDate)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Format work_date harus YYYY-MM-DD")
	}

	input := domain.WorkLogCreateInput{
		WorkerID:    req.WorkerID,
		BatchPoID:   req.BatchPoID,
		OrderID:     req.OrderID, // 👈 Map OrderID
		JobType:     domain.JobType(req.JobType),
		Qty:         req.Qty,
		RatePerQty:  req.RatePerQty,
		WorkDate:    workDate,
		Notes:       req.Notes,
		CreatedByID: userID,
	}

	log, err := h.workLogUsecase.CreateWorkLog(c.Context(), input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil mencatat hasil kerja borongan", dto.ToWorkLogResponse(log))
}

// @Summary Auto Distribute Work Load per Batch PO
// @Tags Work Logs
// @Accept json
// @Produce json
// @Param request body dto.WorkLogDistributeRequest true "Payload Auto Distribute Borongan"
// @Success 201 {object} utils.SuccessResponse[[]dto.WorkLogResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /work-logs/distribute [post]
func (h *workLogHandler) DistributeWorkLoad(c *fiber.Ctx) error {
	var req dto.WorkLogDistributeRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	userID, _ := c.Locals("userID").(string)

	var workDate time.Time
	if req.WorkDate != "" {
		parsed, err := time.Parse("2006-01-02", req.WorkDate)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, "Format work_date harus YYYY-MM-DD")
		}
		workDate = parsed
	}

	input := domain.DistributeWorkLoadInput{
		BatchPOID:   req.BatchPOID,
		JobType:     domain.JobType(req.JobType),
		WorkerIDs:   req.WorkerIDs,
		RatePerQty:  req.RatePerQty,
		WorkDate:    workDate,
		Notes:       req.Notes,
		CreatedByID: userID,
	}

	logs, err := h.workLogUsecase.DistributeWorkLoad(c.Context(), input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil mendistribusikan beban borongan secara otomatis", dto.ToWorkLogResponseList(logs))
}

// @Summary Get Work Log Detail
// @Tags Work Logs
// @Produce json
// @Param id path string true "WorkLog ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.WorkLogResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /work-logs/{id} [get]
func (h *workLogHandler) GetWorkLog(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	log, err := h.workLogUsecase.GetWorkLog(c.Context(), id)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil detail log borongan", dto.ToWorkLogResponse(log))
}

// @Summary List All Work Logs
// @Tags Work Logs
// @Produce json
// @Param page query int false "Halaman" default(1)
// @Param limit query int false "Limit" default(10)
// @Param worker_id query string false "Filter Pekerja (UUID)"
// @Param batch_po_id query string false "Filter Batch PO (UUID)"
// @Param order_id query string false "Filter Order Konsumen (UUID)"
// @Param payroll_id query string false "Filter Payroll (UUID)"
// @Param is_unpaid query boolean false "Filter hanya yang belum digaji (payroll_id null)"
// @Param job_type query string false "Filter Jenis Pekerjaan (jahit, potong, bordir, finishing)"
// @Param start_date query string false "Filter Tanggal Awal (YYYY-MM-DD)"
// @Param end_date query string false "Filter Tanggal Akhir (YYYY-MM-DD)"
// @Success 200 {object} utils.PaginatedResponse[dto.WorkLogResponse]
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /work-logs [get]
func (h *workLogHandler) ListWorkLogs(c *fiber.Ctx) error {
	query := domain.PaginationQuery{
		Page:  c.QueryInt("page", 1),
		Limit: c.QueryInt("limit", 10),
	}

	var workerID, batchPoID, orderID, payrollID *string
	if w := c.Query("worker_id"); w != "" {
		workerID = &w
	}
	if b := c.Query("batch_po_id"); b != "" {
		batchPoID = &b
	}
	if o := c.Query("order_id"); o != "" { // 👈 Filter OrderID
		orderID = &o
	}
	if p := c.Query("payroll_id"); p != "" {
		payrollID = &p
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

	filter := domain.WorkLogFilter{
		WorkerID:  workerID,
		BatchPoID: batchPoID,
		OrderID:   orderID, // 👈 Send to Filter
		PayrollID: payrollID,
		JobType:   domain.JobType(c.Query("job_type")),
		StartDate: startDate,
		EndDate:   endDate,
		IsUnpaid:  c.QueryBool("is_unpaid", false),
	}

	logs, meta, err := h.workLogUsecase.ListWorkLogs(c.Context(), query, filter)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccessPaginated(c, "Berhasil mengambil daftar log borongan", dto.ToWorkLogResponseList(logs), meta)
}

// @Summary Update Work Log
// @Tags Work Logs
// @Accept json
// @Produce json
// @Param id path string true "WorkLog ID (UUID)"
// @Param request body dto.WorkLogUpdateRequest true "Data Perubahan Work Log"
// @Success 200 {object} utils.SuccessResponse[dto.WorkLogResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /work-logs/{id} [put]
func (h *workLogHandler) UpdateWorkLog(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.WorkLogUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var workDate time.Time
	if req.WorkDate != "" {
		parsed, err := time.Parse("2006-01-02", req.WorkDate)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, "Format work_date harus YYYY-MM-DD")
		}
		workDate = parsed
	}

	input := domain.WorkLogUpdateInput{
		WorkerID:   req.WorkerID,
		BatchPoID:  req.BatchPoID,
		OrderID:    req.OrderID, // 👈 Map OrderID
		JobType:    domain.JobType(req.JobType),
		Qty:        req.Qty,
		RatePerQty: req.RatePerQty,
		WorkDate:   workDate,
		Notes:      req.Notes,
	}

	log, err := h.workLogUsecase.UpdateWorkLog(c.Context(), id, input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memperbarui data log borongan", dto.ToWorkLogResponse(log))
}

// @Summary Delete Work Log
// @Tags Work Logs
// @Produce json
// @Param id path string true "WorkLog ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /work-logs/{id} [delete]
func (h *workLogHandler) DeleteWorkLog(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.workLogUsecase.DeleteWorkLog(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus catatan log borongan", nil)
}
