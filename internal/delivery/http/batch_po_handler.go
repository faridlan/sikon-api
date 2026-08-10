package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type BatchPOHandler interface {
	CreateBatchPO(c *fiber.Ctx) error
	GetBatchPO(c *fiber.Ctx) error
	ListBatchPOs(c *fiber.Ctx) error
	ListActiveBatchPOs(c *fiber.Ctx) error
	UpdateBatchPO(c *fiber.Ctx) error
	UpdateStatus(c *fiber.Ctx) error
	DeleteBatchPO(c *fiber.Ctx) error
}

type batchPoHandler struct {
	batchPoUsecase domain.BatchPOUsecase
}

func NewBatchPOHandler(bu domain.BatchPOUsecase) BatchPOHandler {
	return &batchPoHandler{batchPoUsecase: bu}
}

// @Summary Create Batch PO
// @Tags BatchPOs
// @Accept json
// @Produce json
// @Param request body dto.BatchPOCreateRequest true "Data Batch PO Baru"
// @Success 201 {object} utils.SuccessResponse[dto.BatchPOResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request - Format input salah atau validasi gagal"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error - Terjadi kesalahan pada server"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /batch-pos [post]
func (h *batchPoHandler) CreateBatchPO(c *fiber.Ctx) error {
	var req dto.BatchPOCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	input := domain.BatchPOCreateInput{
		Name:        req.Name,
		TargetMonth: req.TargetMonth, // <-- Meneruskan TargetMonth dari JSON
		TargetYear:  req.TargetYear,  // <-- Meneruskan TargetYear dari JSON
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Quota:       req.Quota,
	}

	batchPO, err := h.batchPoUsecase.CreateBatchPO(c.Context(), input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil membuat Batch PO", dto.ToBatchPOResponse(batchPO))
}

// @Summary Get Batch PO Detail
// @Tags BatchPOs
// @Produce json
// @Param id path string true "Batch PO ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.BatchPOResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request - Format ID UUID tidak valid"
// @Failure 404 {object} utils.ErrorResponse "Not Found - Batch PO tidak ditemukan"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /batch-pos/{id} [get]
func (h *batchPoHandler) GetBatchPO(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	batchPO, err := h.batchPoUsecase.GetBatchPO(c.Context(), id)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil detail Batch PO", dto.ToBatchPOResponse(batchPO))
}

// @Summary List All Batch POs (Pagination)
// @Tags BatchPOs
// @Produce json
// @Param page query int false "Halaman" default(1)
// @Param limit query int false "Limit" default(10)
// @Success 200 {object} utils.PaginatedResponse[dto.BatchPOResponse]
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /batch-pos [get]
func (h *batchPoHandler) ListBatchPOs(c *fiber.Ctx) error {
	query := domain.PaginationQuery{
		Page:  c.QueryInt("page", 1),
		Limit: c.QueryInt("limit", 10),
	}

	batchPOs, meta, err := h.batchPoUsecase.ListBatchPOs(c.Context(), query)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccessPaginated(c, "Berhasil mengambil daftar Batch PO", dto.ToBatchPOResponseList(batchPOs), meta)
}

// @Summary List Active Batch POs (For Dropdown)
// @Tags BatchPOs
// @Produce json
// @Success 200 {object} utils.SuccessResponse[[]dto.BatchPOResponse]
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /batch-pos/active [get]
func (h *batchPoHandler) ListActiveBatchPOs(c *fiber.Ctx) error {
	batchPOs, err := h.batchPoUsecase.ListActiveBatchPOs(c.Context())
	if err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil daftar Batch PO aktif", dto.ToBatchPOResponseList(batchPOs))
}

// @Summary Update Batch PO
// @Tags BatchPOs
// @Accept json
// @Produce json
// @Param id path string true "Batch PO ID"
// @Param request body dto.BatchPOUpdateRequest true "Data Update PO"
// @Success 200 {object} utils.SuccessResponse[dto.BatchPOResponse]
// @Failure 400 {object} utils.ErrorResponse "Bad Request - Format ID tidak valid atau body request salah"
// @Failure 404 {object} utils.ErrorResponse "Not Found - Batch PO tidak ditemukan"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /batch-pos/{id} [put]
func (h *batchPoHandler) UpdateBatchPO(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.BatchPOUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	input := domain.BatchPOUpdateInput{
		Name:        req.Name,
		TargetMonth: req.TargetMonth, // <-- Meneruskan TargetMonth dari JSON
		TargetYear:  req.TargetYear,  // <-- Meneruskan TargetYear dari JSON
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Quota:       req.Quota,
	}

	batchPO, err := h.batchPoUsecase.UpdateBatchPO(c.Context(), id, input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memperbarui Batch PO", dto.ToBatchPOResponse(batchPO))
}

// @Summary Update Batch PO Status (Activate / Close)
// @Tags BatchPOs
// @Accept json
// @Produce json
// @Param id path string true "Batch PO ID"
// @Param request body dto.BatchPOStatusUpdateRequest true "Status Baru"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse "Bad Request - Status tidak valid"
// @Failure 404 {object} utils.ErrorResponse "Not Found - Batch PO tidak ditemukan"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /batch-pos/{id}/status [patch]
func (h *batchPoHandler) UpdateStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.BatchPOStatusUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.batchPoUsecase.UpdateStatus(c.Context(), id, domain.BatchPOStatus(req.Status)); err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengubah status Batch PO", nil)
}

// @Summary Delete Batch PO (Soft Delete)
// @Tags BatchPOs
// @Produce json
// @Param id path string true "Batch PO ID"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse "Bad Request - Format ID tidak valid"
// @Failure 404 {object} utils.ErrorResponse "Not Found - Batch PO tidak ditemukan"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Failure 401 {object} utils.ErrorResponse "Unauthorized"
// @Security BearerAuth
// @Router /batch-pos/{id} [delete]
func (h *batchPoHandler) DeleteBatchPO(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.batchPoUsecase.DeleteBatchPO(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}
	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus Batch PO", nil)
}
