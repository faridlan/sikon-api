package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type OrderHandler struct {
	orderUsecase domain.OrderUsecase
}

func NewOrderHandler(ou domain.OrderUsecase) *OrderHandler {
	return &OrderHandler{
		orderUsecase: ou,
	}
}

// @Summary Create Order
// @Description Membuat pesanan (order) baru beserta detail produknya
// @Tags Orders
// @Accept json
// @Produce json
// @Param request body dto.OrderCreateRequest true "Data Pesanan Baru"
// @Success 201 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /orders [post]
func (h *OrderHandler) CreateOrder(c *fiber.Ctx) error {
	var req dto.OrderCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	domainReq := domain.OrderCreateInput{
		CustomerID:      req.CustomerID,
		SalesID:         req.SalesID,
		ShippingCost:    req.ShippingCost,
		CourierName:     req.CourierName,
		ShippingAddress: req.ShippingAddress,
		Notes:           req.Notes,
	}

	for _, itemReq := range req.Items {
		domainReq.Items = append(domainReq.Items, domain.OrderItemInput{
			ProductID: itemReq.ProductID,
			Qty:       itemReq.Qty,
			Price:     itemReq.Price,
			Details:   itemReq.Details,
		})
	}

	if err := h.orderUsecase.CreateOrder(c.Context(), domainReq); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil membuat pesanan", nil)
}

// @Summary Get Order
// @Description Mengambil detail pesanan (invoice lengkap) berdasarkan ID
// @Tags Orders
// @Produce json
// @Param id path string true "Order ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[dto.OrderResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /orders/{id} [get]
func (h *OrderHandler) GetOrder(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	order, err := h.orderUsecase.GetOrder(c.Context(), id)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengambil detail pesanan", dto.ToOrderResponse(order))
}

// @Summary List Orders
// @Description Mengambil daftar seluruh pesanan dengan pagination
// @Tags Orders
// @Produce json
// @Param page query int false "Nomor Halaman" default(1)
// @Param limit query int false "Batas Data per Halaman" default(10)
// @Success 200 {object} utils.PaginatedResponse[dto.OrderResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Router /orders [get]
func (h *OrderHandler) ListOrders(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	query := domain.PaginationQuery{Page: page, Limit: limit}

	orders, meta, err := h.orderUsecase.ListOrders(c.Context(), query)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccessPaginated(c, "Berhasil mengambil daftar pesanan", dto.ToOrderResponseList(orders), meta)
}

// @Summary Update Order Data
// @Description Memperbarui data ongkir, kurir, alamat, dan catatan (Bukan item pesanan)
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID (UUID)"
// @Param request body dto.OrderUpdateRequest true "Data Update Pesanan"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /orders/{id} [put]
func (h *OrderHandler) UpdateOrder(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.OrderUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	domainReq := domain.OrderUpdateInput{
		ShippingCost:    req.ShippingCost,
		CourierName:     req.CourierName,
		ShippingAddress: req.ShippingAddress,
		Notes:           req.Notes,
	}

	if err := h.orderUsecase.UpdateOrder(c.Context(), id, domainReq); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memperbarui data pesanan", nil)
}

// @Summary Update Order Status
// @Description Memperbarui status pengerjaan pesanan (pending, production, completed, canceled)
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID (UUID)"
// @Param request body dto.OrderStatusUpdateRequest true "Data Update Status"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /orders/{id}/status [patch]
func (h *OrderHandler) UpdateOrderStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.OrderStatusUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}

	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	status := domain.OrderStatus(req.OrderStatus)

	if err := h.orderUsecase.UpdateOrderStatus(c.Context(), id, status); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memperbarui status pesanan", nil)
}

// @Summary Delete Order
// @Description Menghapus pesanan secara permanen
// @Tags Orders
// @Produce json
// @Param id path string true "Order ID (UUID)"
// @Success 200 {object} utils.SuccessResponse[utils.EmptyObj]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /orders/{id} [delete]
func (h *OrderHandler) DeleteOrder(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.orderUsecase.DeleteOrder(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus pesanan", nil)
}
