package http

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type AttendanceHandler interface {
	RecordAttendance(c *fiber.Ctx) error
	RecordBatchAttendance(c *fiber.Ctx) error
	GetAttendance(c *fiber.Ctx) error
	ListAttendances(c *fiber.Ctx) error
	UpdateAttendance(c *fiber.Ctx) error
	DeleteAttendance(c *fiber.Ctx) error
}

type attendanceHandler struct {
	attendanceUsecase domain.AttendanceUsecase
}

func NewAttendanceHandler(au domain.AttendanceUsecase) AttendanceHandler {
	return &attendanceHandler{attendanceUsecase: au}
}

// @Summary Record Attendance Single
// @Tags Attendances
// @Accept json
// @Produce json
// @Param request body dto.AttendanceCreateRequest true "Data Absensi Pekerja"
// @Success 201 {object} utils.SuccessResponse[dto.AttendanceResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 409 {object} utils.ErrorResponse "Conflict - Sudah Absen"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /attendances [post]
func (h *attendanceHandler) RecordAttendance(c *fiber.Ctx) error {
	var req dto.AttendanceCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	userID, _ := c.Locals("userID").(string)

	attDate, err := time.Parse("2006-01-02", req.AttendanceDate)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Format attendance_date harus YYYY-MM-DD")
	}

	var durationIdx float64
	if req.WorkDurationIndex != nil {
		durationIdx = *req.WorkDurationIndex
	}

	input := domain.AttendanceCreateInput{
		WorkerID:          req.WorkerID,
		AttendanceDate:    attDate,
		Status:            domain.AttendanceStatus(req.Status),
		WorkDurationIndex: durationIdx,
		Notes:             req.Notes,
		CreatedByID:       userID,
	}

	att, err := h.attendanceUsecase.RecordAttendance(c.Context(), input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil mencatat absensi pekerja", dto.ToAttendanceResponse(att))
}

// @Summary Record Batch Attendance
// @Tags Attendances
// @Accept json
// @Produce json
// @Param request body dto.BatchAttendanceRequest true "Payload Absensi Masal Harian"
// @Success 201 {object} utils.SuccessResponse[[]dto.AttendanceResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /attendances/batch [post]
func (h *attendanceHandler) RecordBatchAttendance(c *fiber.Ctx) error {
	var req dto.BatchAttendanceRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	userID, _ := c.Locals("userID").(string)

	attDate, err := time.Parse("2006-01-02", req.AttendanceDate)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Format attendance_date harus YYYY-MM-DD")
	}

	var items []domain.BatchAttendanceItem
	for _, item := range req.Items {
		items = append(items, domain.BatchAttendanceItem{
			WorkerID:          item.WorkerID,
			Status:            domain.AttendanceStatus(item.Status),
			WorkDurationIndex: item.WorkDurationIndex,
			Notes:             item.Notes,
		})
	}

	input := domain.BatchAttendanceInput{
		AttendanceDate: attDate,
		Items:          items,
		CreatedByID:    userID,
	}

	attendances, err := h.attendanceUsecase.RecordBatchAttendance(c.Context(), input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil mencatat absensi masal harian", dto.ToAttendanceResponseList(attendances))
}

// @Summary Get Attendance Detail
// @Tags Attendances
// @Produce json
// @Param id path string true "Attendance ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.AttendanceResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /attendances/{id} [get]
func (h *attendanceHandler) GetAttendance(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	att, err := h.attendanceUsecase.GetAttendance(c.Context(), id)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil detail absensi", dto.ToAttendanceResponse(att))
}

// @Summary List All Attendances
// @Tags Attendances
// @Produce json
// @Param page query int false "Halaman" default(1)
// @Param limit query int false "Limit" default(10)
// @Param worker_id query string false "Filter Pekerja (UUID)"
// @Param payroll_id query string false "Filter Payroll (UUID)"
// @Param status query string false "Filter Status Absen (present, half_day, permission, alpha)"
// @Param is_unpaid query boolean false "Filter hanya yang belum digaji (payroll_id null)"
// @Param start_date query string false "Filter Tanggal Awal (YYYY-MM-DD)"
// @Param end_date query string false "Filter Tanggal Akhir (YYYY-MM-DD)"
// @Success 200 {object} utils.PaginatedResponse[dto.AttendanceResponse]
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /attendances [get]
func (h *attendanceHandler) ListAttendances(c *fiber.Ctx) error {
	query := domain.PaginationQuery{
		Page:  c.QueryInt("page", 1),
		Limit: c.QueryInt("limit", 10),
	}

	var workerID, payrollID *string
	if w := c.Query("worker_id"); w != "" {
		workerID = &w
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

	filter := domain.AttendanceFilter{
		WorkerID:  workerID,
		PayrollID: payrollID,
		Status:    domain.AttendanceStatus(c.Query("status")),
		StartDate: startDate,
		EndDate:   endDate,
		IsUnpaid:  c.QueryBool("is_unpaid", false),
	}

	attendances, meta, err := h.attendanceUsecase.ListAttendances(c.Context(), query, filter)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccessPaginated(c, "Berhasil mengambil daftar data absensi", dto.ToAttendanceResponseList(attendances), meta)
}

// @Summary Update Attendance
// @Tags Attendances
// @Accept json
// @Produce json
// @Param id path string true "Attendance ID (UUID)"
// @Param request body dto.AttendanceUpdateRequest true "Data Perubahan Absensi"
// @Success 200 {object} utils.SuccessResponse[dto.AttendanceResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /attendances/{id} [put]
func (h *attendanceHandler) UpdateAttendance(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.AttendanceUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	input := domain.AttendanceUpdateInput{
		Status:            domain.AttendanceStatus(req.Status),
		WorkDurationIndex: req.WorkDurationIndex,
		Notes:             req.Notes,
	}

	att, err := h.attendanceUsecase.UpdateAttendance(c.Context(), id, input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memperbarui data absensi", dto.ToAttendanceResponse(att))
}

// @Summary Delete Attendance
// @Tags Attendances
// @Produce json
// @Param id path string true "Attendance ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /attendances/{id} [delete]
func (h *attendanceHandler) DeleteAttendance(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.attendanceUsecase.DeleteAttendance(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus data absensi", nil)
}
