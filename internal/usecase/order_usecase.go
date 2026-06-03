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
	}

	var totalAmount float64
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

		totalAmount += price * float64(itemInput.Qty)

		order.Items = append(order.Items, domain.OrderItem{
			ProductID: itemInput.ProductID,
			Qty:       itemInput.Qty,
			Price:     price,
			Details:   itemInput.Details,
		})
	}

	order.TotalAmount = totalAmount
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

	existingOrder.ShippingCost = input.ShippingCost
	existingOrder.CourierName = input.CourierName
	existingOrder.ShippingAddress = input.ShippingAddress
	existingOrder.Notes = input.Notes
	existingOrder.ValidUntil = input.ValidUntil
	existingOrder.TermsConditions = input.TermsConditions

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

	// Validasi status string
	if status != domain.OrderStatusPending && status != domain.OrderStatusProduction &&
		status != domain.OrderStatusCompleted && status != domain.OrderStatusCanceled {
		return domain.NewError(domain.ErrBadParamInput, "Status order tidak valid")
	}

	return u.orderRepo.UpdateStatus(ctx, id, status, "")
}
