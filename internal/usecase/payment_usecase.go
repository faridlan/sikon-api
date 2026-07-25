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

	// Validasi Rekening Bank (Bisa di luar transaksi)
	if _, err := u.bankAccountRepo.GetByID(ctx, input.BankAccountID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrBadParamInput, "Rekening bank tidak valid")
		}
		return nil, err
	}

	var savedPayment *domain.Payment // Ember penampung hasil

	// 🚨 MEMULAI TRANSAKSI KEUANGAN 🚨
	err := u.txManager.RunInTransaction(ctx, func(txCtx context.Context) error {

		// 1. Ambil Order SAMBIL DIGEMBOK (Mencegah admin lain memproses order yang sama)
		order, err := u.orderRepo.GetByIDForUpdate(txCtx, input.OrderID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.NewError(domain.ErrBadParamInput, "Order tidak ditemukan")
			}
			return err
		}

		grandTotal := order.TotalAmount

		// 2. Hitung total uang masuk sebelumnya (Dalam kondisi database masih digembok)
		existingPayments, err := u.paymentRepo.GetByOrderID(txCtx, input.OrderID)
		if err != nil {
			return err
		}

		var totalPaid float64
		for _, p := range existingPayments {
			totalPaid += p.Amount
		}

		// 3. Validasi
		if totalPaid >= grandTotal {
			return domain.NewError(domain.ErrConflict, "Pesanan ini sudah lunas sepenuhnya")
		}

		sisaTagihan := grandTotal - totalPaid
		if input.Amount > sisaTagihan {
			return domain.NewError(domain.ErrBadParamInput, "Jumlah bayar melebihi sisa tagihan. Sisa tagihan: Rp "+fmt.Sprintf("%.0f", sisaTagihan))
		}

		// 4. Save Payment (Catat ke buku)
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

		if err := u.paymentRepo.Create(txCtx, payment); err != nil {
			return err
		}

		// 5. Update Status Order & Re-alokasi PO (Jika masih Quotation)
		totalPaidSetelahMasuk := totalPaid + payment.Amount
		newPaymentStatus := domain.PaymentStatusPartial

		if totalPaidSetelahMasuk >= grandTotal {
			newPaymentStatus = domain.PaymentStatusPaid
		}

		// 🚨 LOGIC RE-ALOKASI PO OTOMATIS 🚨
		// Jika ini adalah pembayaran pertama, status berubah menjadi Production
		if order.OrderStatus == domain.OrderStatusQuotation {
			order.OrderStatus = domain.OrderStatusProduction
			now := time.Now()
			order.ApprovedAt = &now

			// Cari PO yang aktif hari ini
			activePO, err := u.batchPoRepo.GetActivePOByDate(txCtx, now)
			if err == nil && activePO != nil {
				// Pindahkan order ini ke PO yang sedang aktif saat DP dibayar
				order.BatchPoID = activePO.ID
			}
		}

		order.PaymentStatus = newPaymentStatus

		// Gunakan Update penuh karena kita mengubah banyak field (bukan cuma status)
		if err := u.orderRepo.Update(txCtx, order); err != nil {
			return err
		}

		savedPayment = payment
		return nil // Transaksi sukses. COMMIT & LEPASKAN GEMBOK!
	})

	if err != nil {
		return nil, err
	}

	return savedPayment, nil
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
