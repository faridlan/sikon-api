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
func setupPaymentTest() (*mocks.PaymentRepository, *mocks.OrderRepository, *mocks.BankAccountRepository, domain.PaymentUsecase) {
	mockPaymentRepo := new(mocks.PaymentRepository)
	mockOrderRepo := new(mocks.OrderRepository)
	mockBankAccountRepo := new(mocks.BankAccountRepository)

	uc := usecase.NewPaymentUsecase(mockPaymentRepo, mockOrderRepo, mockBankAccountRepo, time.Second*2)

	return mockPaymentRepo, mockOrderRepo, mockBankAccountRepo, uc
}

func TestPaymentUsecase_ProcessPayment(t *testing.T) {
	mockPaymentRepo, mockOrderRepo, mockBankRepo, uc := setupPaymentTest()

	input := domain.PaymentCreateInput{
		OrderID:         "ord-123",
		BankAccountID:   "bank-123",
		Amount:          500000,
		ReferenceNumber: "TRX-001",
		PaymentType:     domain.PaymentTypeDP,
	}

	mockOrder := &domain.Order{
		ID:           "ord-123",
		TotalAmount:  1000000,
		ShippingCost: 100000,
	} // GrandTotal = 1.100.000

	t.Run("Success - Partial Payment (DP)", func(t *testing.T) {
		// 1. Order Ditemukan
		mockOrderRepo.On("GetByID", mock.Anything, input.OrderID).Return(mockOrder, nil).Once()
		// 2. Bank Ditemukan
		mockBankRepo.On("GetByID", mock.Anything, input.BankAccountID).Return(&domain.BankAccount{ID: "bank-123"}, nil).Once()
		// 3. Belum ada pembayaran sebelumnya (totalPaid = 0)
		mockPaymentRepo.On("GetByOrderID", mock.Anything, input.OrderID).Return([]domain.Payment{}, nil).Once()

		// 4. Simpan Payment
		mockPaymentRepo.On("Create", mock.Anything, mock.MatchedBy(func(p *domain.Payment) bool {
			return p.Amount == input.Amount && p.ReferenceNumber == input.ReferenceNumber
		})).Return(nil).Once()

		// 5. Update Status Order -> Partial (Karena 500rb < 1.1jt)
		mockOrderRepo.On("UpdateStatus", mock.Anything, mockOrder.ID, domain.OrderStatus(""), domain.PaymentStatusPartial).Return(nil).Once()

		payment, err := uc.ProcessPayment(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, payment)
		mockOrderRepo.AssertExpectations(t)
		mockBankRepo.AssertExpectations(t)
		mockPaymentRepo.AssertExpectations(t)
	})

	t.Run("Success - Full Payment (Paid)", func(t *testing.T) {
		inputPelunasan := input
		inputPelunasan.Amount = 600000 // Uang yang dibayar pelunasan
		inputPelunasan.PaymentType = domain.PaymentTypeSettlement

		// Simulasi sudah pernah bayar DP 500.000
		existingPayments := []domain.Payment{
			{Amount: 500000},
		}

		mockOrderRepo.On("GetByID", mock.Anything, input.OrderID).Return(mockOrder, nil).Once()
		mockBankRepo.On("GetByID", mock.Anything, input.BankAccountID).Return(&domain.BankAccount{}, nil).Once()
		mockPaymentRepo.On("GetByOrderID", mock.Anything, input.OrderID).Return(existingPayments, nil).Once()

		mockPaymentRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()

		// Total Paid sekarang: 500k (DP) + 600k (Pelunasan) = 1.1jt (== GrandTotal) -> Status jadi PAID
		mockOrderRepo.On("UpdateStatus", mock.Anything, mockOrder.ID, domain.OrderStatus(""), domain.PaymentStatusPaid).Return(nil).Once()

		payment, err := uc.ProcessPayment(context.Background(), inputPelunasan)

		assert.NoError(t, err)
		assert.NotNil(t, payment)
	})

	t.Run("Error - Already Fully Paid", func(t *testing.T) {
		// Simulasi tagihan sudah lunas sebelumnya
		existingPayments := []domain.Payment{
			{Amount: 1100000},
		}

		mockOrderRepo.On("GetByID", mock.Anything, input.OrderID).Return(mockOrder, nil).Once()
		mockBankRepo.On("GetByID", mock.Anything, input.BankAccountID).Return(&domain.BankAccount{}, nil).Once()
		mockPaymentRepo.On("GetByOrderID", mock.Anything, input.OrderID).Return(existingPayments, nil).Once()

		payment, err := uc.ProcessPayment(context.Background(), input)

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrConflict, appErr.ErrType)
		assert.Equal(t, "Pesanan ini sudah lunas sepenuhnya", appErr.Message)
		assert.Nil(t, payment)

		// Pastikan Create dan UpdateStatus tidak dipanggil!
		mockPaymentRepo.AssertNotCalled(t, "Create")
		mockOrderRepo.AssertNotCalled(t, "UpdateStatus")
	})

	t.Run("Error - Order Not Found", func(t *testing.T) {
		mockOrderRepo.On("GetByID", mock.Anything, input.OrderID).Return(nil, domain.ErrNotFound).Once()

		payment, err := uc.ProcessPayment(context.Background(), input)

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)
		assert.Equal(t, "Order tidak ditemukan", appErr.Message)
		assert.Nil(t, payment)
	})
}

func TestPaymentUsecase_GetPayment(t *testing.T) {
	mockPaymentRepo, _, _, uc := setupPaymentTest()
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
	mockPaymentRepo, _, _, uc := setupPaymentTest()
	query := domain.PaginationQuery{Page: 1, Limit: 10}

	t.Run("Success", func(t *testing.T) {
		mockPayments := []domain.Payment{{ID: "1"}, {ID: "2"}}
		mockPaymentRepo.On("Fetch", mock.Anything, 10, 0).Return(mockPayments, int64(2), nil).Once()

		payments, meta, err := uc.ListPayments(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, payments, 2)
		assert.Equal(t, 1, meta.TotalPages)
	})
}

func TestPaymentUsecase_UpdatePayment(t *testing.T) {
	mockPaymentRepo, _, _, uc := setupPaymentTest()
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
	mockPaymentRepo, _, _, uc := setupPaymentTest()
	mockID := "pay-123"

	t.Run("Success", func(t *testing.T) {
		existingPayment := &domain.Payment{ID: mockID, ReferenceNumber: "TRX-LAMA"}
		mockPaymentRepo.On("GetByID", mock.Anything, mockID).Return(existingPayment, nil).Once()
		mockPaymentRepo.On("Delete", mock.Anything, mockID).Return(nil).Once()
		err := uc.DeletePayment(context.Background(), mockID)
		assert.NoError(t, err)
	})
}
