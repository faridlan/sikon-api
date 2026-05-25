package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type PaymentHandler interface {
	ProcessPayment(c *fiber.Ctx) error
	GetPayment(c *fiber.Ctx) error
	ListPayments(c *fiber.Ctx) error
	UpdatePayment(c *fiber.Ctx) error
	DeletePayment(c *fiber.Ctx) error
	GetPaymentsByOrderID(c *fiber.Ctx) error
}

type paymentHandler struct {
	paymentUsecase domain.PaymentUsecase
}

func NewPaymentHandler(pu domain.PaymentUsecase) PaymentHandler {
	return &paymentHandler{
		paymentUsecase: pu,
	}
}

// @Summary Process Payment
// @Description Memproses pembayaran (DP/Lunas) untuk sebuah pesanan
// @Tags Payments
// @Accept json
// @Produce json
// @Param request body dto.PaymentCreateRequest true "Data Pembayaran"
// @Success 201 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 409 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /payments [post]
func (h *paymentHandler) ProcessPayment(c *fiber.Ctx) error {
	var req dto.PaymentCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	domainReq := domain.PaymentCreateInput{
		OrderID:         req.OrderID,
		BankAccountID:   req.BankAccountID,
		Amount:          req.Amount,
		PaymentDate:     req.PaymentDate,
		ReferenceNumber: req.ReferenceNumber,
		PaymentType:     domain.PaymentType(req.PaymentType),
	}

	if err := h.paymentUsecase.ProcessPayment(c.Context(), domainReq); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil memproses pembayaran", nil)
}

// @Summary Get Payment
// @Description Mengambil detail data pembayaran berdasarkan ID
// @Tags Payments
// @Produce json
// @Param id path string true "Payment ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.PaymentResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /payments/{id} [get]
func (h *paymentHandler) GetPayment(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	payment, err := h.paymentUsecase.GetPayment(c.Context(), id)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil data pembayaran", dto.ToPaymentResponse(payment))
}

// @Summary List Payments
// @Description Mengambil daftar seluruh pembayaran (History transaksi)
// @Tags Payments
// @Produce json
// @Param page query int false "Nomor Halaman" default(1)
// @Param limit query int false "Batas Data per Halaman" default(10)
// @Success 200 {object} utils.PaginatedResponse[dto.PaymentResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Router /payments [get]
func (h *paymentHandler) ListPayments(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	query := domain.PaginationQuery{Page: page, Limit: limit}

	payments, meta, err := h.paymentUsecase.ListPayments(c.Context(), query)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccessPaginated(c, "Berhasil mengambil daftar pembayaran", dto.ToPaymentResponseList(payments), meta)
}

// @Summary Update Payment
// @Description Memperbarui nomor referensi atau tipe pembayaran
// @Tags Payments
// @Accept json
// @Produce json
// @Param id path string true "Payment ID (UUID)"
// @Param request body dto.PaymentUpdateRequest true "Data Update Pembayaran"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /payments/{id} [put]
func (h *paymentHandler) UpdatePayment(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.PaymentUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	domainReq := domain.PaymentUpdateInput{
		ReferenceNumber: req.ReferenceNumber,
		PaymentType:     domain.PaymentType(req.PaymentType),
	}

	if err := h.paymentUsecase.UpdatePayment(c.Context(), id, domainReq); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memperbarui data pembayaran", nil)
}

// @Summary Delete Payment
// @Description Menghapus data pembayaran secara permanen
// @Tags Payments
// @Produce json
// @Param id path string true "Payment ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /payments/{id} [delete]
func (h *paymentHandler) DeletePayment(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.paymentUsecase.DeletePayment(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus pembayaran", nil)
}

// @Summary Get Payments by Order ID
// @Description Mengambil daftar pembayaran berdasarkan ID pesanan
// @Tags Payments
// @Produce json
// @Param order_id path string true "Order ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[[]dto.PaymentResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /orders/{order_id}/payments [get]
func (h *paymentHandler) GetPaymentsByOrderID(c *fiber.Ctx) error {
	orderID := c.Params("order_id")
	if err := utils.ValidateUUID(orderID, "order_id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	payments, err := h.paymentUsecase.GetPaymentsByOrderID(c.Context(), orderID)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil daftar pembayaran untuk pesanan", dto.ToPaymentResponseList(payments))
}
