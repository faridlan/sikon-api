package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/domain/mocks"
	"github.com/faridlan/sikon-api/internal/usecase"
)

// setupPaymentTest adalah helper untuk menginisialisasi ke-3 mock repository
func setupPaymentTest() (*mocks.PaymentRepository, *mocks.OrderRepository, *mocks.BankAccountRepository, *mocks.TransactionManager, *mocks.BatchPORepository, domain.PaymentUsecase) {
	mockPaymentRepo := new(mocks.PaymentRepository)
	mockOrderRepo := new(mocks.OrderRepository)
	mockBankAccountRepo := new(mocks.BankAccountRepository)
	mockTxManager := new(mocks.TransactionManager)
	mockBatchPORepo := new(mocks.BatchPORepository)

	uc := usecase.NewPaymentUsecase(mockPaymentRepo, mockOrderRepo, mockBankAccountRepo, mockTxManager, mockBatchPORepo, time.Second*2)

	return mockPaymentRepo, mockOrderRepo, mockBankAccountRepo, mockTxManager, mockBatchPORepo, uc
}

func TestPaymentUsecase_ProcessPayment(t *testing.T) {
	// Variabel input dan ekspektasi awal bisa ditaruh di luar karena statis (tidak berubah)
	input := domain.PaymentCreateInput{
		OrderID:         "ord-123",
		BankAccountID:   "bank-123",
		Amount:          500000,
		ReferenceNumber: "TRX-001",
		PaymentType:     domain.PaymentTypeDP,
	}

	mockOrder := &domain.Order{
		ID:          "ord-123",
		TotalAmount: 1000000,
		OrderStatus: domain.OrderStatusQuotation, // Set awal sebagai Quotation
		BatchPoID:   "po-lama-juli",
	}

	t.Run("Success - Partial Payment (DP) & Auto PO Reallocation", func(t *testing.T) {
		// 🚨 PINDAHKAN KE SINI: Inisiasi ulang mock agar 100% fresh untuk test ini
		mockPaymentRepo, mockOrderRepo, mockBankRepo, mockTxManager, mockBatchPoRepo, uc := setupPaymentTest()

		// Helper Transaksi: Cukup gunakan mock.Anything untuk menghindari panic tipe data
		mockTxManager.On("RunInTransaction", mock.Anything, mock.Anything).
			Return(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()

		// 1. Bank Ditemukan
		mockBankRepo.On("GetByID", mock.Anything, input.BankAccountID).Return(&domain.BankAccount{ID: "bank-123"}, nil).Once()

		// 2. Order Ditemukan DENGAN GEMBOK
		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, input.OrderID).Return(mockOrder, nil).Once()

		// 3. Belum ada pembayaran sebelumnya
		mockPaymentRepo.On("GetByOrderID", mock.Anything, input.OrderID).Return([]domain.Payment{}, nil).Once()

		// 4. Simpan Payment
		mockPaymentRepo.On("Create", mock.Anything, mock.MatchedBy(func(p *domain.Payment) bool {
			return p.Amount == input.Amount && p.ReferenceNumber == input.ReferenceNumber
		})).Return(nil).Once()

		// 4.5 MOCK: Pencarian PO Aktif (Gunakan mock.Anything untuk amannya)
		activePO := &domain.BatchPO{ID: "po-baru-agustus"}
		mockBatchPoRepo.On("GetActivePOByDate", mock.Anything, mock.Anything).Return(activePO, nil).Once()

		// 5. Update Order (Update Penuh)
		mockOrderRepo.On("Update", mock.Anything, mock.MatchedBy(func(o *domain.Order) bool {
			return o.OrderStatus == domain.OrderStatusProduction &&
				o.PaymentStatus == domain.PaymentStatusPartial &&
				o.BatchPoID == "po-baru-agustus" &&
				o.ApprovedAt != nil
		})).Return(nil).Once()

		payment, err := uc.ProcessPayment(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, payment)
		mockTxManager.AssertExpectations(t)
	})

	t.Run("Error - Overpayment", func(t *testing.T) {
		// 🚨 PINDAHKAN KE SINI JUGA: Mock baru, tidak akan tercampur dengan test case di atas
		mockPaymentRepo, mockOrderRepo, mockBankRepo, mockTxManager, mockBatchPoRepo, uc := setupPaymentTest()

		inputOverpayment := input
		inputOverpayment.Amount = 600000

		existingPayments := []domain.Payment{
			{Amount: 500000},
		}

		mockTxManager.On("RunInTransaction", mock.Anything, mock.Anything).
			Return(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()

		mockBankRepo.On("GetByID", mock.Anything, input.BankAccountID).Return(&domain.BankAccount{ID: "bank-123"}, nil).Once()
		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, input.OrderID).Return(mockOrder, nil).Once()
		mockPaymentRepo.On("GetByOrderID", mock.Anything, input.OrderID).Return(existingPayments, nil).Once()

		payment, err := uc.ProcessPayment(context.Background(), inputOverpayment)

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)
		assert.Nil(t, payment)

		// Assert Not Called agar memastikan data aman
		mockPaymentRepo.AssertNotCalled(t, "Create")
		mockOrderRepo.AssertNotCalled(t, "Update")
		mockBatchPoRepo.AssertNotCalled(t, "GetActivePOByDate")
	})
}

func TestPaymentUsecase_GetPayment(t *testing.T) {
	mockPaymentRepo, _, _, _, _, uc := setupPaymentTest()
	mockID := "pay-123"

	t.Run("Success", func(t *testing.T) {
		mockPaymentRepo.On("GetByID", mock.Anything, mockID).Return(&domain.Payment{ID: mockID}, nil).Once()

		result, err := uc.GetPayment(context.Background(), mockID)

		assert.NoError(t, err)
		assert.Equal(t, mockID, result.ID)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockPaymentRepo.On("GetByID", mock.Anything, mockID).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.GetPayment(context.Background(), mockID)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestPaymentUsecase_ListPayments(t *testing.T) {
	mockPaymentRepo, _, _, _, _, uc := setupPaymentTest() // Sesuaikan dengan setup test Anda
	query := domain.PaginationQuery{Page: 1, Limit: 10}

	t.Run("Success_TanpaFilter", func(t *testing.T) {
		emptyFilter := domain.PaymentFilter{} // Filter kosong
		mockPayments := []domain.Payment{{ID: "1"}, {ID: "2"}}

		// Tambahkan emptyFilter pada .On("Fetch")
		mockPaymentRepo.On("Fetch", mock.Anything, 10, 0, emptyFilter).Return(mockPayments, int64(2), nil).Once()

		// Tambahkan emptyFilter pada pemanggilan ListPayments
		payments, meta, err := uc.ListPayments(context.Background(), query, emptyFilter)

		assert.NoError(t, err)
		assert.Len(t, payments, 2)
		assert.Equal(t, 1, meta.TotalPages)
	})

	t.Run("Success_DenganFilter", func(t *testing.T) {
		// Simulasi Frontend mengirim query params lengkap
		activeFilter := domain.PaymentFilter{
			Search:      "TRX-123",
			PaymentType: string(domain.PaymentTypeDP),
			StartDate:   "2023-10-01",
			EndDate:     "2023-10-31",
		}
		mockPayments := []domain.Payment{{ID: "1"}} // Misalnya hasil pencarian hanya 1

		// Pastikan Mocking mengharapkan activeFilter
		mockPaymentRepo.On("Fetch", mock.Anything, 10, 0, activeFilter).Return(mockPayments, int64(1), nil).Once()

		payments, meta, err := uc.ListPayments(context.Background(), query, activeFilter)

		assert.NoError(t, err)
		assert.Len(t, payments, 1) // Memastikan data sesuai return mock (1 data)
		assert.Equal(t, 1, meta.TotalPages)
		assert.Equal(t, int64(1), meta.TotalItems)
	})

	t.Run("Error_DariRepository", func(t *testing.T) {
		emptyFilter := domain.PaymentFilter{}

		// Simulasi jika Database sedang down atau error
		mockPaymentRepo.On("Fetch", mock.Anything, 10, 0, emptyFilter).Return(nil, int64(0), errors.New("database connection failed")).Once()

		payments, _, err := uc.ListPayments(context.Background(), query, emptyFilter)

		assert.Error(t, err)
		assert.Nil(t, payments)
	})
}

func TestPaymentUsecase_UpdatePayment(t *testing.T) {
	mockPaymentRepo, _, _, _, _, uc := setupPaymentTest()
	mockID := "pay-123"
	input := domain.PaymentUpdateInput{ReferenceNumber: "TRX-REVISI"}

	t.Run("Success", func(t *testing.T) {
		existingPayment := &domain.Payment{ID: mockID, ReferenceNumber: "TRX-LAMA"}
		mockPaymentRepo.On("GetByID", mock.Anything, mockID).Return(existingPayment, nil).Once()

		mockPaymentRepo.On("Update", mock.Anything, mock.MatchedBy(func(p *domain.Payment) bool {
			return p.ReferenceNumber == "TRX-REVISI"
		})).Return(nil).Once()

		payment, err := uc.UpdatePayment(context.Background(), mockID, input)

		assert.NoError(t, err)
		assert.NotNil(t, payment)
	})
}

func TestPaymentUsecase_DeletePayment(t *testing.T) {
	// 🚨 Pastikan setupPaymentTest() mengembalikan mockTxManager juga!
	// Urutan return biasanya: PaymentRepo, OrderRepo, BankRepo, TxManager, Usecase
	mockPaymentRepo, mockOrderRepo, _, mockTxManager, _, uc := setupPaymentTest()

	paymentID := "pay-123"
	orderID := "ord-123"

	// Setup data mock dasar
	mockPayment := &domain.Payment{
		ID:      paymentID,
		OrderID: orderID,
		Amount:  500000,
	}

	mockOrder := &domain.Order{
		ID:          orderID,
		TotalAmount: 1000000, // Total tagihan 1 juta
	}

	// Helper untuk mock transaksi agar closure tereksekusi
	mockTransaction := func() {
		mockTxManager.ExpectedCalls = nil
		mockTxManager.On("RunInTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()
	}

	t.Run("Success - Status Reverts to Partial", func(t *testing.T) {
		mockPaymentRepo.ExpectedCalls = nil
		mockOrderRepo.ExpectedCalls = nil

		// 1. Ambil data payment di awal (Luar Transaksi)
		mockPaymentRepo.On("GetByID", mock.Anything, paymentID).Return(mockPayment, nil).Once()

		// 🚨 Buka Transaksi
		mockTransaction()

		// 2. Gembok Order (Dalam Transaksi)
		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, orderID).Return(mockOrder, nil).Once()

		// 3. Hapus Payment (Dalam Transaksi)
		mockPaymentRepo.On("Delete", mock.Anything, paymentID).Return(nil).Once()

		// 4. Ambil SISA payment (Kondisi SETELAH payment pertama dihapus)
		// Skenario: Awalnya customer bayar DP 2x @500rb. Karena 1 dihapus, mock ini mereturn sisa 1.
		remainingPayments := []domain.Payment{
			{ID: "pay-456", Amount: 500000},
		}
		mockPaymentRepo.On("GetByOrderID", mock.Anything, orderID).Return(remainingPayments, nil).Once()

		// 5. Ekspektasi: Karena sisa 500rb < TotalAmount(1jt), status otomatis Partial
		mockOrderRepo.On("UpdateStatus", mock.Anything, orderID, domain.OrderStatus(""), domain.PaymentStatusPartial).Return(nil).Once()

		err := uc.DeletePayment(context.Background(), paymentID)

		assert.NoError(t, err)
		mockPaymentRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
	})

	t.Run("Success - Status Reverts to Unpaid", func(t *testing.T) {
		mockPaymentRepo.ExpectedCalls = nil
		mockOrderRepo.ExpectedCalls = nil

		// 1. Ambil data payment
		mockPaymentRepo.On("GetByID", mock.Anything, paymentID).Return(mockPayment, nil).Once()

		// 🚨 Buka Transaksi
		mockTransaction()

		// 2. Gembok Order
		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, orderID).Return(mockOrder, nil).Once()

		// 3. Hapus Payment
		mockPaymentRepo.On("Delete", mock.Anything, paymentID).Return(nil).Once()

		// 4. Ambil SISA payment (Skenario: Hanya ada 1x DP, jika dihapus sisanya KOSONG)
		mockPaymentRepo.On("GetByOrderID", mock.Anything, orderID).Return([]domain.Payment{}, nil).Once()

		// 5. Ekspektasi: Karena 0, status otomatis jadi Unpaid
		mockOrderRepo.On("UpdateStatus", mock.Anything, orderID, domain.OrderStatus(""), domain.PaymentStatusUnpaid).Return(nil).Once()

		err := uc.DeletePayment(context.Background(), paymentID)

		assert.NoError(t, err)
		mockPaymentRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
	})

	t.Run("Error - Payment Not Found", func(t *testing.T) {
		mockPaymentRepo.ExpectedCalls = nil
		mockOrderRepo.ExpectedCalls = nil
		mockTxManager.ExpectedCalls = nil

		// Skenario: Data yang mau dihapus tidak ada di DB (Gagal di Fase 1 luar transaksi)
		mockPaymentRepo.On("GetByID", mock.Anything, paymentID).Return(nil, domain.ErrNotFound).Once()

		err := uc.DeletePayment(context.Background(), paymentID)

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrNotFound, appErr.ErrType)

		// Pastikan method di dalam transaksi tidak ada yang tereksekusi!
		mockTxManager.AssertNotCalled(t, "RunInTransaction")
		mockOrderRepo.AssertNotCalled(t, "GetByIDForUpdate")
		mockPaymentRepo.AssertNotCalled(t, "Delete")
	})

	t.Run("Error - Order Not Found", func(t *testing.T) {
		mockPaymentRepo.ExpectedCalls = nil
		mockOrderRepo.ExpectedCalls = nil
		mockTxManager.ExpectedCalls = nil

		// Fase 1 Lolos
		mockPaymentRepo.On("GetByID", mock.Anything, paymentID).Return(mockPayment, nil).Once()

		// Buka Transaksi
		mockTransaction()

		// Skenario: Payment ada, tapi saat mencoba digembok, Ordernya mendadak hilang
		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, orderID).Return(nil, domain.ErrNotFound).Once()

		err := uc.DeletePayment(context.Background(), paymentID)

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)
		assert.Contains(t, appErr.Message, "Order terkait tidak ditemukan")

		// Pastikan tidak ada penghapusan data
		mockPaymentRepo.AssertNotCalled(t, "Delete")
	})
}

func TestVerifyPayment_Success(t *testing.T) {
	mockPaymentRepo, mockOrderRepo, _, mockTxManager, _, uc := setupPaymentTest()

	paymentID := "pay-uuid-1"
	orderID := "ord-uuid-1"
	verifiedByID := "finance-uuid-1"

	paymentInput := domain.PaymentVerifyInput{
		Status:       domain.PaymentVerificationVerified,
		VerifiedByID: verifiedByID,
	}

	existingPayment := &domain.Payment{
		ID:          paymentID,
		OrderID:     orderID,
		Amount:      500000,
		Status:      domain.PaymentVerificationPending,
		PaymentType: domain.PaymentTypeDP,
	}

	allPayments := []domain.Payment{
		{
			ID:      paymentID,
			OrderID: orderID,
			Amount:  500000,
			Status:  domain.PaymentVerificationVerified, // Status setelah ter-update
		},
	}

	existingOrder := &domain.Order{
		ID:            orderID,
		TotalAmount:   1000000,
		PaymentStatus: domain.PaymentStatusUnpaid,
	}

	// Mock Transaction Manager
	mockTxManager.On("RunInTransaction", mock.Anything, mock.Anything).
		Return(func(ctx context.Context, fn func(txCtx context.Context) error) error {
			return fn(ctx)
		})

	// Expectations
	mockPaymentRepo.On("GetByID", mock.Anything, paymentID).Return(existingPayment, nil)
	mockPaymentRepo.On("UpdateVerificationStatus", mock.Anything, paymentID, domain.PaymentVerificationVerified, verifiedByID, mock.AnythingOfType("time.Time")).Return(nil)
	mockPaymentRepo.On("GetByOrderID", mock.Anything, orderID).Return(allPayments, nil)
	mockOrderRepo.On("GetByID", mock.Anything, orderID).Return(existingOrder, nil)

	// Harapannya PaymentStatus order berubah jadi "partial" karena total paid (500rb) < total amount (1jt)
	mockOrderRepo.On("Update", mock.Anything, mock.MatchedBy(func(o *domain.Order) bool {
		return o.ID == orderID && o.PaymentStatus == domain.PaymentStatusPartial
	})).Return(nil)

	// Execute Test
	res, err := uc.VerifyPayment(context.Background(), paymentID, paymentInput)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, domain.PaymentVerificationVerified, res.Status)
	assert.Equal(t, verifiedByID, *res.VerifiedByID)

	mockTxManager.AssertExpectations(t)
	mockPaymentRepo.AssertExpectations(t)
	mockOrderRepo.AssertExpectations(t)
}

func TestVerifyPayment_InvalidStatus_Error(t *testing.T) {
	_, _, _, _, _, uc := setupPaymentTest()

	input := domain.PaymentVerifyInput{
		Status:       domain.PaymentVerificationStatus("invalid_status"),
		VerifiedByID: "finance-uuid-1",
	}

	res, err := uc.VerifyPayment(context.Background(), "pay-uuid-1", input)

	assert.Error(t, err)
	assert.Nil(t, res)

	var appErr *domain.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)
}

func TestVerifyPayment_PaymentNotFound_Error(t *testing.T) {
	mockPaymentRepo, _, _, mockTxManager, _, uc := setupPaymentTest()

	paymentID := "non-existing-id"
	input := domain.PaymentVerifyInput{
		Status:       domain.PaymentVerificationVerified,
		VerifiedByID: "finance-uuid-1",
	}

	mockTxManager.On("RunInTransaction", mock.Anything, mock.Anything).
		Return(func(ctx context.Context, fn func(txCtx context.Context) error) error {
			return fn(ctx)
		})

	mockPaymentRepo.On("GetByID", mock.Anything, paymentID).Return(nil, domain.ErrNotFound)

	res, err := uc.VerifyPayment(context.Background(), paymentID, input)

	assert.Error(t, err)
	assert.Nil(t, res)

	var appErr *domain.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, domain.ErrNotFound, appErr.ErrType)

	mockTxManager.AssertExpectations(t)
	mockPaymentRepo.AssertExpectations(t)
}

func TestVerifyPayment_RejectedPayment_SetsOrderUnpaid(t *testing.T) {
	mockPaymentRepo, mockOrderRepo, _, mockTxManager, _, uc := setupPaymentTest()

	paymentID := "pay-uuid-2"
	orderID := "ord-uuid-2"
	verifiedByID := "finance-uuid-1"

	paymentInput := domain.PaymentVerifyInput{
		Status:       domain.PaymentVerificationRejected,
		VerifiedByID: verifiedByID,
	}

	existingPayment := &domain.Payment{
		ID:      paymentID,
		OrderID: orderID,
		Amount:  500000,
		Status:  domain.PaymentVerificationPending,
	}

	// Pembayaran ditolak, jadi total verified payment = 0
	allPayments := []domain.Payment{
		{
			ID:      paymentID,
			OrderID: orderID,
			Amount:  500000,
			Status:  domain.PaymentVerificationRejected,
		},
	}

	existingOrder := &domain.Order{
		ID:            orderID,
		TotalAmount:   1000000,
		PaymentStatus: domain.PaymentStatusPartial,
	}

	mockTxManager.On("RunInTransaction", mock.Anything, mock.Anything).
		Return(func(ctx context.Context, fn func(txCtx context.Context) error) error {
			return fn(ctx)
		})

	mockPaymentRepo.On("GetByID", mock.Anything, paymentID).Return(existingPayment, nil)
	mockPaymentRepo.On("UpdateVerificationStatus", mock.Anything, paymentID, domain.PaymentVerificationRejected, verifiedByID, mock.AnythingOfType("time.Time")).Return(nil)
	mockPaymentRepo.On("GetByOrderID", mock.Anything, orderID).Return(allPayments, nil)
	mockOrderRepo.On("GetByID", mock.Anything, orderID).Return(existingOrder, nil)

	// Harapannya PaymentStatus order kembali menjadi "unpaid" karena tidak ada payment verified
	mockOrderRepo.On("Update", mock.Anything, mock.MatchedBy(func(o *domain.Order) bool {
		return o.ID == orderID && o.PaymentStatus == domain.PaymentStatusUnpaid
	})).Return(nil)

	res, err := uc.VerifyPayment(context.Background(), paymentID, paymentInput)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, domain.PaymentVerificationRejected, res.Status)

	mockTxManager.AssertExpectations(t)
	mockPaymentRepo.AssertExpectations(t)
	mockOrderRepo.AssertExpectations(t)
}
