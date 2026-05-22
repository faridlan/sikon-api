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

func (u *paymentUsecase) ProcessPayment(c context.Context, payment *domain.Payment) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// 1. Validasi nominal uang
	if payment.Amount <= 0 {
		return domain.NewError(domain.ErrBadParamInput, "Nominal pembayaran harus lebih dari 0")
	}

	// 2. Validasi Order
	order, err := u.orderRepo.GetByID(ctx, payment.OrderID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrBadParamInput, "Order tidak ditemukan")
		}
		return err
	}

	// 3. Validasi Bank Account
	_, err = u.bankAccountRepo.GetByID(ctx, payment.BankAccountID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrBadParamInput, "Rekening bank tidak valid")
		}
		return err
	}

	// 4. Kalkulasi Total Kewajiban (Harga Barang + Ongkos Kirim)
	grandTotal := order.TotalAmount + order.ShippingCost

	// 5. Ambil riwayat pembayaran sebelumnya untuk order ini
	existingPayments, err := u.paymentRepo.GetByOrderID(ctx, payment.OrderID)
	if err != nil {
		return err
	}

	// Hitung total uang yang sudah masuk sebelumnya
	var totalPaid float64
	for _, p := range existingPayments {
		totalPaid += p.Amount
	}

	// Cek apakah order ini sebenarnya sudah lunas
	if totalPaid >= grandTotal {
		return domain.NewError(domain.ErrConflict, "Pesanan ini sudah lunas sepenuhnya")
	}

	// 6. Simpan Pembayaran Baru
	if payment.PaymentDate.IsZero() {
		payment.PaymentDate = time.Now()
	}
	if err := u.paymentRepo.Create(ctx, payment); err != nil {
		return err
	}

	// 7. Update Payment Status di Order
	totalPaidSetelahMasuk := totalPaid + payment.Amount
	newPaymentStatus := domain.PaymentStatusPartial // Asumsi awal: baru bayar sebagian (DP)

	if totalPaidSetelahMasuk >= grandTotal {
		newPaymentStatus = domain.PaymentStatusPaid // Jika sudah menutupi total tagihan, set Lunas
	}

	// Panggil repository order untuk update status pembayarannya
	if err := u.orderRepo.UpdateStatus(ctx, order.ID, "", newPaymentStatus); err != nil {
		return err
	}

	return nil
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

func (u *paymentUsecase) ListPayments(c context.Context, query domain.PaginationQuery) ([]domain.Payment, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	offset := query.GetOffset()
	limit := query.Limit

	payments, totalItems, err := u.paymentRepo.Fetch(ctx, limit, offset)
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

func (u *paymentUsecase) UpdatePayment(c context.Context, payment *domain.Payment) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Opsional: Untuk sistem keuangan ERP, biasanya update nominal payment tidak diizinkan.
	// Jika ada kesalahan, praktiknya adalah membatalkan payment tersebut dan membuat yang baru.
	// Namun untuk MVP kita sediakan fungsi updatenya (misal untuk update nomor referensi transfer).

	existingPayment, err := u.paymentRepo.GetByID(ctx, payment.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Data pembayaran tidak ditemukan")
		}
		return err
	}

	if payment.ReferenceNumber != "" {
		existingPayment.ReferenceNumber = payment.ReferenceNumber
	}
	if payment.PaymentType != "" {
		existingPayment.PaymentType = payment.PaymentType
	}

	return u.paymentRepo.Update(ctx, existingPayment)
}

func (u *paymentUsecase) DeletePayment(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Jika payment dihapus, idealnya kita harus menghitung ulang status Order
	// (apakah dari lunas menjadi DP kembali). Namun untuk MVP tahap awal,
	// kita lakukan hard delete saja terlebih dahulu.

	return u.paymentRepo.Delete(ctx, id)
}
