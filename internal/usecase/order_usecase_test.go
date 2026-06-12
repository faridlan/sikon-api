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

func setupOrderTest() (*mocks.OrderRepository, *mocks.CustomerRepository, *mocks.UserRepository, *mocks.ProductRepository, domain.OrderUsecase) {
	mockOrderRepo := new(mocks.OrderRepository)
	mockCustomerRepo := new(mocks.CustomerRepository)
	mockUserRepo := new(mocks.UserRepository)
	mockProductRepo := new(mocks.ProductRepository)

	uc := usecase.NewOrderUsecase(mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, time.Second*2)

	return mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, uc
}

func TestOrderUsecase_CreateOrder(t *testing.T) {
	validUntil := time.Now().AddDate(0, 0, 7) // Penawaran berlaku 7 hari

	// Base Input: Kita buat template input dasar yang akan dipakai di semua test case
	baseInput := domain.OrderCreateInput{
		CustomerID:      "cust-123",
		SalesID:         "user-123",
		ShippingCost:    50000,
		ValidUntil:      &validUntil,
		TermsConditions: "DP Minimal 50%",
		Items: []domain.OrderItemInput{
			{ProductID: "prod-1", Qty: 2, Price: 0},     // Price 0 -> harus fallback ke BasePrice (50000 * 2 = 100000)
			{ProductID: "prod-2", Qty: 1, Price: 15000}, // Price > 0 -> pakai price ini (15000 * 1 = 15000)
			// Total Harga Barang (Subtotal) = 115000
		},
	}

	t.Run("Success - Create as Quotation (Surat Penawaran)", func(t *testing.T) {
		mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, uc := setupOrderTest()

		input := baseInput
		input.OrderStatus = domain.OrderStatusQuotation // Set eksplisit sebagai Quotation

		// 1. Mock Customer & User exist
		mockCustomerRepo.On("GetByID", mock.Anything, input.CustomerID).Return(&domain.Customer{ID: input.CustomerID}, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, input.SalesID).Return(&domain.User{ID: input.SalesID}, nil).Once()

		// 2. Mock Product 1 & 2
		mockProductRepo.On("GetByID", mock.Anything, "prod-1").Return(&domain.Product{ID: "prod-1", BasePrice: 50000}, nil).Once()
		mockProductRepo.On("GetByID", mock.Anything, "prod-2").Return(&domain.Product{ID: "prod-2", BasePrice: 10000}, nil).Once()

		// 3. Kalkulasi ekspektasi: Subtotal & Grand Total
		mockOrderRepo.On("Create", mock.Anything, mock.MatchedBy(func(o *domain.Order) bool {
			return o.CustomerID == input.CustomerID &&
				o.Subtotal == 115000 && // Harga barang saja
				o.TotalAmount == 165000 && // Subtotal + Shipping Cost (115000 + 50000)
				o.DiscountAmount == 0 &&
				o.TaxPpn == 0 &&
				o.TaxPph == 0 &&
				len(o.Items) == 2 &&
				o.OrderStatus == domain.OrderStatusQuotation && // Validasi Status
				o.PaymentStatus == domain.PaymentStatusUnpaid &&
				o.TermsConditions == input.TermsConditions &&
				o.ValidUntil == input.ValidUntil
		})).Return(nil).Once()

		order, err := uc.CreateOrder(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, domain.OrderStatusQuotation, order.OrderStatus)
		mockCustomerRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockProductRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("Success - Create as Pending (Order Langsung)", func(t *testing.T) {
		mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, uc := setupOrderTest()

		input := baseInput
		input.OrderStatus = "" // Kosong (Simulasi jika Modal 1 frontend tidak ngirim status)

		// 1. Mock Customer & User exist
		mockCustomerRepo.On("GetByID", mock.Anything, input.CustomerID).Return(&domain.Customer{ID: input.CustomerID}, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, input.SalesID).Return(&domain.User{ID: input.SalesID}, nil).Once()

		// 2. Mock Product 1 & 2
		mockProductRepo.On("GetByID", mock.Anything, "prod-1").Return(&domain.Product{ID: "prod-1", BasePrice: 50000}, nil).Once()
		mockProductRepo.On("GetByID", mock.Anything, "prod-2").Return(&domain.Product{ID: "prod-2", BasePrice: 10000}, nil).Once()

		// 3. Kalkulasi ekspektasi: Status harus Pending
		mockOrderRepo.On("Create", mock.Anything, mock.MatchedBy(func(o *domain.Order) bool {
			return o.Subtotal == 115000 &&
				o.TotalAmount == 165000 &&
				o.OrderStatus == domain.OrderStatusPending // Validasi Status harus PENDING
		})).Return(nil).Once()

		order, err := uc.CreateOrder(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, domain.OrderStatusPending, order.OrderStatus) // Pastikan return-nya juga Pending
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("Error - Customer Not Found", func(t *testing.T) {
		_, mockCustomerRepo, _, _, uc := setupOrderTest()

		input := baseInput
		mockCustomerRepo.On("GetByID", mock.Anything, input.CustomerID).Return(nil, domain.ErrNotFound).Once()

		order, err := uc.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, order)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, "Customer tidak ditemukan", appErr.Message)
	})
}

func TestOrderUsecase_GetOrder(t *testing.T) {
	mockOrderRepo, _, _, _, uc := setupOrderTest()
	mockID := "ord-123"

	t.Run("Success", func(t *testing.T) {
		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(&domain.Order{ID: mockID}, nil).Once()

		result, err := uc.GetOrder(context.Background(), mockID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, mockID, result.ID)
	})

	t.Run("Error - Not Found", func(t *testing.T) {
		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.GetOrder(context.Background(), mockID)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestOrderUsecase_ListOrders(t *testing.T) {
	mockOrderRepo, _, _, _, uc := setupOrderTest()

	// 1. Siapkan mock input
	query := domain.PaginationQuery{Page: 1, Limit: 10}

	// Tambahkan mock filter (boleh kosong atau diisi sebagai representasi request FE)
	filter := domain.OrderFilter{
		OrderStatus: "pending",
	}

	t.Run("Success", func(t *testing.T) {
		mockOrders := []domain.Order{{ID: "1"}, {ID: "2"}}

		// Perhatikan penambahan parameter 'filter' pada mock.On()
		mockOrderRepo.On("Fetch", mock.Anything, filter, 10, 0).
			Return(mockOrders, int64(2), nil).Once()

		// Panggil usecase dengan menyertakan parameter 'filter'
		orders, meta, err := uc.ListOrders(context.Background(), filter, query)

		assert.NoError(t, err)
		assert.Len(t, orders, 2)
		assert.Equal(t, 1, meta.TotalPages)
		assert.Equal(t, int64(2), meta.TotalItems)
		assert.Equal(t, 10, meta.Limit)
	})

	t.Run("Error_From_Repository", func(t *testing.T) {
		// Mock ketika repository mengembalikan error (misal koneksi database terputus)
		expectedErr := errors.New("database error")

		mockOrderRepo.On("Fetch", mock.Anything, filter, 10, 0).
			Return(nil, int64(0), expectedErr).Once()

		orders, meta, err := uc.ListOrders(context.Background(), filter, query)

		// Verifikasi bahwa usecase meneruskan error dengan benar
		assert.Error(t, err)
		assert.Nil(t, orders)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 0, meta.TotalPages) // Meta harus kosong jika error
	})
}

func TestOrderUsecase_UpdateOrder(t *testing.T) {
	mockOrderRepo, _, _, _, uc := setupOrderTest()
	mockID := "ord-123"
	validUntilUpdate := time.Now().AddDate(0, 0, 14)

	input := domain.OrderUpdateInput{
		ShippingCost:    15000,
		CourierName:     "JNE",
		ValidUntil:      &validUntilUpdate,
		TermsConditions: "Pembayaran Lunas di awal",
	}

	t.Run("Success", func(t *testing.T) {
		// Simulasikan order lama yang ada di DB.
		// Kita taruh Subtotal 100rb, agar test bisa membuktikan bahwa Grand Total
		// benar-benar terupdate menjadi (Subtotal + ShippingCost baru).
		existingOrder := &domain.Order{
			ID:             mockID,
			ShippingCost:   0,
			Subtotal:       100000,
			DiscountAmount: 0,
			TaxPpn:         0,
			TaxPph:         0,
			TotalAmount:    100000, // Total lama (sebelum ada ongkir)
		}
		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(existingOrder, nil).Once()

		mockOrderRepo.On("Update", mock.Anything, mock.MatchedBy(func(o *domain.Order) bool {
			return o.ShippingCost == 15000 &&
				o.TotalAmount == 115000 && // Subtotal (100.000) + Ongkir Baru (15.000)
				o.CourierName == "JNE" &&
				o.TermsConditions == input.TermsConditions &&
				o.ValidUntil == input.ValidUntil
		})).Return(nil).Once()

		order, err := uc.UpdateOrder(context.Background(), mockID, input)

		assert.NoError(t, err)
		assert.NotNil(t, order)
	})
}

func TestOrderUsecase_DeleteOrder(t *testing.T) {
	mockOrderRepo, _, _, _, uc := setupOrderTest()
	mockID := "ord-123"
	existingOrder := &domain.Order{ID: mockID, ShippingCost: 0}

	t.Run("Success", func(t *testing.T) {
		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(existingOrder, nil).Once()
		mockOrderRepo.On("Delete", mock.Anything, mockID).Return(nil).Once()
		err := uc.DeleteOrder(context.Background(), mockID)
		assert.NoError(t, err)
	})
}

func TestOrderUsecase_UpdateOrderStatus(t *testing.T) {
	mockOrderRepo, _, _, _, uc := setupOrderTest()
	mockID := "ord-123"

	t.Run("Success - Valid Status", func(t *testing.T) {
		mockOrderRepo.On("UpdateStatus", mock.Anything, mockID, domain.OrderStatusProduction, domain.PaymentStatus("")).Return(nil).Once()

		err := uc.UpdateOrderStatus(context.Background(), mockID, domain.OrderStatusProduction)

		assert.NoError(t, err)
	})

	t.Run("Success - Quotation Status", func(t *testing.T) {
		mockOrderRepo.On("UpdateStatus", mock.Anything, mockID, domain.OrderStatusQuotation, domain.PaymentStatus("")).Return(nil).Once()

		err := uc.UpdateOrderStatus(context.Background(), mockID, domain.OrderStatusQuotation)

		assert.NoError(t, err)
	})

	t.Run("Error - Invalid Status", func(t *testing.T) {
		err := uc.UpdateOrderStatus(context.Background(), mockID, "STATUS_NGAWUR")

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)

		// Pastikan repo tidak dipanggil jika status invalid
		mockOrderRepo.AssertNotCalled(t, "UpdateStatus")
	})
}

func TestOrderUsecase_UpdatePaymentStatus(t *testing.T) {
	// Asumsi Anda memiliki fungsi setupOrderTest() untuk inisialisasi mock repo & usecase
	// mockOrderRepo, _, _, uc := setupOrderTest()
	mockOrderRepo, _, _, _, uc := setupOrderTest()
	mockID := "order-123"

	t.Run("Success", func(t *testing.T) {
		// Ekspektasi: Repo UpdateStatus dipanggil dengan OrderStatus kosong ("")
		// dan PaymentStatus valid ("paid")
		mockOrderRepo.On("UpdateStatus", mock.Anything, mockID, domain.OrderStatus(""), domain.PaymentStatusPaid).Return(nil).Once()

		err := uc.UpdatePaymentStatus(context.Background(), mockID, domain.PaymentStatusPaid)

		// Assertions
		assert.NoError(t, err)
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("Error_InvalidStatus", func(t *testing.T) {
		// Kita paksa mengirim status yang tidak ada di map validasi
		invalidStatus := domain.PaymentStatus("ngutang_dulu")

		err := uc.UpdatePaymentStatus(context.Background(), mockID, invalidStatus)

		// Assertions: Harus error dan repo TIDAK BOLEH dipanggil
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Status pembayaran tidak valid")
		mockOrderRepo.AssertNotCalled(t, "UpdateStatus")
	})

	t.Run("Error_FromRepository", func(t *testing.T) {
		repoError := errors.New("database connection lost")

		// Ekspektasi: Repo mengembalikan error
		mockOrderRepo.On("UpdateStatus", mock.Anything, mockID, domain.OrderStatus(""), domain.PaymentStatusPartial).Return(repoError).Once()

		err := uc.UpdatePaymentStatus(context.Background(), mockID, domain.PaymentStatusPartial)

		// Assertions
		assert.Error(t, err)
		assert.Equal(t, repoError, err)
		mockOrderRepo.AssertExpectations(t)
	})
}
