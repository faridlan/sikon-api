package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type orderUsecase struct {
	orderRepo      domain.OrderRepository
	customerRepo   domain.CustomerRepository
	userRepo       domain.UserRepository // Untuk memvalidasi Sales
	productRepo    domain.ProductRepository
	contextTimeout time.Duration
}

func NewOrderUsecase(or domain.OrderRepository, cr domain.CustomerRepository, ur domain.UserRepository, pr domain.ProductRepository, timeout time.Duration) domain.OrderUsecase {
	return &orderUsecase{
		orderRepo:      or,
		customerRepo:   cr,
		userRepo:       ur,
		productRepo:    pr,
		contextTimeout: timeout,
	}
}

func (u *orderUsecase) CreateOrder(c context.Context, order *domain.Order) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// 1. Validasi Customer
	if _, err := u.customerRepo.GetByID(ctx, order.CustomerID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrBadParamInput, "Customer tidak ditemukan")
		}
		return err
	}

	// 2. Validasi Sales
	if _, err := u.userRepo.GetByID(ctx, order.SalesID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrBadParamInput, "Sales tidak ditemukan")
		}
		return err
	}

	// 3. Validasi Produk & Hitung Total Amount
	var totalAmount float64
	for i, item := range order.Items {
		product, err := u.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NewError(domain.ErrBadParamInput, fmt.Sprintf("Produk dengan ID %s tidak ditemukan", item.ProductID))
			}
			return err
		}

		// Jika harga tidak dikirim dari input, gunakan BasePrice dari tabel produk
		if item.Price <= 0 {
			order.Items[i].Price = product.BasePrice
		}

		// Validasi Qty minimum
		if item.Qty <= 0 {
			return domain.NewError(domain.ErrBadParamInput, "Kuantitas produk minimal 1")
		}

		// Hitung subtotal untuk item ini
		totalAmount += order.Items[i].Price * float64(item.Qty)
	}

	// 4. Set Default Values untuk Order Baru
	order.TotalAmount = totalAmount
	order.OrderStatus = domain.OrderStatusPending
	order.PaymentStatus = domain.PaymentStatusUnpaid

	// Generate Order Number (Contoh sederhana: ORD-YYYYMMDD-XXXX)
	// Untuk production SaaS sebaiknya pakai sequence DB atau generator yang lebih aman
	randomStr := rand.Intn(9999)
	order.OrderNumber = fmt.Sprintf("ORD-%s-%04d", time.Now().Format("20060102"), randomStr)

	// 5. Simpan Order (Ini akan memanggil DB Transaction di Repository yang sudah kita buat)
	return u.orderRepo.Create(ctx, order)
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

func (u *orderUsecase) ListOrders(c context.Context, query domain.PaginationQuery) ([]domain.Order, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	offset := query.GetOffset()
	limit := query.Limit

	orders, totalItems, err := u.orderRepo.Fetch(ctx, limit, offset)
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

func (u *orderUsecase) UpdateOrder(c context.Context, order *domain.Order) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Update order (hanya data master ordernya saja, bukan itemnya)
	existingOrder, err := u.orderRepo.GetByID(ctx, order.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Order tidak ditemukan")
		}
		return err
	}

	// Update field yang diperbolehkan
	existingOrder.ShippingCost = order.ShippingCost
	existingOrder.CourierName = order.CourierName
	existingOrder.ShippingAddress = order.ShippingAddress
	existingOrder.Notes = order.Notes

	return u.orderRepo.Update(ctx, existingOrder)
}

func (u *orderUsecase) DeleteOrder(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.orderRepo.Delete(ctx, id)
}

func (u *orderUsecase) UpdateOrderStatus(c context.Context, id string, status domain.OrderStatus) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Validasi status string
	if status != domain.OrderStatusPending && status != domain.OrderStatusProduction &&
		status != domain.OrderStatusCompleted && status != domain.OrderStatusCanceled {
		return domain.NewError(domain.ErrBadParamInput, "Status order tidak valid")
	}

	return u.orderRepo.UpdateStatus(ctx, id, status, "")
}
