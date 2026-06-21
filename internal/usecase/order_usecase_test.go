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

func setupOrderTest() (*mocks.OrderRepository, *mocks.CustomerRepository, *mocks.UserRepository, *mocks.ProductRepository, *mocks.PaymentRepository, *mocks.TransactionManager, domain.OrderUsecase) {
	mockOrderRepo := new(mocks.OrderRepository)
	mockCustomerRepo := new(mocks.CustomerRepository)
	mockUserRepo := new(mocks.UserRepository)
	mockProductRepo := new(mocks.ProductRepository)
	mockPaymentRepo := new(mocks.PaymentRepository)
	mockTxManager := new(mocks.TransactionManager)

	uc := usecase.NewOrderUsecase(mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, mockPaymentRepo, mockTxManager, time.Second*2)

	return mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, mockPaymentRepo, mockTxManager, uc
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
		// Pastikan setupOrderTest kamu me-return mockTxManager juga.
		// Asumsi urutan return: OrderRepo, CustomerRepo, UserRepo, ProductRepo, PaymentRepo, TxManager, Usecase
		mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, _, mockTxManager, uc := setupOrderTest()

		input := baseInput
		input.OrderStatus = domain.OrderStatusQuotation // Set eksplisit sebagai Quotation

		// 1. Mock Customer & User exist
		mockCustomerRepo.On("GetByID", mock.Anything, input.CustomerID).Return(&domain.Customer{ID: input.CustomerID}, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, input.SalesID).Return(&domain.User{ID: input.SalesID}, nil).Once()

		// 2. Mock Product 1 & 2
		mockProductRepo.On("GetByID", mock.Anything, "prod-1").Return(&domain.Product{ID: "prod-1", BasePrice: 50000}, nil).Once()
		mockProductRepo.On("GetByID", mock.Anything, "prod-2").Return(&domain.Product{ID: "prod-2", BasePrice: 10000}, nil).Once()

		// =========================================================
		// 🚨 PERBAIKAN: MOCK TRANSACTION MANAGER
		// =========================================================
		mockTxManager.On("RunInTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)
				fn := args.Get(1).(func(context.Context) error)

				// Eksekusi fungsi closure-nya agar mock Create di bawah ikut berjalan!
				err := fn(ctx)
				assert.NoError(t, err)
			}).
			Return(nil).Once()
		// =========================================================

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
		mockTxManager.AssertExpectations(t) // Pastikan transaksi dipanggil
	})

	t.Run("Success - Create as Pending (Order Langsung)", func(t *testing.T) {
		mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, _, mockTxManager, uc := setupOrderTest()

		input := baseInput
		input.OrderStatus = "" // Kosong (Simulasi jika dari frontend tidak ngirim status)

		// 1. Mock Customer & User exist
		mockCustomerRepo.On("GetByID", mock.Anything, input.CustomerID).Return(&domain.Customer{ID: input.CustomerID}, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, input.SalesID).Return(&domain.User{ID: input.SalesID}, nil).Once()

		// 2. Mock Product 1 & 2
		mockProductRepo.On("GetByID", mock.Anything, "prod-1").Return(&domain.Product{ID: "prod-1", BasePrice: 50000}, nil).Once()
		mockProductRepo.On("GetByID", mock.Anything, "prod-2").Return(&domain.Product{ID: "prod-2", BasePrice: 10000}, nil).Once()

		// 🚨 PERBAIKAN: MOCK TRANSACTION MANAGER
		mockTxManager.On("RunInTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)
				fn := args.Get(1).(func(context.Context) error)
				err := fn(ctx)
				assert.NoError(t, err)
			}).
			Return(nil).Once()

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
		mockTxManager.AssertExpectations(t)
	})

	t.Run("Error - Customer Not Found", func(t *testing.T) {
		_, mockCustomerRepo, _, _, _, mockTxManager, uc := setupOrderTest()

		input := baseInput
		mockCustomerRepo.On("GetByID", mock.Anything, input.CustomerID).Return(nil, domain.ErrNotFound).Once()

		order, err := uc.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, order)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, "Customer tidak ditemukan", appErr.Message)

		// TxManager tidak boleh dipanggil jika validasi awal (customer) sudah gagal
		mockTxManager.AssertNotCalled(t, "RunInTransaction")
	})
}

func TestOrderUsecase_GetOrder(t *testing.T) {
	mockOrderRepo, _, _, _, _, _, uc := setupOrderTest()
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
	mockOrderRepo, _, _, _, _, _, uc := setupOrderTest()

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
	mockOrderRepo, _, _, _, _, _, uc := setupOrderTest()
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
	mockOrderRepo, _, _, _, _, _, uc := setupOrderTest()
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
	// Pastikan mockTxManager tertangkap dari setupOrderTest()
	mockOrderRepo, _, _, _, _, mockTxManager, uc := setupOrderTest()
	mockID := "ord-123"

	// Helper untuk mengeksekusi closure di dalam transaksi
	mockTransaction := func() {
		mockTxManager.ExpectedCalls = nil
		mockTxManager.On("RunInTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Return(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()
	}

	t.Run("Success - Quotation to Pending", func(t *testing.T) {
		mockOrderRepo.ExpectedCalls = nil
		mockTransaction()

		// Kondisi Awal: Order masih Quotation
		mockOrder := &domain.Order{
			ID:          mockID,
			OrderStatus: domain.OrderStatusQuotation,
		}

		// Ekspektasi: Gembok Order, validasi sukses, lalu update
		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, mockID).Return(mockOrder, nil).Once()
		mockOrderRepo.On("UpdateStatus", mock.Anything, mockID, domain.OrderStatusPending, domain.PaymentStatus("")).Return(nil).Once()

		err := uc.UpdateOrderStatus(context.Background(), mockID, domain.OrderStatusPending)

		assert.NoError(t, err)
		mockOrderRepo.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
	})

	t.Run("Success - Pending to Production", func(t *testing.T) {
		mockOrderRepo.ExpectedCalls = nil
		mockTransaction()

		// Kondisi Awal: Order Pending dan SUDAH DP (Syarat Mutlak State Machine)
		mockOrder := &domain.Order{
			ID:            mockID,
			OrderStatus:   domain.OrderStatusPending,
			PaymentStatus: domain.PaymentStatusPartial,
		}

		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, mockID).Return(mockOrder, nil).Once()
		mockOrderRepo.On("UpdateStatus", mock.Anything, mockID, domain.OrderStatusProduction, domain.PaymentStatus("")).Return(nil).Once()

		err := uc.UpdateOrderStatus(context.Background(), mockID, domain.OrderStatusProduction)

		assert.NoError(t, err)
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("Error - State Machine Rejected (Pending to Production but Unpaid)", func(t *testing.T) {
		mockOrderRepo.ExpectedCalls = nil
		mockTransaction()

		// Kondisi Awal: Order Pending, tapi BELUM BAYAR (Akan ditolak Satpam Pabrik)
		mockOrder := &domain.Order{
			ID:            mockID,
			OrderStatus:   domain.OrderStatusPending,
			PaymentStatus: domain.PaymentStatusUnpaid,
		}

		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, mockID).Return(mockOrder, nil).Once()

		err := uc.UpdateOrderStatus(context.Background(), mockID, domain.OrderStatusProduction)

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrConflict, appErr.ErrType)
		assert.Contains(t, appErr.Message, "belum ada pembayaran")

		// Pastikan TIDAK ADA proses penyimpanan ke database
		mockOrderRepo.AssertNotCalled(t, "UpdateStatus")
	})

	t.Run("Error - Order Not Found", func(t *testing.T) {
		mockOrderRepo.ExpectedCalls = nil
		mockTransaction()

		// Skenario: Data order tidak ada di DB saat mau digembok
		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, mockID).Return(nil, domain.ErrNotFound).Once()

		err := uc.UpdateOrderStatus(context.Background(), mockID, domain.OrderStatusPending)

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrNotFound, appErr.ErrType)

		mockOrderRepo.AssertNotCalled(t, "UpdateStatus")
	})

	t.Run("Error - Invalid Status Parameter", func(t *testing.T) {
		mockOrderRepo.ExpectedCalls = nil
		mockTxManager.ExpectedCalls = nil

		// Input mentah langsung ngawur, harusnya gagal sebelum masuk transaksi
		err := uc.UpdateOrderStatus(context.Background(), mockID, "STATUS_NGAWUR")

		assert.Error(t, err)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)

		mockTxManager.AssertNotCalled(t, "RunInTransaction")
		mockOrderRepo.AssertNotCalled(t, "GetByIDForUpdate")
		mockOrderRepo.AssertNotCalled(t, "UpdateStatus")
	})
}

func TestOrderUsecase_UpdatePaymentStatus(t *testing.T) {
	// Asumsi Anda memiliki fungsi setupOrderTest() untuk inisialisasi mock repo & usecase
	// mockOrderRepo, _, _, uc := setupOrderTest()
	mockOrderRepo, _, _, _, _, _, uc := setupOrderTest()
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

// --- TAMBAHAN BARU: UNIT TEST ORDER ITEMS ---

func TestAddOrderItem(t *testing.T) {
	mockOrderRepo := new(mocks.OrderRepository)
	mockCustomerRepo := new(mocks.CustomerRepository)
	mockUserRepo := new(mocks.UserRepository)
	mockProductRepo := new(mocks.ProductRepository)
	mockPaymentRepo := new(mocks.PaymentRepository)
	mockTxManager := new(mocks.TransactionManager)

	u := usecase.NewOrderUsecase(mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, mockPaymentRepo, mockTxManager, time.Second*2)

	orderID := "order-123"
	productID := "prod-123"
	input := domain.OrderItemInput{
		ProductID: productID,
		Qty:       2,
		Price:     15000,
	}

	t.Run("Success", func(t *testing.T) {
		mockOrderRepo.ExpectedCalls = nil
		mockProductRepo.ExpectedCalls = nil
		mockPaymentRepo.ExpectedCalls = nil
		mockTxManager.ExpectedCalls = nil // Pastikan reset mock transaksi

		// 1. Validasi awal (Dijalankan di luar transaksi)
		mockOrderRepo.On("GetByID", mock.Anything, orderID).Return(&domain.Order{ID: orderID}, nil).Once()
		mockProductRepo.On("GetByID", mock.Anything, productID).Return(&domain.Product{ID: productID, BasePrice: 15000}, nil).Once()

		// =========================================================
		// 🚨 PERBAIKAN: MOCK TRANSACTION MANAGER
		// =========================================================
		mockTxManager.On("RunInTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Run(func(args mock.Arguments) {
				// Ambil parameter yang dikirim ke RunInTransaction
				ctx := args.Get(0).(context.Context)
				fn := args.Get(1).(func(context.Context) error)

				// Eksekusi fungsi closure-nya agar mock repo di dalamnya ikut berjalan!
				err := fn(ctx)
				assert.NoError(t, err)
			}).
			Return(nil).Once()
		// =========================================================

		// 2. Buat item (Akan tereksekusi berkat Run() di atas)
		mockOrderRepo.On("CreateItem", mock.Anything, mock.AnythingOfType("*domain.OrderItem")).Return(nil).Once()

		// 3. Masuk ke recalculateOrderTotal
		mockOrderRepo.On("GetByID", mock.Anything, orderID).Return(&domain.Order{
			ID: orderID,
			Items: []domain.OrderItem{
				{Price: 15000, Qty: 2}, // Subtotal & Total = 30000
			},
		}, nil).Once()

		mockPaymentRepo.On("GetByOrderID", mock.Anything, orderID).Return([]domain.Payment{}, nil).Once()

		// 4. Update order header dengan total dan status yang baru
		mockOrderRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(nil).Once()

		result, err := u.AddOrderItem(context.Background(), orderID, input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, float64(30000), result.Subtotal)
		assert.Equal(t, float64(30000), result.TotalAmount)
		assert.Equal(t, domain.PaymentStatusUnpaid, result.PaymentStatus)

		// Verifikasi semua mock terpanggil
		mockOrderRepo.AssertExpectations(t)
		mockProductRepo.AssertExpectations(t)
		mockPaymentRepo.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
	})

	t.Run("Error_OrderNotFound", func(t *testing.T) {
		mockOrderRepo.ExpectedCalls = nil
		mockTxManager.ExpectedCalls = nil

		mockOrderRepo.On("GetByID", mock.Anything, orderID).Return(nil, domain.ErrNotFound).Once()

		result, err := u.AddOrderItem(context.Background(), orderID, input)

		assert.Error(t, err)
		assert.Nil(t, result)

		mockOrderRepo.AssertExpectations(t)

		// TxManager tidak boleh terpanggil karena validasi awal gagal
		mockTxManager.AssertNotCalled(t, "RunInTransaction")
	})
}

func TestUpdateOrderItem(t *testing.T) {
	mockOrderRepo := new(mocks.OrderRepository)
	mockCustomerRepo := new(mocks.CustomerRepository)
	mockUserRepo := new(mocks.UserRepository)
	mockProductRepo := new(mocks.ProductRepository)
	mockPaymentRepo := new(mocks.PaymentRepository)
	mockTxManager := new(mocks.TransactionManager)

	u := usecase.NewOrderUsecase(mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, mockPaymentRepo, mockTxManager, time.Second*2)

	orderID := "order-123"
	itemID := "item-123"
	productID := "prod-123"
	input := domain.OrderItemInput{
		ProductID: productID,
		Qty:       5, // Qty diubah jadi 5
		Price:     0, // Misal harga dikosongkan agar pakai BasePrice product
	}

	t.Run("Success", func(t *testing.T) {
		mockOrderRepo.ExpectedCalls = nil
		mockProductRepo.ExpectedCalls = nil
		mockPaymentRepo.ExpectedCalls = nil // Reset mock payment
		mockTxManager.ExpectedCalls = nil   // Reset mock transaction

		existingItem := &domain.OrderItem{ID: itemID, OrderID: orderID, ProductID: productID, Qty: 2, Price: 10000}

		// 1. Ambil data item & produk (Dijalankan di LUAR transaksi)
		mockOrderRepo.On("GetItemByID", mock.Anything, orderID, itemID).Return(existingItem, nil).Once()
		mockProductRepo.On("GetByID", mock.Anything, productID).Return(&domain.Product{ID: productID, BasePrice: 20000}, nil).Once()

		// =========================================================
		// 🚨 PERBAIKAN: MOCK TRANSACTION MANAGER
		// =========================================================
		mockTxManager.On("RunInTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)
				fn := args.Get(1).(func(context.Context) error)

				// Eksekusi closure agar UpdateItem dan recalculateOrderTotal ikut berjalan
				err := fn(ctx)
				assert.NoError(t, err)
			}).
			Return(nil).Once()
		// =========================================================

		// 2. Update item (Dijalankan di DALAM transaksi)
		mockOrderRepo.On("UpdateItem", mock.Anything, mock.AnythingOfType("*domain.OrderItem")).Return(nil).Once()

		// 3. Masuk ke recalculateOrderTotal (Dijalankan di DALAM transaksi)
		mockOrderRepo.On("GetByID", mock.Anything, orderID).Return(&domain.Order{
			ID: orderID,
			Items: []domain.OrderItem{
				{Price: 20000, Qty: 5}, // Subtotal = 100000
			},
		}, nil).Once()

		// Skenario Partial: Customer sudah DP 40.000 (Tagihan 100.000)
		mockPaymentRepo.On("GetByOrderID", mock.Anything, orderID).Return([]domain.Payment{
			{Amount: 40000},
		}, nil).Once()

		mockOrderRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(nil).Once()

		result, err := u.UpdateOrderItem(context.Background(), orderID, itemID, input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, float64(100000), result.Subtotal)                  // 20000 * 5
		assert.Equal(t, domain.PaymentStatusPartial, result.PaymentStatus) // Pastikan otomatis jadi Partial

		// Verifikasi semua mock terpanggil sesuai urutan
		mockOrderRepo.AssertExpectations(t)
		mockProductRepo.AssertExpectations(t)
		mockPaymentRepo.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
	})
}

func TestDeleteOrderItem(t *testing.T) {
	mockOrderRepo := new(mocks.OrderRepository)
	mockCustomerRepo := new(mocks.CustomerRepository)
	mockUserRepo := new(mocks.UserRepository)
	mockProductRepo := new(mocks.ProductRepository)
	mockPaymentRepo := new(mocks.PaymentRepository)
	mockTxManager := new(mocks.TransactionManager)

	u := usecase.NewOrderUsecase(mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, mockPaymentRepo, mockTxManager, time.Second*2)

	orderID := "order-123"
	itemID := "item-123"

	t.Run("Success", func(t *testing.T) {
		mockOrderRepo.ExpectedCalls = nil
		mockPaymentRepo.ExpectedCalls = nil // Reset mock payment
		mockTxManager.ExpectedCalls = nil   // Reset mock transaction

		// =========================================================
		// 🚨 PERBAIKAN: MOCK TRANSACTION MANAGER (Di urutan paling atas)
		// Karena fungsi DeleteOrderItem langsung memulai transaksi.
		// =========================================================
		mockTxManager.On("RunInTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)
				fn := args.Get(1).(func(context.Context) error)

				// Eksekusi closure agar DeleteItem dan recalculateOrderTotal berjalan
				err := fn(ctx)
				assert.NoError(t, err)
			}).
			Return(nil).Once()
		// =========================================================

		// 1. Hapus Item (Dijalankan di DALAM transaksi)
		mockOrderRepo.On("DeleteItem", mock.Anything, orderID, itemID).Return(nil).Once()

		// 2. Masuk ke recalculateOrderTotal (Dijalankan di DALAM transaksi)
		mockOrderRepo.On("GetByID", mock.Anything, orderID).Return(&domain.Order{
			ID:           orderID,
			ShippingCost: 15000,                // Misal ada ongkir 15rb
			Items:        []domain.OrderItem{}, // Kosong karena sudah dihapus
		}, nil).Once()

		// Skenario Paid: Customer sebelumnya udah bayar 15000. Maka status harus lunas otomatis.
		mockPaymentRepo.On("GetByOrderID", mock.Anything, orderID).Return([]domain.Payment{
			{Amount: 15000},
		}, nil).Once()

		// 3. Update order header (Dijalankan di DALAM transaksi)
		mockOrderRepo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Order")).Return(nil).Once()

		result, err := u.DeleteOrderItem(context.Background(), orderID, itemID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, float64(0), result.Subtotal)                    // Subtotal barang 0
		assert.Equal(t, float64(15000), result.TotalAmount)             // Total akhir sisa ongkir saja
		assert.Equal(t, domain.PaymentStatusPaid, result.PaymentStatus) // Pastikan otomatis berubah Paid

		// Verifikasi semua mock terpanggil
		mockOrderRepo.AssertExpectations(t)
		mockPaymentRepo.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
	})
}
