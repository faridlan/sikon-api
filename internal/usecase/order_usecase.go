package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/google/uuid"
)

type orderUsecase struct {
	orderRepo      domain.OrderRepository
	customerRepo   domain.CustomerRepository
	userRepo       domain.UserRepository // Untuk memvalidasi Sales
	productRepo    domain.ProductRepository
	paymentRepo    domain.PaymentRepository
	contextTimeout time.Duration
}

func NewOrderUsecase(or domain.OrderRepository, cr domain.CustomerRepository, ur domain.UserRepository, pr domain.ProductRepository, payRepo domain.PaymentRepository, timeout time.Duration) domain.OrderUsecase {
	return &orderUsecase{
		orderRepo:      or,
		customerRepo:   cr,
		userRepo:       ur,
		productRepo:    pr,
		paymentRepo:    payRepo,
		contextTimeout: timeout,
	}
}

func (u *orderUsecase) CreateOrder(c context.Context, input domain.OrderCreateInput) (*domain.Order, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Validasi Bisnis (Customer & Sales exist)
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

	statusOrder := domain.OrderStatusPending

	if input.OrderStatus == domain.OrderStatusQuotation {
		statusOrder = domain.OrderStatusQuotation
	}

	order := &domain.Order{
		CustomerID:      input.CustomerID,
		SalesID:         input.SalesID,
		ShippingCost:    input.ShippingCost,
		CourierName:     input.CourierName,
		ShippingAddress: input.ShippingAddress,
		Notes:           input.Notes,
		ValidUntil:      input.ValidUntil,
		TermsConditions: input.TermsConditions,
		OrderStatus:     statusOrder,
		PaymentStatus:   domain.PaymentStatusUnpaid,
		DiscountAmount:  0,
		TaxPpn:          0,
		TaxPph:          0,
	}

	var subtotal float64 // Variabel penampung total harga barang murni
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

		// Hitung subtotal akumulatif barang
		subtotal += price * float64(itemInput.Qty)

		order.Items = append(order.Items, domain.OrderItem{
			ProductID: itemInput.ProductID,
			Qty:       itemInput.Qty,
			Price:     price,
			Details:   itemInput.Details,
		})
	}

	// Masukkan nilai subtotal murni ke entity order
	order.Subtotal = subtotal

	// Rumus Grand Total Masa Depan (Saat ini discount, ppn, pph masih bernilai 0)
	order.TotalAmount = (order.Subtotal - order.DiscountAmount) + order.TaxPpn - order.TaxPph + order.ShippingCost

	randomStr := rand.Intn(9999)
	order.OrderNumber = fmt.Sprintf("ORD-%s-%04d", time.Now().Format("20060102"), randomStr)

	err := u.orderRepo.Create(ctx, order)
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

	// Hitung ulang Grand Total berdasarkan kondisi data terbaru
	existingOrder.TotalAmount = (existingOrder.Subtotal - existingOrder.DiscountAmount) + existingOrder.TaxPpn - existingOrder.TaxPph + existingOrder.ShippingCost

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

	validStatuses := map[domain.OrderStatus]bool{
		domain.OrderStatusPending:    true,
		domain.OrderStatusProduction: true,
		domain.OrderStatusCompleted:  true,
		domain.OrderStatusCanceled:   true,
		domain.OrderStatusQuotation:  true,
	}

	if !validStatuses[status] {
		return domain.NewError(domain.ErrBadParamInput, "Status order tidak valid")
	}

	return u.orderRepo.UpdateStatus(ctx, id, status, "")
}

func (u *orderUsecase) UpdatePaymentStatus(c context.Context, id string, status domain.PaymentStatus) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Validasi input status pembayaran
	validStatuses := map[domain.PaymentStatus]bool{
		domain.PaymentStatusUnpaid:  true,
		domain.PaymentStatusPartial: true,
		domain.PaymentStatusPaid:    true,
	}

	if !validStatuses[status] {
		return domain.NewError(domain.ErrBadParamInput, "Status pembayaran tidak valid")
	}

	// Perhatikan: orderStatus dikosongkan (""), hanya paymentStatus yang diisi
	// karena fungsi repo Anda akan mendeteksinya otomatis
	return u.orderRepo.UpdateStatus(ctx, id, "", status)
}

// --- TAMBAHAN BARU: Private Helper untuk Hitung Ulang Total ---
func (u *orderUsecase) recalculateOrderTotal(ctx context.Context, orderID string) (*domain.Order, error) {
	// 1. Ambil data order terbaru beserta seluruh items-nya
	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// 2. Hitung ulang subtotal murni dari list order.Items
	var subtotal float64
	for _, item := range order.Items {
		subtotal += item.Price * float64(item.Qty)
	}

	// 3. Masukkan ke order & hitung ulang Grand Total
	order.Subtotal = subtotal
	order.TotalAmount = (order.Subtotal - order.DiscountAmount) + order.TaxPpn - order.TaxPph + order.ShippingCost

	// =========================================================
	// 🚨 TAMBAHAN BARU: RE-EVALUASI STATUS PEMBAYARAN 🚨
	// =========================================================

	// Ambil semua histori pembayaran untuk order ini
	existingPayments, err := u.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Jumlahkan total yang sudah dibayar (kas nyata yang masuk)
	var totalPaid float64
	for _, p := range existingPayments {
		totalPaid += p.Amount
	}

	// Tentukan status pembayaran baru berdasarkan selisih
	if totalPaid <= 0 {
		order.PaymentStatus = domain.PaymentStatusUnpaid
	} else if totalPaid < order.TotalAmount {
		order.PaymentStatus = domain.PaymentStatusPartial
	} else {
		order.PaymentStatus = domain.PaymentStatusPaid
	}
	// =========================================================

	// 4. Update Header Order-nya saja ke Database (Subtotal, TotalAmount, & PaymentStatus ikut terupdate)
	err = u.orderRepo.Update(ctx, order)
	if err != nil {
		return nil, err
	}

	return order, nil
}

// --- TAMBAHAN BARU: Usecase Manipulasi Item ---

func (u *orderUsecase) AddOrderItem(c context.Context, orderID string, input domain.OrderItemInput) (*domain.Order, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// 1. Validasi keberadaan Order & Product
	if _, err := u.orderRepo.GetByID(ctx, orderID); err != nil {
		return nil, domain.NewError(domain.ErrNotFound, "Order tidak ditemukan")
	}
	product, err := u.productRepo.GetByID(ctx, input.ProductID)
	if err != nil {
		return nil, domain.NewError(domain.ErrBadParamInput, "Produk tidak valid")
	}

	// 2. Tentukan harga
	price := input.Price
	if price <= 0 {
		price = product.BasePrice
	}

	// 3. Buat Item Baru di Database
	newItem := &domain.OrderItem{
		ID:        uuid.New().String(),
		OrderID:   orderID,
		ProductID: input.ProductID,
		Qty:       input.Qty,
		Price:     price,
		Details:   input.Details,
	}

	if err := u.orderRepo.CreateItem(ctx, newItem); err != nil {
		return nil, err
	}

	// 4. Hitung ulang total dan kembalikan order terbaru
	return u.recalculateOrderTotal(ctx, orderID)
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
	existingItem.Qty = input.Qty
	existingItem.Price = price
	existingItem.Details = input.Details

	// 4. Simpan perubahan Item
	if err := u.orderRepo.UpdateItem(ctx, existingItem); err != nil {
		return nil, err
	}

	// 5. Hitung ulang total dan kembalikan order terbaru
	return u.recalculateOrderTotal(ctx, orderID)
}

func (u *orderUsecase) DeleteOrderItem(c context.Context, orderID, itemID string) (*domain.Order, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Hapus item
	if err := u.orderRepo.DeleteItem(ctx, orderID, itemID); err != nil {
		return nil, err
	}

	// Hitung ulang total setelah terhapus dan kembalikan order terbaru
	return u.recalculateOrderTotal(ctx, orderID)
}
