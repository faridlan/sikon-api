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
func setupPaymentTest() (*mocks.PaymentRepository, *mocks.OrderRepository, *mocks.BankAccountRepository, *mocks.TransactionManager, domain.PaymentUsecase) {
	mockPaymentRepo := new(mocks.PaymentRepository)
	mockOrderRepo := new(mocks.OrderRepository)
	mockBankAccountRepo := new(mocks.BankAccountRepository)
	mockTxManager := new(mocks.TransactionManager)

	uc := usecase.NewPaymentUsecase(mockPaymentRepo, mockOrderRepo, mockBankAccountRepo, mockTxManager, time.Second*2)

	return mockPaymentRepo, mockOrderRepo, mockBankAccountRepo, mockTxManager, uc
}

func TestPaymentUsecase_ProcessPayment(t *testing.T) {
	// Pastikan setupPaymentTest() mengembalikan mockTxManager juga
	mockPaymentRepo, mockOrderRepo, mockBankRepo, mockTxManager, uc := setupPaymentTest()

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
	}

	// Helper untuk mock transaksi agar tidak mengulang kode
	mockTransaction := func() {
		mockTxManager.ExpectedCalls = nil
		mockTxManager.On("RunInTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(func(ctx context.Context, fn func(context.Context) error) error {
				// Eksekusi fungsi closure TEPAT SATU KALI dan kembalikan error-nya ke Usecase
				return fn(ctx)
			}).Once()
	}

	t.Run("Success - Partial Payment (DP)", func(t *testing.T) {
		mockOrderRepo.ExpectedCalls = nil
		mockBankRepo.ExpectedCalls = nil
		mockPaymentRepo.ExpectedCalls = nil

		// 1. Bank Ditemukan (Di luar transaksi)
		mockBankRepo.On("GetByID", mock.Anything, input.BankAccountID).Return(&domain.BankAccount{ID: "bank-123"}, nil).Once()

		// 🚨 Mulai Transaksi
		mockTransaction()

		// 2. Order Ditemukan DENGAN GEMBOK (Di dalam transaksi)
		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, input.OrderID).Return(mockOrder, nil).Once()

		// 3. Belum ada pembayaran sebelumnya (totalPaid = 0)
		mockPaymentRepo.On("GetByOrderID", mock.Anything, input.OrderID).Return([]domain.Payment{}, nil).Once()

		// 4. Simpan Payment
		mockPaymentRepo.On("Create", mock.Anything, mock.MatchedBy(func(p *domain.Payment) bool {
			return p.Amount == input.Amount && p.ReferenceNumber == input.ReferenceNumber
		})).Return(nil).Once()

		// 5. Update Status Order -> Partial
		mockOrderRepo.On("UpdateStatus", mock.Anything, mockOrder.ID, domain.OrderStatus(""), domain.PaymentStatusPartial).Return(nil).Once()

		payment, err := uc.ProcessPayment(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, payment)
		mockTxManager.AssertExpectations(t)
	})

	t.Run("Error - Overpayment", func(t *testing.T) {
		mockOrderRepo.ExpectedCalls = nil
		mockBankRepo.ExpectedCalls = nil
		mockPaymentRepo.ExpectedCalls = nil

		inputOverpayment := input
		inputOverpayment.Amount = 600000 // Simulasi bayar 600rb

		existingPayments := []domain.Payment{
			{Amount: 500000}, // Sudah DP 500rb, sisa tagihan 500rb
		}

		// 1. Bank Valid
		mockBankRepo.On("GetByID", mock.Anything, input.BankAccountID).Return(&domain.BankAccount{ID: "bank-123"}, nil).Once()

		// 🚨 Mulai Transaksi
		mockTransaction()

		// 2. Gembok Order
		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, input.OrderID).Return(mockOrder, nil).Once()

		// 3. Ambil total yang sudah dibayar
		mockPaymentRepo.On("GetByOrderID", mock.Anything, input.OrderID).Return(existingPayments, nil).Once()

		// HARUS GAGAL DI SINI KARENA OVERPAYMENT, Create & UpdateStatus tidak boleh dipanggil!

		payment, err := uc.ProcessPayment(context.Background(), inputOverpayment)

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)
		assert.Contains(t, appErr.Message, "Jumlah bayar melebihi sisa tagihan")
		assert.Nil(t, payment)

		mockPaymentRepo.AssertNotCalled(t, "Create")
		mockOrderRepo.AssertNotCalled(t, "UpdateStatus")
	})
}

func TestPaymentUsecase_GetPayment(t *testing.T) {
	mockPaymentRepo, _, _, _, uc := setupPaymentTest()
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
	mockPaymentRepo, _, _, _, uc := setupPaymentTest() // Sesuaikan dengan setup test Anda
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
	mockPaymentRepo, _, _, _, uc := setupPaymentTest()
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
	// Update destructuring: kita butuh mockOrderRepo sekarang!
	mockPaymentRepo, mockOrderRepo, _, _, uc := setupPaymentTest()

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

	t.Run("Success - Status Reverts to Partial", func(t *testing.T) {
		// Skenario: Customer bayar DP 2x @500rb (Lunas).
		// Kasir menghapus pembayaran pertama, sehingga order kembali jadi "Partial".
		existingPayments := []domain.Payment{
			{ID: "pay-123", Amount: 500000},
			{ID: "pay-456", Amount: 500000},
		}

		mockPaymentRepo.On("GetByID", mock.Anything, paymentID).Return(mockPayment, nil).Once()
		mockOrderRepo.On("GetByID", mock.Anything, orderID).Return(mockOrder, nil).Once()
		mockPaymentRepo.On("GetByOrderID", mock.Anything, orderID).Return(existingPayments, nil).Once()

		// Ekspektasi: 1jt - 500rb = 500rb. Karena 500rb < TotalAmount(1jt), status = Partial
		mockOrderRepo.On("UpdateStatus", mock.Anything, orderID, domain.OrderStatus(""), domain.PaymentStatusPartial).Return(nil).Once()
		mockPaymentRepo.On("Delete", mock.Anything, paymentID).Return(nil).Once()

		err := uc.DeletePayment(context.Background(), paymentID)

		assert.NoError(t, err)
		mockPaymentRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("Success - Status Reverts to Unpaid", func(t *testing.T) {
		// Skenario: Hanya ada 1x pembayaran DP.
		// Jika dihapus, maka total uang masuk jadi 0, status order harus "Unpaid".
		existingPayments := []domain.Payment{
			{ID: "pay-123", Amount: 500000},
		}

		mockPaymentRepo.On("GetByID", mock.Anything, paymentID).Return(mockPayment, nil).Once()
		mockOrderRepo.On("GetByID", mock.Anything, orderID).Return(mockOrder, nil).Once()
		mockPaymentRepo.On("GetByOrderID", mock.Anything, orderID).Return(existingPayments, nil).Once()

		// Ekspektasi: 500rb - 500rb = 0. Karena 0, status = Unpaid
		mockOrderRepo.On("UpdateStatus", mock.Anything, orderID, domain.OrderStatus(""), domain.PaymentStatusUnpaid).Return(nil).Once()
		mockPaymentRepo.On("Delete", mock.Anything, paymentID).Return(nil).Once()

		err := uc.DeletePayment(context.Background(), paymentID)

		assert.NoError(t, err)
		mockPaymentRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("Error - Payment Not Found", func(t *testing.T) {
		// Skenario: Data yang mau dihapus tidak ada di DB
		mockPaymentRepo.On("GetByID", mock.Anything, paymentID).Return(nil, domain.ErrNotFound).Once()

		err := uc.DeletePayment(context.Background(), paymentID)

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrNotFound, appErr.ErrType)

		// Pastikan method lain tidak dipanggil
		mockOrderRepo.AssertNotCalled(t, "GetByID")
		mockPaymentRepo.AssertNotCalled(t, "Delete")
	})

	t.Run("Error - Order Not Found", func(t *testing.T) {
		// Skenario: Payment ada, tapi Order terkait hilang secara aneh di DB
		mockPaymentRepo.On("GetByID", mock.Anything, paymentID).Return(mockPayment, nil).Once()
		mockOrderRepo.On("GetByID", mock.Anything, orderID).Return(nil, domain.ErrNotFound).Once()

		err := uc.DeletePayment(context.Background(), paymentID)

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)
		assert.Contains(t, appErr.Message, "Order terkait tidak ditemukan")
	})
}
