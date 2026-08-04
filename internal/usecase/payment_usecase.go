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
	txManager       domain.TransactionManager
	batchPoRepo     domain.BatchPORepository
	contextTimeout  time.Duration
}

func NewPaymentUsecase(pr domain.PaymentRepository, or domain.OrderRepository, br domain.BankAccountRepository, tx domain.TransactionManager, bpr domain.BatchPORepository, timeout time.Duration) domain.PaymentUsecase {
	return &paymentUsecase{
		paymentRepo:     pr,
		orderRepo:       or,
		bankAccountRepo: br,
		txManager:       tx,
		batchPoRepo:     bpr,
		contextTimeout:  timeout,
	}
}

func (u *paymentUsecase) ProcessPayment(c context.Context, input domain.PaymentCreateInput) (*domain.Payment, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	var createdPayment *domain.Payment

	err := u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		// 🚨 PASTIKAN SELALU MENGGUNAKAN txCtx DI SINI:

		// 1. Ambil order dengan lock
		order, err := u.orderRepo.GetByIDForUpdate(txCtx, input.OrderID)
		if err != nil {
			return err
		}

		// 2. Ambil payments yang terverifikasi
		existingPayments, err := u.paymentRepo.GetByOrderID(txCtx, input.OrderID)
		if err != nil {
			return err
		}

		var totalPaidVerified float64
		for _, p := range existingPayments {
			if p.Status == domain.PaymentVerificationVerified {
				totalPaidVerified += p.Amount
			}
		}

		sisaTagihan := order.TotalAmount - totalPaidVerified
		if sisaTagihan <= 0 {
			return domain.NewError(domain.ErrConflict, "Pesanan ini sudah lunas sepenuhnya")
		}

		if input.Amount > sisaTagihan {
			return domain.NewError(domain.ErrBadParamInput, fmt.Sprintf("Nominal pembayaran (Rp %.0f) melebihi sisa tagihan (Rp %.0f)", input.Amount, sisaTagihan))
		}

		// 3. Simpan payment baru
		payment := &domain.Payment{
			OrderID:         input.OrderID,
			BankAccountID:   input.BankAccountID,
			Amount:          input.Amount,
			PaymentDate:     input.PaymentDate,
			ReferenceNumber: input.ReferenceNumber,
			PaymentType:     input.PaymentType,
			Status:          domain.PaymentVerificationPending,
		}

		// 🚨 GUNAKAN txCtx
		if err := u.paymentRepo.Create(txCtx, payment); err != nil {
			return err
		}

		createdPayment = payment
		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdPayment, nil
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

	// 1. FASE MEMBACA AWAL (Di luar transaksi)
	// Ambil data payment untuk mendapatkan detail OrderID
	payment, err := u.paymentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Data pembayaran tidak ditemukan")
		}
		return err
	}

	// 🚨 MEMULAI TRANSAKSI & PENGUNCIAN 🚨
	err = u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {

		// 2. Dapatkan data Order SAMBIL DIGEMBOK (GetByIDForUpdate)
		order, err := u.orderRepo.GetByIDForUpdate(txCtx, payment.OrderID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NewError(domain.ErrBadParamInput, "Order terkait tidak ditemukan")
			}
			return err
		}

		// 3. Eksekusi penghapusan data payment (Otomatis tercatat di Kertas Buram)
		if err := u.paymentRepo.Delete(txCtx, id); err != nil {
			return err
		}

		// 4. Hitung ulang total bayar dari sisa pembayaran yang masih ada di database
		// Karena menggunakan txCtx, query ini otomatis tidak akan menyertakan payment yang baru saja dihapus di baris atas!
		remainingPayments, err := u.paymentRepo.GetByOrderID(txCtx, order.ID)
		if err != nil {
			return err
		}

		var totalPaidAfterDelete float64
		for _, p := range remainingPayments {
			totalPaidAfterDelete += p.Amount
		}

		// 5. Tentukan status pembayaran yang baru secara otomatis
		var newPaymentStatus domain.PaymentStatus
		if totalPaidAfterDelete <= 0 {
			newPaymentStatus = domain.PaymentStatusUnpaid
		} else if totalPaidAfterDelete >= order.TotalAmount {
			// Jaga-jaga jika ternyata totalnya masih menutupi tagihan
			newPaymentStatus = domain.PaymentStatusPaid
		} else {
			newPaymentStatus = domain.PaymentStatusPartial
		}

		// 6. Update status pembayaran di tabel order
		if err := u.orderRepo.UpdateStatus(txCtx, order.ID, domain.OrderStatus(""), newPaymentStatus); err != nil {
			return err
		}

		return nil // Semuanya beres, COMMIT!
	})

	return err
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

func (u *paymentUsecase) VerifyPayment(c context.Context, paymentID string, input domain.PaymentVerifyInput) (*domain.Payment, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// 1. Validasi Input Status Verifikasi
	if input.Status != domain.PaymentVerificationVerified && input.Status != domain.PaymentVerificationRejected {
		return nil, domain.NewError(domain.ErrBadParamInput, "Status verifikasi harus 'verified' atau 'rejected'")
	}

	var updatedPayment *domain.Payment

	// 2. Jalankan dalam DB Transaction (Atomic & Safe)
	err := u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		// A. Ambil data payment yang akan diverifikasi
		payment, err := u.paymentRepo.GetByID(txCtx, paymentID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NewError(domain.ErrNotFound, "Pembayaran tidak ditemukan")
			}
			return err
		}

		now := time.Now()
		// B. Update status verifikasi pembayaran di DB
		err = u.paymentRepo.UpdateVerificationStatus(txCtx, paymentID, input.Status, input.VerifiedByID, now)
		if err != nil {
			return err
		}

		// C. Ambil SELURUH histori pembayaran untuk Order ini
		allPayments, err := u.paymentRepo.GetByOrderID(txCtx, payment.OrderID)
		if err != nil {
			return err
		}

		// D. Hitung total uang yang sah (HANYA pembayaran yang berstatus VERIFIED)
		var totalPaidVerified float64
		for _, p := range allPayments {
			// Karena payment saat ini di-DB sudah ter-update statusnya, kita evaluasi status terbarunya
			currentStatus := p.Status
			if p.ID == paymentID {
				currentStatus = input.Status
			}

			if currentStatus == domain.PaymentVerificationVerified {
				totalPaidVerified += p.Amount
			}
		}

		// E. Ambil data Order terkait via orderRepo
		order, err := u.orderRepo.GetByID(txCtx, payment.OrderID)
		if err != nil {
			return err
		}

		// F. Evaluasi dan tentukan PaymentStatus baru pada Order
		if totalPaidVerified <= 0 {
			order.PaymentStatus = domain.PaymentStatusUnpaid
		} else if totalPaidVerified < order.TotalAmount {
			order.PaymentStatus = domain.PaymentStatusPartial
		} else {
			order.PaymentStatus = domain.PaymentStatusPaid
		}

		// G. Simpan perubahan PaymentStatus ke tabel orders
		if err := u.orderRepo.Update(txCtx, order); err != nil {
			return err
		}

		// H. Set data penampung untuk response
		payment.Status = input.Status
		payment.VerifiedByID = &input.VerifiedByID
		payment.VerifiedAt = &now
		updatedPayment = payment

		return nil
	})

	if err != nil {
		return nil, err
	}

	return updatedPayment, nil
}
