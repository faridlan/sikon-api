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
	paymentRepo            domain.PaymentRepository
	orderRepo              domain.OrderRepository
	bankAccountRepo        domain.BankAccountRepository
	txManager              domain.TransactionManager
	batchPoRepo            domain.BatchPORepository
	productMaterialUsecase domain.ProductMaterialUsecase // untuk kalkulasi HPP
	contextTimeout         time.Duration
}

func NewPaymentUsecase(pr domain.PaymentRepository, or domain.OrderRepository, br domain.BankAccountRepository, tx domain.TransactionManager, bpr domain.BatchPORepository, pmUsecase domain.ProductMaterialUsecase, timeout time.Duration) domain.PaymentUsecase {
	return &paymentUsecase{
		paymentRepo:            pr,
		orderRepo:              or,
		bankAccountRepo:        br,
		txManager:              tx,
		batchPoRepo:            bpr,
		productMaterialUsecase: pmUsecase,
		contextTimeout:         timeout,
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

		// 2. Hitung sisa tagihan riil (Verified + Pending dalam antrean)
		existingPayments, err := u.paymentRepo.GetByOrderID(txCtx, input.OrderID)
		if err != nil {
			return err
		}

		var totalPaidVerified float64
		var totalPaidPending float64

		for _, p := range existingPayments {
			switch p.Status {
			case domain.PaymentVerificationVerified:
				totalPaidVerified += p.Amount
			case domain.PaymentVerificationPending:
				totalPaidPending += p.Amount
			}
		}

		sisaTagihanRil := order.TotalAmount - totalPaidVerified
		maxBolehInput := sisaTagihanRil - totalPaidPending

		if sisaTagihanRil <= 0 {
			return domain.NewError(domain.ErrConflict, "Pesanan ini sudah lunas sepenuhnya")
		}

		if totalPaidPending > 0 && maxBolehInput <= 0 {
			return domain.NewError(domain.ErrConflict, fmt.Sprintf("Masih ada pembayaran dalam antrean verifikasi Accounting sebesar (Rp %.0f). Harap tunggu verifikasi selesai sebelum menginput pembayaran baru.", totalPaidPending))
		}

		if input.Amount > maxBolehInput {
			return domain.NewError(domain.ErrBadParamInput, fmt.Sprintf("Nominal pembayaran (Rp %.0f) melebihi sisa tagihan yang belum diverifikasi (Rp %.0f)", input.Amount, maxBolehInput))
		}

		// 3. Simpan payment baru (Status Verifikasi: Pending)
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

		// 🚨 ATURAN BISNIS BARU:
		// Saat Sales menginput pembayaran baru:
		// Jika status bayar order masih 'unpaid', ubah status bayar order ke 'pending' (menunggu verifikasi)
		// Jika status alur order masih 'quotation', naikkan ke 'pending'
		newOrderStatus := order.OrderStatus
		if order.OrderStatus == domain.OrderStatusQuotation {
			newOrderStatus = domain.OrderStatusPending
		}

		newPaymentStatus := order.PaymentStatus
		if order.PaymentStatus == domain.PaymentStatusUnpaid {
			newPaymentStatus = domain.PaymentStatusPending
		}

		// Update order status jika ada perubahan
		if newOrderStatus != order.OrderStatus || newPaymentStatus != order.PaymentStatus {
			if err := u.orderRepo.UpdateStatus(txCtx, order.ID, newOrderStatus, newPaymentStatus); err != nil {
				return err
			}
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

		// 1. Lock baris Order
		order, err := u.orderRepo.GetByIDForUpdate(txCtx, payment.OrderID)
		if err != nil {
			return err
		}

		// 1.5. Validasi Guarding Over-payment jika aksi verifikasi disetujui (VERIFIED)
		if input.Status == domain.PaymentVerificationVerified {
			allExistingPayments, err := u.paymentRepo.GetByOrderID(txCtx, payment.OrderID)
			if err != nil {
				return err
			}

			var currentVerifiedTotal float64
			for _, p := range allExistingPayments {
				// Hitung total bayar yang sudah VERIFIED (selain payment yang sedang diverifikasi saat ini)
				if p.ID != paymentID && p.Status == domain.PaymentVerificationVerified {
					currentVerifiedTotal += p.Amount
				}
			}

			// Cek apakah verifikasi ini akan menyebabkan over-payment
			if (currentVerifiedTotal + payment.Amount) > order.TotalAmount {
				sisaMaksimal := order.TotalAmount - currentVerifiedTotal
				return domain.NewError(domain.ErrConflict, fmt.Sprintf("Gagal verifikasi: Approval ini akan menyebabkan total pembayaran (Rp %.0f) melebihi nilai pesanan (Rp %.0f). Sisa tagihan sebenarnya hanya Rp %.0f. Silakan Reject pembayaran ganda ini.", currentVerifiedTotal+payment.Amount, order.TotalAmount, sisaMaksimal))
			}
		}

		now := time.Now()

		// 2. Update status verifikasi pembayaran
		err = u.paymentRepo.UpdateVerificationStatus(txCtx, paymentID, input.Status, input.VerifiedByID, now)
		if err != nil {
			return err
		}

		// 3. Ambil seluruh pembayaran untuk kalkulasi ulang total bayar sah
		allPayments, err := u.paymentRepo.GetByOrderID(txCtx, payment.OrderID)
		if err != nil {
			return err
		}

		var totalPaidVerified float64
		var hasPendingPayment bool

		for _, p := range allPayments {
			currStatus := p.Status
			if p.ID == paymentID {
				currStatus = input.Status
			}

			switch currStatus {
			case domain.PaymentVerificationVerified:
				totalPaidVerified += p.Amount
			case domain.PaymentVerificationPending:
				hasPendingPayment = true
			}
		}

		// Kalkulasi Status Pembayaran (PaymentStatus) pada Order
		var newPaymentStatus domain.PaymentStatus
		if totalPaidVerified <= 0 {
			if hasPendingPayment {
				newPaymentStatus = domain.PaymentStatusPending
			} else {
				newPaymentStatus = domain.PaymentStatusUnpaid
			}
		} else if totalPaidVerified < order.TotalAmount {
			newPaymentStatus = domain.PaymentStatusPartial
		} else {
			// Jika ada pembayaran pending lain yang belum diverifikasi, status bayar tetap 'partial'
			if hasPendingPayment {
				newPaymentStatus = domain.PaymentStatusPartial
			} else {
				newPaymentStatus = domain.PaymentStatusPaid
			}
		}

		// =========================================================================
		// 🚨 LOGICA BARU ALUR ORDER & APPROVAL
		// =========================================================================
		newOrderStatus := order.OrderStatus

		switch input.Status {
		case domain.PaymentVerificationVerified:
			// 1. ISI APPROVED_AT: Jika ini adalah pembayaran pertama yang di-approve, catat waktunya!
			if order.ApprovedAt == nil {
				order.ApprovedAt = &now
			}

			// 2. OTOMATIS NAIK KE PRODUCTION:
			// Jika status order saat ini 'quotation' atau 'pending', langsung masuk meja produksi!
			if order.OrderStatus == domain.OrderStatusQuotation || order.OrderStatus == domain.OrderStatusPending {
				newOrderStatus = domain.OrderStatusProduction
			}

			// 3. JIKA PELUNASAN: otomatis selesaikan order ke 'completed' jika barang sudah 'ready'
			if order.OrderStatus == domain.OrderStatusReady && newPaymentStatus == domain.PaymentStatusPaid {
				newOrderStatus = domain.OrderStatusCompleted
			}

		case domain.PaymentVerificationRejected:
			// Jika pembayaran DITOLAK dan tidak ada uang verified sama sekali
			if totalPaidVerified <= 0 && !hasPendingPayment {
				// Turunkan kembali ke quotation dan HAPUS approved_at
				if order.OrderStatus == domain.OrderStatusPending || order.OrderStatus == domain.OrderStatusProduction {
					newOrderStatus = domain.OrderStatusQuotation
					order.ApprovedAt = nil
				}
			}
		}

		// 4. Update data Order secara utuh ke database (MENGGUNAKAN UPDATE, BUKAN UPDATESTATUS)
		// Supaya field ApprovedAt ikut tersimpan ke PostgreSQL.
		order.OrderStatus = newOrderStatus
		order.PaymentStatus = newPaymentStatus

		if err := u.orderRepo.Update(txCtx, order); err != nil {
			return err
		}

		// 🚨 BEKUKAN HPP MATERIAL
		// Hitung HPP ketika order pertama kali masuk production dan belum pernah dihitung (atau masih 0).
		if newOrderStatus == domain.OrderStatusProduction && (order.HPPCalculatedAt == nil || order.HPPMaterialCost == 0) {
			fullOrder, err := u.orderRepo.GetByID(txCtx, order.ID)
			if err != nil {
				return err
			}

			var materialCost float64
			for _, item := range fullOrder.Items {
				var cost float64
				var err error
				if item.FabricID != nil && *item.FabricID != "" {
					cost, err = u.productMaterialUsecase.CalculateMaterialCost(txCtx, item.ProductID, item.Qty, *item.FabricID)
				} else {
					cost, err = u.productMaterialUsecase.CalculateMaterialCost(txCtx, item.ProductID, item.Qty)
				}
				if err != nil {
					return err
				}
				materialCost += cost
			}

			if err := u.orderRepo.UpdateHPP(txCtx, order.ID, materialCost, now); err != nil {
				return err
			}
			order.HPPMaterialCost = materialCost
			order.HPPCalculatedAt = &now
		}
		// =========================================================================

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
