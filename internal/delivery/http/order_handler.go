package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
)

type OrderHandler interface {
	CreateOrder(c *fiber.Ctx) error
	GetOrder(c *fiber.Ctx) error
	ListOrders(c *fiber.Ctx) error
	UpdateOrder(c *fiber.Ctx) error
	UpdateOrderStatus(c *fiber.Ctx) error
	DeleteOrder(c *fiber.Ctx) error
	UpdatePaymentStatus(c *fiber.Ctx) error

	AddOrderItem(c *fiber.Ctx) error
	UpdateOrderItem(c *fiber.Ctx) error
	DeleteOrderItem(c *fiber.Ctx) error
}

type orderHandler struct {
	orderUsecase domain.OrderUsecase
}

func NewOrderHandler(ou domain.OrderUsecase) OrderHandler {
	return &orderHandler{
		orderUsecase: ou,
	}
}

// @Summary Create Order
// @Description Membuat pesanan (order) baru beserta detail produknya
// @Tags Orders
// @Accept json
// @Produce json
// @Param request body dto.OrderCreateRequest true "Data Pesanan Baru"
// @Success 201 {object} utils.SuccessResponse[dto.OrderResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /orders [post]
func (h *orderHandler) CreateOrder(c *fiber.Ctx) error {
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
		OrderStatus:     domain.OrderStatus(req.OrderStatus),
		ValidUntil:      req.ValidUntil,
		TermsConditions: req.TermsConditions,
	}

	for _, itemReq := range req.Items {
		domainReq.Items = append(domainReq.Items, domain.OrderItemInput{
			ProductID: itemReq.ProductID,
			Qty:       itemReq.Qty,
			Price:     itemReq.Price,
			Details:   itemReq.Details,
		})
	}

	order, err := h.orderUsecase.CreateOrder(c.Context(), domainReq)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil membuat pesanan", dto.ToOrderResponse(order))
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
func (h *orderHandler) GetOrder(c *fiber.Ctx) error {
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
// @Description Mengambil daftar seluruh pesanan dengan pagination dan filter
// @Tags Orders
// @Produce json
// @Param page query int false "Nomor Halaman" default(1)
// @Param limit query int false "Batas Data per Halaman" default(10)
// @Param search query string false "Cari berdasarkan Nomor Order"
// @Param customer_id query string false "Filter berdasarkan ID Customer"
// @Param sales_id query string false "Filter berdasarkan ID Sales"
// @Param order_status query string false "Filter Status Order (quotation, pending, production, completed, canceled)"
// @Param payment_status query string false "Filter Status Pembayaran (unpaid, partial, paid)"
// @Param start_date query string false "Tanggal Mulai (YYYY-MM-DD)"
// @Param end_date query string false "Tanggal Selesai (YYYY-MM-DD)"
// @Success 200 {object} utils.PaginatedResponse[dto.OrderResponse]
// @Failure 500 {object} utils.ErrorResponse
// @Router /orders [get]
func (h *orderHandler) ListOrders(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	// Tangkap semua filter dari URL
	filter := domain.OrderFilter{
		Search:        c.Query("search"),
		CustomerID:    c.Query("customer_id"),
		SalesID:       c.Query("sales_id"),
		OrderStatus:   c.Query("order_status"),
		PaymentStatus: c.Query("payment_status"),
		StartDate:     c.Query("start_date"),
		EndDate:       c.Query("end_date"),
	}

	query := domain.PaginationQuery{Page: page, Limit: limit}

	// Kirim filter ke usecase
	orders, meta, err := h.orderUsecase.ListOrders(c.Context(), filter, query)
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
// @Success 200 {object} utils.SuccessResponse[dto.OrderResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /orders/{id} [put]
func (h *orderHandler) UpdateOrder(c *fiber.Ctx) error {
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
		ValidUntil:      req.ValidUntil,
		TermsConditions: req.TermsConditions,
	}

	order, err := h.orderUsecase.UpdateOrder(c.Context(), id, domainReq)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memperbarui data pesanan", dto.ToOrderResponse(order))
}

// @Summary Update Order Status
// @Description Memperbarui status pengerjaan pesanan (pending, production, completed, canceled)
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID (UUID)"
// @Param request body dto.OrderStatusUpdateRequest true "Data Update Status"
// @Success 200 {object} utils.SuccessResponse[dto.OrderResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /orders/{id}/status [patch]
func (h *orderHandler) UpdateOrderStatus(c *fiber.Ctx) error {
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
func (h *orderHandler) DeleteOrder(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if err := h.orderUsecase.DeleteOrder(c.Context(), id); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus pesanan", nil)
}

// @Summary Update Payment Status
// @Description Memperbarui status pembayaran pesanan (unpaid, partial, paid)
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID (UUID)"
// @Param request body dto.PaymentStatusUpdateRequest true "Data Update Status Pembayaran"
// @Success 200 {object} utils.SuccessResponse[any]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /orders/{id}/payment-status [patch]
func (h *orderHandler) UpdatePaymentStatus(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := utils.ValidateUUID(id, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.PaymentStatusUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}

	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	status := domain.PaymentStatus(req.PaymentStatus)

	if err := h.orderUsecase.UpdatePaymentStatus(c.Context(), id, status); err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil memperbarui status pembayaran", nil)
}

// @Summary Add Item to Order
// @Description Menambahkan item baru ke dalam pesanan yang sudah ada
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param request body dto.OrderItemRequest true "Data Item Baru"
// @Success 201 {object} utils.SuccessResponse[dto.OrderResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /orders/{id}/items [post]
func (h *orderHandler) AddOrderItem(c *fiber.Ctx) error {
	orderID := c.Params("id")
	if err := utils.ValidateUUID(orderID, "id"); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	var req dto.OrderItemRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	input := domain.OrderItemInput{
		ProductID: req.ProductID,
		Qty:       req.Qty,
		Price:     req.Price,
		Details:   req.Details,
	}

	order, err := h.orderUsecase.AddOrderItem(c.Context(), orderID, input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusCreated, "Berhasil menambah item pesanan", dto.ToOrderResponse(order))
}

// @Summary Update Order Item
// @Description Memperbarui informasi item dalam pesanan yang sudah ada
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param itemId path string true "Item ID"
// @Param request body dto.OrderItemRequest true "Data Update Item"
// @Success 200 {object} utils.SuccessResponse[dto.OrderResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /orders/{id}/items/{itemId} [put]
func (h *orderHandler) UpdateOrderItem(c *fiber.Ctx) error {
	orderID := c.Params("id")
	itemID := c.Params("itemId")

	var req dto.OrderItemRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Gagal memparsing request body")
	}
	if err := utils.ValidateStruct(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	input := domain.OrderItemInput{
		ProductID: req.ProductID,
		Qty:       req.Qty,
		Price:     req.Price,
		Details:   req.Details,
	}

	order, err := h.orderUsecase.UpdateOrderItem(c.Context(), orderID, itemID, input)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil mengubah item pesanan", dto.ToOrderResponse(order))
}

// @Summary Delete Order Item
// @Description Menghapus item dari pesanan yang sudah ada
// @Tags Orders
// @Produce json
// @Param id path string true "Order ID"
// @Param itemId path string true "Item ID"
// @Success 200 {object} utils.SuccessResponse[dto.OrderResponse]
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /orders/{id}/items/{itemId} [delete]
func (h *orderHandler) DeleteOrderItem(c *fiber.Ctx) error {
	orderID := c.Params("id")
	itemID := c.Params("itemId")

	order, err := h.orderUsecase.DeleteOrderItem(c.Context(), orderID, itemID)
	if err != nil {
		return utils.HandleDomainError(c, err)
	}

	return utils.SendSuccess(c, fiber.StatusOK, "Berhasil menghapus item pesanan", dto.ToOrderResponse(order))
}
