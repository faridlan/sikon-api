package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type WorkerHandler interface {
	CreateWorker(c *fiber.Ctx) error
	GetWorker(c *fiber.Ctx) error
	ListWorkers(c *fiber.Ctx) error
	UpdateWorker(c *fiber.Ctx) error
	DeleteWorker(c *fiber.Ctx) error
}

type workerHandler struct {
	workerUsecase domain.WorkerUsecase
}

func NewWorkerHandler(wu domain.WorkerUsecase) WorkerHandler {
	return &workerHandler{workerUsecase: wu}
}

// @Summary Create Worker
// @Tags Workers
// @Accept json
// @Produce json
// @Param request body dto.WorkerCreateRequest true "Data Pekerja Baru"
// @Success 201 {object} utils.SuccessResponse[dto.WorkerResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /workers [post]
func (h *workerHandler) CreateWorker(c *fiber.Ctx) error {
	var req dto.WorkerCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	input := domain.WorkerCreateInput{
		UserID:     req.UserID,
		Name:       req.Name,
		Phone:      req.Phone,
		Role:       domain.WorkerRole(req.Role),
		SalaryType: domain.WorkerSalaryType(req.SalaryType),
		DailyRate:  req.DailyRate,
	}

	worker, err := h.workerUsecase.CreateWorker(c.Context(), input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil menambahkan data pekerja baru", dto.ToWorkerResponse(worker))
}

// @Summary Get Worker Detail
// @Tags Workers
// @Produce json
// @Param id path string true "Worker ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.WorkerResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /workers/{id} [get]
func (h *workerHandler) GetWorker(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	worker, err := h.workerUsecase.GetWorker(c.Context(), id)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil detail data pekerja", dto.ToWorkerResponse(worker))
}

// @Summary List All Workers
// @Tags Workers
// @Produce json
// @Param page query int false "Halaman" default(1)
// @Param limit query int false "Limit" default(10)
// @Param search query string false "Pencarian nama atau nomor telepon"
// @Param role query string false "Filter Peran (tailor, cutter, finishing, sales, staff, helper)"
// @Param salary_type query string false "Filter Tipe Gaji (piece_rate, daily, monthly)"
// @Param status query string false "Filter Status (active, inactive)"
// @Success 200 {object} utils.PaginatedResponse[dto.WorkerResponse]
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /workers [get]
func (h *workerHandler) ListWorkers(c *fiber.Ctx) error {
	query := domain.PaginationQuery{
		Page:  c.QueryInt("page", 1),
		Limit: c.QueryInt("limit", 10),
	}

	filter := domain.WorkerFilter{
		Search:     c.Query("search"),
		Role:       domain.WorkerRole(c.Query("role")),
		SalaryType: domain.WorkerSalaryType(c.Query("salary_type")),
		Status:     domain.WorkerStatus(c.Query("status")),
	}

	workers, meta, err := h.workerUsecase.ListWorkers(c.Context(), query, filter)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccessPaginated(c, "Berhasil mengambil daftar data pekerja", dto.ToWorkerResponseList(workers), meta)
}

// @Summary Update Worker
// @Tags Workers
// @Accept json
// @Produce json
// @Param id path string true "Worker ID (UUID)"
// @Param request body dto.WorkerUpdateRequest true "Data Perubahan Pekerja"
// @Success 200 {object} utils.SuccessResponse[dto.WorkerResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /workers/{id} [put]
func (h *workerHandler) UpdateWorker(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.WorkerUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	input := domain.WorkerUpdateInput{
		UserID:     req.UserID,
		Name:       req.Name,
		Phone:      req.Phone,
		Role:       domain.WorkerRole(req.Role),
		SalaryType: domain.WorkerSalaryType(req.SalaryType),
		DailyRate:  req.DailyRate,
		Status:     domain.WorkerStatus(req.Status),
	}

	worker, err := h.workerUsecase.UpdateWorker(c.Context(), id, input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memperbarui data pekerja", dto.ToWorkerResponse(worker))
}

// @Summary Delete Worker
// @Tags Workers
// @Produce json
// @Param id path string true "Worker ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse "Bad Request"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Failure 403 {object} utils.ErrorResponse "Forbidden"
// @Failure 404 {object} utils.ErrorResponse "Not Found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Security BearerAuth
// @Router /workers/{id} [delete]
func (h *workerHandler) DeleteWorker(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.workerUsecase.DeleteWorker(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus data pekerja", nil)
}
