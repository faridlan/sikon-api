package usecase

import (
	"context"
	"errors"
	"fmt"
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

	// 1. Validasi Bisnis (Order exist, Bank Account exist)
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

	// PERBAIKAN 1: Tidak perlu ditambah ShippingCost lagi (karena sudah include di TotalAmount saat order dibuat)
	grandTotal := order.TotalAmount

	// 2. Hitung total uang yang sudah masuk sebelumnya
	existingPayments, err := u.paymentRepo.GetByOrderID(ctx, input.OrderID)
	if err != nil {
		return nil, err
	}

	var totalPaid float64
	for _, p := range existingPayments {
		totalPaid += p.Amount
	}

	// 3. Validasi Lunas di Awal
	if totalPaid >= grandTotal {
		return nil, domain.NewError(domain.ErrConflict, "Pesanan ini sudah lunas sepenuhnya")
	}

	// PERBAIKAN 2: Validasi Cegah Overpayment (Bayar lebih dari sisa tagihan)
	sisaTagihan := grandTotal - totalPaid
	if input.Amount > sisaTagihan {
		return nil, domain.NewError(domain.ErrBadParamInput, "Jumlah bayar melebihi sisa tagihan. Sisa tagihan: Rp "+fmt.Sprintf("%.0f", sisaTagihan))
	}

	// 4. Mapping & Save Payment
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

	// 5. Penentuan Status Pembayaran Baru
	totalPaidSetelahMasuk := totalPaid + payment.Amount
	newPaymentStatus := domain.PaymentStatusPartial

	// Karena kita sudah pakai proteksi overpayment di atas,
	// totalPaidSetelahMasuk pasti mentok maksimal sama dengan grandTotal
	if totalPaidSetelahMasuk >= grandTotal {
		newPaymentStatus = domain.PaymentStatusPaid
	}

	// 6. Update Status Order
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

	// 1. Ambil data payment untuk mendapatkan detail OrderID dan Amount
	payment, err := u.paymentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Data pembayaran tidak ditemukan")
		}
		return err
	}

	// 2. Dapatkan data Order terkait untuk melihat total tagihan (TotalAmount)
	order, err := u.orderRepo.GetByID(ctx, payment.OrderID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrBadParamInput, "Order terkait tidak ditemukan")
		}
		return err
	}

	// 3. Ambil semua pembayaran yang terdaftar untuk order ini saat ini
	existingPayments, err := u.paymentRepo.GetByOrderID(ctx, payment.OrderID)
	if err != nil {
		return err
	}

	// 4. Hitung total bayar saat ini, lalu kurangi dengan nominal payment yang akan dihapus
	var totalPaidBeforeDelete float64
	for _, p := range existingPayments {
		totalPaidBeforeDelete += p.Amount
	}
	totalPaidAfterDelete := totalPaidBeforeDelete - payment.Amount

	// 5. Tentukan status pembayaran yang baru secara otomatis
	var newPaymentStatus domain.PaymentStatus
	if totalPaidAfterDelete <= 0 {
		newPaymentStatus = domain.PaymentStatusUnpaid
	} else if totalPaidAfterDelete < order.TotalAmount {
		newPaymentStatus = domain.PaymentStatusPartial
	} else {
		newPaymentStatus = domain.PaymentStatusPaid
	}

	// 6. Update status pembayaran di tabel order terlebih dahulu
	err = u.orderRepo.UpdateStatus(ctx, order.ID, domain.OrderStatus(""), newPaymentStatus)
	if err != nil {
		return err
	}

	// 7. Setelah status order aman disinkronkan, lakukan penghapusan data payment
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
