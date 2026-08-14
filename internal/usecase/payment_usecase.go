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
	orderRepo       domain.OrderRepository
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
		// 1. Ambil order dengan lock (FOR UPDATE)
		order, err := u.orderRepo.GetByIDForUpdate(txCtx, input.OrderID)
		if err != nil {
			return err
		}

		// 2. Ambil payments
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

	payment, err := u.paymentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewError(domain.ErrNotFound, "Data pembayaran tidak ditemukan")
		}
		return err
	}

	err = u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		// 1. Dapatkan data Order SAMBIL DIGEMBOK (FOR UPDATE)
		order, err := u.orderRepo.GetByIDForUpdate(txCtx, payment.OrderID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NewError(domain.ErrBadParamInput, "Order terkait tidak ditemukan")
			}
			return err
		}

		// 2. Eksekusi penghapusan data payment
		if err := u.paymentRepo.Delete(txCtx, id); err != nil {
			return err
		}

		// 3. Hitung ulang total bayar dari sisa pembayaran yang terverifikasi (VERIFIED ONLY)
		remainingPayments, err := u.paymentRepo.GetByOrderID(txCtx, order.ID)
		if err != nil {
			return err
		}

		var totalPaidAfterDelete float64
		for _, p := range remainingPayments {
			// 🚨 FIX BUG: Hanya hitung pembayaran yang terverifikasi!
			if p.Status == domain.PaymentVerificationVerified {
				totalPaidAfterDelete += p.Amount
			}
		}

		// 4. Tentukan status pembayaran yang baru secara otomatis
		var newPaymentStatus domain.PaymentStatus
		if totalPaidAfterDelete <= 0 {
			newPaymentStatus = domain.PaymentStatusUnpaid
		} else if totalPaidAfterDelete >= order.TotalAmount {
			newPaymentStatus = domain.PaymentStatusPaid
		} else {
			newPaymentStatus = domain.PaymentStatusPartial
		}

		// 5. Update status pembayaran di tabel order
		if err := u.orderRepo.UpdateStatus(txCtx, order.ID, domain.OrderStatus(""), newPaymentStatus); err != nil {
			return err
		}

		return nil
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

	if input.Status != domain.PaymentVerificationVerified && input.Status != domain.PaymentVerificationRejected {
		return nil, domain.NewError(domain.ErrBadParamInput, "Status verifikasi harus 'verified' atau 'rejected'")
	}

	var updatedPayment *domain.Payment

	err := u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {
		payment, err := u.paymentRepo.GetByID(txCtx, paymentID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NewError(domain.ErrNotFound, "Pembayaran tidak ditemukan")
			}
			return err
		}

		// 🚨 FIX RACE CONDITION #1: Gembok baris Order dengan GetByIDForUpdate
		order, err := u.orderRepo.GetByIDForUpdate(txCtx, payment.OrderID)
		if err != nil {
			return err
		}

		now := time.Now()

		// 🚨 FIX RACE CONDITION #2: Guarding di DB level (Cek status = pending & RowsAffected)
		err = u.paymentRepo.UpdateVerificationStatus(txCtx, paymentID, input.Status, input.VerifiedByID, now)
		if err != nil {
			return err
		}

		// Ambil seluruh pembayaran terbaru
		allPayments, err := u.paymentRepo.GetByOrderID(txCtx, payment.OrderID)
		if err != nil {
			return err
		}

		// Hitung total bayar sah (HANYA status VERIFIED)
		var totalPaidVerified float64
		for _, p := range allPayments {
			currentStatus := p.Status
			if p.ID == paymentID {
				currentStatus = input.Status
			}

			if currentStatus == domain.PaymentVerificationVerified {
				totalPaidVerified += p.Amount
			}
		}

		// Evaluasi PaymentStatus baru
		var newPaymentStatus domain.PaymentStatus
		if totalPaidVerified <= 0 {
			newPaymentStatus = domain.PaymentStatusUnpaid
		} else if totalPaidVerified < order.TotalAmount {
			newPaymentStatus = domain.PaymentStatusPartial
		} else {
			newPaymentStatus = domain.PaymentStatusPaid
		}

		// Update PaymentStatus di tabel orders
		if err := u.orderRepo.UpdateStatus(txCtx, order.ID, domain.OrderStatus(""), newPaymentStatus); err != nil {
			return err
		}

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
