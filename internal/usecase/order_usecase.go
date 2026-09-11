package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/google/uuid"
)

type orderUsecase struct {
	orderRepo              domain.OrderRepository
	customerRepo           domain.CustomerRepository
	userRepo               domain.UserRepository
	productRepo            domain.ProductRepository
	batchPoRepo            domain.BatchPORepository
	paymentRepo            domain.PaymentRepository
	productMaterialUsecase domain.ProductMaterialUsecase // BARU
	workLogRepo            domain.WorkLogRepository      // BARU
	txManager              domain.TransactionManager
	contextTimeout         time.Duration
}

func NewOrderUsecase(
	or domain.OrderRepository,
	cr domain.CustomerRepository,
	ur domain.UserRepository,
	pr domain.ProductRepository,
	bpr domain.BatchPORepository,
	payRepo domain.PaymentRepository,
	txManager domain.TransactionManager,
	pmUsecase domain.ProductMaterialUsecase, // BARU
	workLogRepo domain.WorkLogRepository, // BARU
	timeout time.Duration,
) domain.OrderUsecase {
	return &orderUsecase{
		orderRepo:              or,
		customerRepo:           cr,
		userRepo:               ur,
		productRepo:            pr,
		batchPoRepo:            bpr,
		paymentRepo:            payRepo,
		productMaterialUsecase: pmUsecase,
		workLogRepo:            workLogRepo,
		txManager:              txManager,
		contextTimeout:         timeout,
	}
}

func (u *orderUsecase) CreateOrder(c context.Context, input domain.OrderCreateInput) (*domain.Order, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// ========================================================================
	// FASE 1: MEMBACA DATA & PERSIAPAN MEMORY (Di luar transaksi)
	// ========================================================================

	// Validasi Customer & Sales
	if _, err := u.customerRepo.GetByID(ctx, input.CustomerID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrBadParamInput, "Customer tidak ditemukan")
		}
		return nil, err
	}
	if _, err := u.userRepo.GetByID(ctx, input.SalesID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrBadParamInput, "Sales tidak ditemukan")
		}
		return nil, err
	}

	// ========================================================================
	// VALIDASI BATCH PO
	// ========================================================================
	if input.BatchPoID == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Batch PO wajib dipilih")
	}

	batchPO, err := u.batchPoRepo.GetByID(ctx, input.BatchPoID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrBadParamInput, "Batch PO tidak ditemukan")
		}
		return nil, err
	}

	if batchPO.Status == domain.BatchPOStatusClosed {
		return nil, domain.NewError(domain.ErrForbidden, "PO sudah ditutup. Hanya Admin yang dapat memasukkan order susulan.")
	}

	// ========================================================================
	// INISIALISASI ENTITAS ORDER
	// ========================================================================
	order := &domain.Order{
		BatchPoID:       input.BatchPoID,
		BatchPO:         batchPO,
		CustomerID:      input.CustomerID,
		SalesID:         input.SalesID,
		IsTaxable:       input.IsTaxable,    // 👈 Parameter Pajak Instansi
		TaxPpnRate:      input.TaxPpnRate,   // 👈 Rate PPN (default 12%)
		TaxPph22Rate:    input.TaxPph22Rate, // 👈 Rate PPh 22 (default 1.5%)
		ShippingCost:    input.ShippingCost,
		CourierName:     input.CourierName,
		ShippingAddress: input.ShippingAddress,
		Notes:           input.Notes,
		ValidUntil:      input.ValidUntil,
		TermsConditions: input.TermsConditions,
		OrderStatus:     domain.OrderStatusQuotation,
		PaymentStatus:   domain.PaymentStatusUnpaid,
	}

	// Validasi Produk dan Penyusunan Items
	for _, itemInput := range input.Items {
		product, err := u.productRepo.GetByID(ctx, itemInput.ProductID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, domain.NewError(domain.ErrBadParamInput, fmt.Sprintf("Produk dengan ID %s tidak ditemukan", itemInput.ProductID))
			}
			return nil, err
		}

		price := itemInput.Price
		if price <= 0 {
			price = product.BasePrice
		}

		order.Items = append(order.Items, domain.OrderItem{
			ProductID:  itemInput.ProductID,
			CustomName: itemInput.CustomName,
			Qty:        itemInput.Qty,
			Price:      price,
			Details:    itemInput.Details,
		})
	}

	// Delegasi perhitungan matematika & pajak ke Domain
	order.CalculateTotals()

	// Delegasi pembuatan nomor unik ke Domain
	if err := order.GenerateOrderNumber(); err != nil {
		return nil, domain.NewError(domain.ErrInternalServerError, "Gagal membuat nomor pesanan")
	}

	// ========================================================================
	// FASE 2: MENGUBAH DATABASE (Di dalam transaksi)
	// ========================================================================

	err = u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		if err := u.orderRepo.Create(txCtx, order); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return order, nil
}

func (u *orderUsecase) GetOrder(c context.Context, id string) (*domain.Order, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	order, err := u.orderRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Order tidak ditemukan")
		}
		return nil, err
	}

	return order, nil
}

func (u *orderUsecase) ListOrders(c context.Context, filter domain.OrderFilter, query domain.PaginationQuery) ([]domain.Order, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if filter.BatchPoID == "" {
		activePO, err := u.batchPoRepo.GetActivePOByDate(ctx, time.Now())
		if err == nil && activePO != nil {
			filter.BatchPoID = activePO.ID
		}
	}

	offset := query.GetOffset()
	limit := query.Limit

	orders, totalItems, err := u.orderRepo.Fetch(ctx, filter, limit, offset)
	if err != nil {
		return nil, domain.PaginationMeta{}, err
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))

	meta := domain.PaginationMeta{
		CurrentPage: query.Page,
		Limit:       limit,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
	}

	return orders, meta, nil
}

func (u *orderUsecase) UpdateOrder(c context.Context, id string, input domain.OrderUpdateInput) (*domain.Order, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingOrder, err := u.orderRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Order tidak ditemukan")
		}
		return nil, err
	}

	// Perbarui komponen pajak & biaya pengiriman
	existingOrder.IsTaxable = input.IsTaxable       // 👈
	existingOrder.TaxPpnRate = input.TaxPpnRate     // 👈
	existingOrder.TaxPph22Rate = input.TaxPph22Rate // 👈
	existingOrder.ShippingCost = input.ShippingCost
	existingOrder.CourierName = input.CourierName
	existingOrder.ShippingAddress = input.ShippingAddress
	existingOrder.Notes = input.Notes
	existingOrder.ValidUntil = input.ValidUntil
	existingOrder.TermsConditions = input.TermsConditions

	existingOrder.CalculateTotals() // Hitung ulang total & kalkulasi pajak pengadaan

	err = u.orderRepo.Update(ctx, existingOrder)
	if err != nil {
		return nil, err
	}
	return existingOrder, nil
}

func (u *orderUsecase) DeleteOrder(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	_, err := u.orderRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Order tidak ditemukan")
		}
		return err
	}

	return u.orderRepo.Delete(ctx, id)
}

func (u *orderUsecase) UpdateOrderStatus(c context.Context, id string, status domain.OrderStatus) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if !status.IsValid() {
		return domain.NewError(domain.ErrBadParamInput, "Status order tidak valid")
	}

	return u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		order, err := u.orderRepo.GetByIDForUpdate(txCtx, id)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NewError(domain.ErrNotFound, "Order tidak ditemukan")
			}
			return err
		}

		oldStatus := order.OrderStatus

		if err := order.TransitionStatus(status); err != nil {
			return err
		}

		if oldStatus == domain.OrderStatusQuotation && status == domain.OrderStatusPending {
			now := time.Now()
			order.ApprovedAt = &now

			// Cari PO yang aktif hari ini
			activePO, err := u.batchPoRepo.GetActivePOByDate(txCtx, now)
			if err == nil && activePO != nil {
				order.BatchPoID = activePO.ID
			}

			// 🚨 BEKUKAN HPP MATERIAL DI SINI
			// GetByIDForUpdate() sengaja tidak preload Items (biar locking ringan),
			// jadi ambil Items-nya lewat query terpisah di dalam transaksi yang sama.
			fullOrder, err := u.orderRepo.GetByID(txCtx, id)
			if err != nil {
				return err
			}

			slog.Debug("[HPP-DEBUG] mulai hitung HPP material", "order_id", id, "jumlah_items", len(fullOrder.Items))
			var materialCost float64
			for _, item := range fullOrder.Items {
				slog.Debug("[HPP-DEBUG] proses item", "product_id", item.ProductID, "qty", item.Qty)
				cost, err := u.productMaterialUsecase.CalculateMaterialCost(txCtx, item.ProductID, item.Qty)
				if err != nil {
					slog.Error("[HPP-DEBUG] CalculateMaterialCost error", "error", err)
					return err
				}
				slog.Debug("[HPP-DEBUG] hasil item", "product_id", item.ProductID, "cost", cost)
				materialCost += cost
			}
			slog.Debug("[HPP-DEBUG] TOTAL", "materialCost", materialCost)
			order.HPPMaterialCost = materialCost
			order.HPPCalculatedAt = &now
		}

		if err := u.orderRepo.Update(txCtx, order); err != nil {
			return err
		}

		return nil
	})
}

func (u *orderUsecase) UpdatePaymentStatus(c context.Context, id string, status domain.PaymentStatus) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if !status.IsValid() {
		return domain.NewError(domain.ErrBadParamInput, "Status pembayaran tidak valid")
	}

	return u.orderRepo.UpdateStatus(ctx, id, "", status)
}

// --- Private Helper untuk Hitung Ulang Total ---
func (u *orderUsecase) recalculateOrderTotal(ctx context.Context, orderID string) (*domain.Order, error) {
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	var subtotal float64
	for _, item := range order.Items {
		subtotal += item.Price * float64(item.Qty)
	}
	order.Subtotal = subtotal
	order.CalculateTotals()

	existingPayments, err := u.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	var totalPaidVerified float64
	for _, p := range existingPayments {
		if p.Status == domain.PaymentVerificationVerified {
			totalPaidVerified += p.Amount
		}
	}

	// Catatan: Bandingkan pembayaran verified dengan TotalAmount / PaguAmount
	if totalPaidVerified <= 0 {
		order.PaymentStatus = domain.PaymentStatusUnpaid
	} else if totalPaidVerified < order.TotalAmount {
		order.PaymentStatus = domain.PaymentStatusPartial
	} else {
		order.PaymentStatus = domain.PaymentStatusPaid
	}

	err = u.orderRepo.Update(ctx, order)
	if err != nil {
		return nil, err
	}

	return order, nil
}

// --- Usecase Manipulasi Item ---

func (u *orderUsecase) AddOrderItem(c context.Context, orderID string, input domain.OrderItemInput) (*domain.Order, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if _, err := u.orderRepo.GetByID(ctx, orderID); err != nil {
		return nil, domain.NewError(domain.ErrNotFound, "Order tidak ditemukan")
	}
	product, err := u.productRepo.GetByID(ctx, input.ProductID)
	if err != nil {
		return nil, domain.NewError(domain.ErrBadParamInput, "Produk tidak valid")
	}

	price := input.Price
	if price <= 0 {
		price = product.BasePrice
	}

	var updatedOrder *domain.Order

	err = u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		newItem := &domain.OrderItem{
			ID:         uuid.New().String(),
			OrderID:    orderID,
			ProductID:  input.ProductID,
			CustomName: input.CustomName,
			Qty:        input.Qty,
			Price:      price,
			Details:    input.Details,
		}

		if err := u.orderRepo.CreateItem(txCtx, newItem); err != nil {
			return err
		}

		order, err := u.recalculateOrderTotal(txCtx, orderID)
		if err != nil {
			return err
		}

		updatedOrder = order
		return nil
	})

	if err != nil {
		return nil, err
	}

	return updatedOrder, nil
}

func (u *orderUsecase) UpdateOrderItem(c context.Context, orderID, itemID string, input domain.OrderItemInput) (*domain.Order, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingItem, err := u.orderRepo.GetItemByID(ctx, orderID, itemID)
	if err != nil {
		return nil, domain.NewError(domain.ErrNotFound, "Item order tidak ditemukan")
	}

	product, err := u.productRepo.GetByID(ctx, input.ProductID)
	if err != nil {
		return nil, domain.NewError(domain.ErrBadParamInput, "Produk tidak valid")
	}

	price := input.Price
	if price <= 0 {
		price = product.BasePrice
	}

	existingItem.ProductID = input.ProductID
	existingItem.CustomName = input.CustomName
	existingItem.Qty = input.Qty
	existingItem.Price = price
	existingItem.Details = input.Details

	var updatedOrder *domain.Order

	err = u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		if err := u.orderRepo.UpdateItem(txCtx, existingItem); err != nil {
			return err
		}

		order, err := u.recalculateOrderTotal(txCtx, orderID)
		if err != nil {
			return err
		}
		updatedOrder = order
		return nil
	})

	if err != nil {
		return nil, err
	}

	return updatedOrder, nil
}

func (u *orderUsecase) DeleteOrderItem(c context.Context, orderID, itemID string) (*domain.Order, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	var updatedOrder *domain.Order

	err := u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		if err := u.orderRepo.DeleteItem(txCtx, orderID, itemID); err != nil {
			return err
		}

		order, err := u.recalculateOrderTotal(txCtx, orderID)
		if err != nil {
			return err
		}

		updatedOrder = order
		return nil
	})

	if err != nil {
		return nil, err
	}

	return updatedOrder, nil
}

// GetOrderHPP mengembalikan breakdown HPP: Material (snapshot beku dari saat approve)
// digabung dengan Tenaga Kerja (live, dihitung dari Work Log aktual saat ini).
func (u *orderUsecase) GetOrderHPP(c context.Context, id string) (*domain.OrderHPPBreakdown, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	order, err := u.orderRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Order tidak ditemukan")
		}
		return nil, err
	}

	laborCost, err := u.workLogRepo.GetTotalCostByOrder(ctx, id)
	if err != nil {
		return nil, err
	}

	return &domain.OrderHPPBreakdown{
		OrderID:              order.ID,
		MaterialCost:         order.HPPMaterialCost,
		MaterialCalculatedAt: order.HPPCalculatedAt,
		LaborCost:            laborCost,
		TotalCost:            order.HPPMaterialCost + laborCost,
	}, nil
}
