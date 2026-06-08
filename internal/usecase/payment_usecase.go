package usecase

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type paymentUsecase struct {
	paymentRepo     domain.PaymentRepository
	orderRepo       domain.OrderRepository // Butuh ini untuk mengecek & update status pesanan
	bankAccountRepo domain.BankAccountRepository
	contextTimeout  time.Duration
}

func NewPaymentUsecase(pr domain.PaymentRepository, or domain.OrderRepository, br domain.BankAccountRepository, timeout time.Duration) domain.PaymentUsecase {
	return &paymentUsecase{
		paymentRepo:     pr,
		orderRepo:       or,
		bankAccountRepo: br,
		contextTimeout:  timeout,
	}
}

func (u *paymentUsecase) ProcessPayment(c context.Context, input domain.PaymentCreateInput) (*domain.Payment, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Validasi Bisnis (Order exist, Bank Account exist)
	order, err := u.orderRepo.GetByID(ctx, input.OrderID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrBadParamInput, "Order tidak ditemukan")
		}
		return nil, err
	}

	if _, err = u.bankAccountRepo.GetByID(ctx, input.BankAccountID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrBadParamInput, "Rekening bank tidak valid")
		}
		return nil, err
	}

	grandTotal := order.TotalAmount + order.ShippingCost

	existingPayments, err := u.paymentRepo.GetByOrderID(ctx, input.OrderID)
	if err != nil {
		return nil, err
	}

	var totalPaid float64
	for _, p := range existingPayments {
		totalPaid += p.Amount
	}

	if totalPaid >= grandTotal {
		return nil, domain.NewError(domain.ErrConflict, "Pesanan ini sudah lunas sepenuhnya")
	}

	payment := &domain.Payment{
		OrderID:         input.OrderID,
		BankAccountID:   input.BankAccountID,
		Amount:          input.Amount,
		PaymentDate:     input.PaymentDate,
		ReferenceNumber: input.ReferenceNumber,
		PaymentType:     input.PaymentType,
	}

	if payment.PaymentDate.IsZero() {
		payment.PaymentDate = time.Now()
	}

	if err := u.paymentRepo.Create(ctx, payment); err != nil {
		return nil, err
	}

	totalPaidSetelahMasuk := totalPaid + payment.Amount
	newPaymentStatus := domain.PaymentStatusPartial

	if totalPaidSetelahMasuk >= grandTotal {
		newPaymentStatus = domain.PaymentStatusPaid
	}

	if err := u.orderRepo.UpdateStatus(ctx, order.ID, "", newPaymentStatus); err != nil {
		return nil, err
	}

	return payment, nil
}

func (u *paymentUsecase) GetPayment(c context.Context, id string) (*domain.Payment, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	payment, err := u.paymentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Data pembayaran tidak ditemukan")
		}
		return nil, err
	}

	return payment, nil
}

func (u *paymentUsecase) ListPayments(c context.Context, query domain.PaginationQuery, filter domain.PaymentFilter) ([]domain.Payment, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	offset := query.GetOffset()
	limit := query.Limit

	// Tambahkan filter ke parameter Fetch
	payments, totalItems, err := u.paymentRepo.Fetch(ctx, limit, offset, filter)
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

	return payments, meta, nil
}

func (u *paymentUsecase) UpdatePayment(c context.Context, id string, input domain.PaymentUpdateInput) (*domain.Payment, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	existingPayment, err := u.paymentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Data pembayaran tidak ditemukan")
		}
		return nil, err
	}

	if input.ReferenceNumber != "" {
		existingPayment.ReferenceNumber = input.ReferenceNumber
	}
	if input.PaymentType != "" {
		existingPayment.PaymentType = input.PaymentType
	}

	err = u.paymentRepo.Update(ctx, existingPayment)
	if err != nil {
		return nil, err
	}

	return existingPayment, nil

}

func (u *paymentUsecase) DeletePayment(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	_, err := u.paymentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Data pembayaran tidak ditemukan")
		}
		return err
	}

	// Jika payment dihapus, idealnya kita harus menghitung ulang status Order
	// (apakah dari lunas menjadi DP kembali). Namun untuk MVP tahap awal,
	// kita lakukan hard delete saja terlebih dahulu.

	return u.paymentRepo.Delete(ctx, id)
}

func (u *paymentUsecase) GetPaymentsByOrderID(c context.Context, orderID string) ([]domain.Payment, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	order, err := u.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrNotFound, "Order tidak ditemukan")
		}
		return nil, err
	}

	return u.paymentRepo.GetByOrderID(ctx, order.ID)
}
