package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/google/uuid"
)

type orderUsecase struct {
	orderRepo      domain.OrderRepository
	customerRepo   domain.CustomerRepository
	userRepo       domain.UserRepository // Untuk memvalidasi Sales
	productRepo    domain.ProductRepository
	batchPoRepo    domain.BatchPORepository
	paymentRepo    domain.PaymentRepository
	txManager      domain.TransactionManager
	contextTimeout time.Duration
}

func NewOrderUsecase(or domain.OrderRepository, cr domain.CustomerRepository, ur domain.UserRepository, pr domain.ProductRepository, bpr domain.BatchPORepository, payRepo domain.PaymentRepository, txManager domain.TransactionManager, timeout time.Duration) domain.OrderUsecase {
	return &orderUsecase{
		orderRepo:      or,
		customerRepo:   cr,
		userRepo:       ur,
		productRepo:    pr,
		batchPoRepo:    bpr,
		paymentRepo:    payRepo,
		txManager:      txManager,
		contextTimeout: timeout,
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

	// Gembok Order Susulan: Hanya untuk PO Aktif (atau hak akses Admin di masa depan)
	if batchPO.Status == domain.BatchPOStatusClosed {
		// TODO (AUTH JWT): Nanti setelah fitur JWT terpasang, ubah logika ini untuk mengizinkan Admin
		// if currentUser.Role != domain.RoleAdmin { ... }
		return nil, domain.NewError(domain.ErrForbidden, "PO sudah ditutup. Hanya Admin yang dapat memasukkan order susulan.")
	}

	// ========================================================================
	// INISIALISASI ENTITAS ORDER
	// ========================================================================
	order := &domain.Order{
		BatchPoID:       input.BatchPoID, // 🚨 Menyambungkan pesanan ke Gelombang PO
		BatchPO:         batchPO,
		CustomerID:      input.CustomerID,
		SalesID:         input.SalesID,
		ShippingCost:    input.ShippingCost,
		CourierName:     input.CourierName,
		ShippingAddress: input.ShippingAddress,
		Notes:           input.Notes,
		ValidUntil:      input.ValidUntil,
		TermsConditions: input.TermsConditions,
		OrderStatus:     domain.OrderStatusQuotation, // Default status saat pembuatan order
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

	// Delegasi perhitungan ke Domain
	order.CalculateTotals()

	// Delegasi pembuatan nomor unik ke Domain
	if err := order.GenerateOrderNumber(); err != nil {
		return nil, domain.NewError(domain.ErrInternalServerError, "Gagal membuat nomor pesanan")
	}

	// ========================================================================
	// FASE 2: MENGUBAH DATABASE (Di dalam transaksi)
	// ========================================================================

	err = u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		// PERHATIAN: Gunakan txCtx HANYA untuk operasi tulis ke database

		if err := u.orderRepo.Create(txCtx, order); err != nil {
			return err // Otomatis Rollback
		}

		// (Ruang aman untuk penambahan fitur potong stok bahan baku di masa depan)

		return nil // Otomatis Commit
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

	// 🚨 ATURAN BISNIS BARU (DEFAULT ACTIVE BATCH PO):
	// Jika Frontend TIDAK mengirim batch_po_id (kosong ""),
	// Otomatis cari Batch PO yang sedang AKTIF hari ini.
	if filter.BatchPoID == "" {
		activePO, err := u.batchPoRepo.GetActivePOByDate(ctx, time.Now())
		if err == nil && activePO != nil {
			filter.BatchPoID = activePO.ID
		}
	}

	offset := query.GetOffset()
	limit := query.Limit

	// Teruskan filter ke repository
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

	// Perbarui komponen biaya pengiriman
	existingOrder.ShippingCost = input.ShippingCost
	existingOrder.CourierName = input.CourierName
	existingOrder.ShippingAddress = input.ShippingAddress
	existingOrder.Notes = input.Notes
	existingOrder.ValidUntil = input.ValidUntil
	existingOrder.TermsConditions = input.TermsConditions

	existingOrder.CalculateTotals() // Hitung ulang total setelah update biaya pengiriman

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

		// 🚨 SIMPAN STATUS LAMA SEBELUM SATPAM MENGUBAHNYA
		oldStatus := order.OrderStatus

		// 2. Panggil Satpam (State Machine)
		if err := order.TransitionStatus(status); err != nil {
			return err
		}

		// 🚨 LOGIC RE-ALOKASI PO OTOMATIS
		// Picu re-alokasi saat order di-approve (Quotation -> Pending)
		if oldStatus == domain.OrderStatusQuotation && status == domain.OrderStatusPending {
			now := time.Now()
			order.ApprovedAt = &now

			// Cari PO yang aktif hari ini
			activePO, err := u.batchPoRepo.GetActivePOByDate(txCtx, now)
			if err == nil && activePO != nil {
				order.BatchPoID = activePO.ID
			}
		}

		// 3. Simpan perubahan penuh ke Database
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
	// 1. Ambil data order terbaru
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// 2. Hitung ulang subtotal dari items
	var subtotal float64
	for _, item := range order.Items {
		subtotal += item.Price * float64(item.Qty)
	}
	order.Subtotal = subtotal
	order.CalculateTotals()

	// 3. Re-evaluasi status pembayaran HANYA dari Payment yang VERIFIED
	existingPayments, err := u.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	var totalPaidVerified float64
	for _, p := range existingPayments {
		// 🚨 HANYA HITUNG PAYMENT YANG SUDAH DIVERIFIKASI FINANCE
		if p.Status == domain.PaymentVerificationVerified {
			totalPaidVerified += p.Amount
		}
	}

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

	// 1. Validasi keberadaan Order & Product (Bisa pakai ctx biasa karena hanya membaca)
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

	var updatedOrder *domain.Order // Variabel penampung hasil akhir

	// =========================================================
	// 🚨 MEMULAI TRANSAKSI (MENGGUNAKAN KERTAS BURAM) 🚨
	// =========================================================
	err = u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		// PERHATIAN: Di dalam blok ini, WAJIB menggunakan txCtx, BUKAN ctx!

		newItem := &domain.OrderItem{
			ID:         uuid.New().String(),
			OrderID:    orderID,
			ProductID:  input.ProductID,
			CustomName: input.CustomName,
			Qty:        input.Qty,
			Price:      price,
			Details:    input.Details,
		}

		// Query 1: Simpan item (Staf otomatis membaca txCtx dan memakai Kertas Buram)
		if err := u.orderRepo.CreateItem(txCtx, newItem); err != nil {
			return err // Jika error, otomatis ROLLBACK semua!
		}

		// Query 2: Hitung ulang total dan simpan order (Juga pakai txCtx)
		order, err := u.recalculateOrderTotal(txCtx, orderID)
		if err != nil {
			return err // Jika gagal hitung, item yang tadi tersimpan juga ikut di-ROLLBACK!
		}

		updatedOrder = order
		return nil // Semuanya sukses! Otomatis COMMIT ke Buku Besar!
	})

	// Cek apakah proses transaksi secara keseluruhan gagal
	if err != nil {
		return nil, err
	}

	return updatedOrder, nil
}

func (u *orderUsecase) UpdateOrderItem(c context.Context, orderID, itemID string, input domain.OrderItemInput) (*domain.Order, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// 1. Ambil dan validasi data Item lama
	existingItem, err := u.orderRepo.GetItemByID(ctx, orderID, itemID)
	if err != nil {
		return nil, domain.NewError(domain.ErrNotFound, "Item order tidak ditemukan")
	}

	// 2. Validasi Produk baru (jika ganti produk)
	product, err := u.productRepo.GetByID(ctx, input.ProductID)
	if err != nil {
		return nil, domain.NewError(domain.ErrBadParamInput, "Produk tidak valid")
	}

	// 3. Tentukan harga (update data)
	price := input.Price
	if price <= 0 {
		price = product.BasePrice
	}

	existingItem.ProductID = input.ProductID
	existingItem.CustomName = input.CustomName
	existingItem.Qty = input.Qty
	existingItem.Price = price
	existingItem.Details = input.Details

	var updatedOrder *domain.Order // Variabel penampung hasil akhir

	err = u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		// 4. Simpan perubahan Item
		if err := u.orderRepo.UpdateItem(txCtx, existingItem); err != nil {
			return err
		}

		// 5. Hitung ulang total dan kembalikan order terbaru
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

	var updatedOrder *domain.Order // Variabel penampung hasil akhir

	err := u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Hapus item
		if err := u.orderRepo.DeleteItem(ctx, orderID, itemID); err != nil {
			return err
		}

		// Hitung ulang total setelah terhapus dan kembalikan order terbaru
		order, err := u.recalculateOrderTotal(ctx, orderID)
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
