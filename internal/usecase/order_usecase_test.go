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

func setupOrderTest() (*mocks.OrderRepository, *mocks.CustomerRepository, *mocks.UserRepository, *mocks.ProductRepository, *mocks.BatchPORepository, *mocks.PaymentRepository, *mocks.TransactionManager, *mocks.ProductMaterialUsecase, *mocks.WorkLogRepository, domain.OrderUsecase) {
	mockOrderRepo := new(mocks.OrderRepository)
	mockCustomerRepo := new(mocks.CustomerRepository)
	mockUserRepo := new(mocks.UserRepository)
	mockProductRepo := new(mocks.ProductRepository)
	mockBatchPORepo := new(mocks.BatchPORepository)
	mockPaymentRepo := new(mocks.PaymentRepository)
	mockTxManager := new(mocks.TransactionManager)
	mockProductMaterialUsecase := new(mocks.ProductMaterialUsecase)
	mockWorkLogRepo := new(mocks.WorkLogRepository)

	uc := usecase.NewOrderUsecase(mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, mockBatchPORepo, mockPaymentRepo, mockTxManager, mockProductMaterialUsecase, mockWorkLogRepo, time.Second*2)

	return mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, mockBatchPORepo, mockPaymentRepo, mockTxManager, mockProductMaterialUsecase, mockWorkLogRepo, uc
}

func TestOrderUsecase_CreateOrder(t *testing.T) {
	validUntil := time.Now().AddDate(0, 0, 7) // Penawaran berlaku 7 hari

	baseInput := domain.OrderCreateInput{
		BatchPoID:       "batch-123", // 🚨 Wajib Ada
		CustomerID:      "cust-123",
		SalesID:         "user-123",
		ShippingCost:    50000,
		ValidUntil:      &validUntil,
		TermsConditions: "DP Minimal 50%",
		Items: []domain.OrderItemInput{
			// 🌟 TAMBAHAN: Sisipkan CustomName di salah satu item untuk diuji
			{ProductID: "prod-1", CustomName: "Kemeja PDH PT ABC", Qty: 2, Price: 0},
			{ProductID: "prod-2", Qty: 1, Price: 15000},
		},
	}

	t.Run("Success - Create with Array Details (Produk Setelan)", func(t *testing.T) {
		// 🚨 Pastikan setupOrderTest me-return mockBatchPoRepo
		mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, mockBatchPoRepo, _, mockTxManager, _, _, uc := setupOrderTest()

		// Kita copy baseInput agar tidak merusak test lain
		input := baseInput
		input.OrderStatus = domain.OrderStatusQuotation

		// 👇 SETUP ARRAY DETAILS: Simulasi input spesifikasi setelan dari FE
		arrayDetails := []map[string]any{
			{
				"part":          "Kemeja",
				"material_name": "American Drill",
				"spec":          "Tebal",
			},
			{
				"part":          "Celana",
				"material_name": "American Drill",
				"spec":          "Kuat",
			},
		}
		// Suntikkan array tersebut ke item pertama
		input.Items[0].Details = arrayDetails
		input.Items[0].CustomName = "PDH ERT ABU (Setelan)"

		mockCustomerRepo.On("GetByID", mock.Anything, input.CustomerID).Return(&domain.Customer{ID: input.CustomerID}, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, input.SalesID).Return(&domain.User{ID: input.SalesID}, nil).Once()
		mockBatchPoRepo.On("GetByID", mock.Anything, input.BatchPoID).Return(&domain.BatchPO{ID: input.BatchPoID, Status: domain.BatchPOStatusActive}, nil).Once()

		// Asumsi baseInput memiliki 2 produk seperti pada test normal flow Anda
		mockProductRepo.On("GetByID", mock.Anything, input.Items[0].ProductID).Return(&domain.Product{ID: input.Items[0].ProductID, BasePrice: 50000}, nil).Once()
		mockProductRepo.On("GetByID", mock.Anything, input.Items[1].ProductID).Return(&domain.Product{ID: input.Items[1].ProductID, BasePrice: 10000}, nil).Once()

		mockTxManager.On("RunInTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)
				fn := args.Get(1).(func(context.Context) error)
				err := fn(ctx)
				assert.NoError(t, err)
			}).
			Return(nil).Once()

		// 🚨 MOCK ORDER REPO: Validasi ketat bahwa payload Array tidak rusak
		mockOrderRepo.On("Create", mock.Anything, mock.MatchedBy(func(o *domain.Order) bool {
			// Kita casting dulu any ke []map[string]any untuk pengecekan
			detailsArr, ok := o.Items[0].Details.([]map[string]any)

			return o.BatchPoID == input.BatchPoID &&
				o.CustomerID == input.CustomerID &&
				o.Subtotal == 115000 && // Mengikuti baseInput Anda
				o.TotalAmount == 165000 &&
				len(o.Items) == 2 &&
				o.Items[0].CustomName == "PDH ERT ABU (Setelan)" &&
				// 🌟 ASERSI ARRAY: Pastikan bisa di-casting dan isinya utuh 2 item
				ok && len(detailsArr) == 2 &&
				detailsArr[0]["part"] == "Kemeja" &&
				detailsArr[1]["part"] == "Celana"
		})).Return(nil).Once()

		order, err := uc.CreateOrder(context.Background(), input)

		assert.NoError(t, err)
		assert.NotNil(t, order)
		assert.Equal(t, domain.OrderStatusQuotation, order.OrderStatus)
		assert.Equal(t, "PDH ERT ABU (Setelan)", order.Items[0].CustomName)

		// 🌟 Ekstra Asersi pada hasil Return dari Usecase
		returnedDetails, ok := order.Items[0].Details.([]map[string]any)
		assert.True(t, ok, "Details harus tetap berupa Array pada response")
		assert.Equal(t, "Kemeja", returnedDetails[0]["part"])

		mockBatchPoRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("Error - Batch PO Closed (Ditolak Satpam)", func(t *testing.T) {
		_, mockCustomerRepo, mockUserRepo, _, mockBatchPoRepo, _, mockTxManager, _, _, uc := setupOrderTest()

		input := baseInput

		mockCustomerRepo.On("GetByID", mock.Anything, input.CustomerID).Return(&domain.Customer{ID: input.CustomerID}, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, input.SalesID).Return(&domain.User{ID: input.SalesID}, nil).Once()

		// 🚨 MOCK BATCH PO: Kembalikan PO yang statusnya Closed
		mockBatchPoRepo.On("GetByID", mock.Anything, input.BatchPoID).Return(&domain.BatchPO{ID: input.BatchPoID, Status: domain.BatchPOStatusClosed}, nil).Once()

		order, err := uc.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, order)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrForbidden, appErr.ErrType)
		assert.Contains(t, appErr.Message, "PO sudah ditutup")

		mockTxManager.AssertNotCalled(t, "RunInTransaction")
	})

	t.Run("Error - Customer Not Found", func(t *testing.T) {
		_, mockCustomerRepo, _, _, _, _, mockTxManager, _, _, uc := setupOrderTest()

		input := baseInput
		mockCustomerRepo.On("GetByID", mock.Anything, input.CustomerID).Return(nil, domain.ErrNotFound).Once()

		order, err := uc.CreateOrder(context.Background(), input)

		assert.Error(t, err)
		assert.Nil(t, order)
		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, "Customer tidak ditemukan", appErr.Message)

		mockTxManager.AssertNotCalled(t, "RunInTransaction")
	})
}

func TestOrderUsecase_GetOrder(t *testing.T) {
	mockOrderRepo, _, _, _, _, _, _, _, _, uc := setupOrderTest()
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
	query := domain.PaginationQuery{Page: 1, Limit: 10}

	t.Run("Success - Auto Set Active Batch PO When Filter Empty", func(t *testing.T) {
		mockOrderRepo, _, _, _, mockBatchPoRepo, _, _, _, _, uc := setupOrderTest()

		// Filter dari FE tanpa batch_po_id
		inputFilter := domain.OrderFilter{
			OrderStatus: "pending",
		}

		activePO := &domain.BatchPO{ID: "po-active-august"}

		// 1. Ekspektasi: Usecase akan mencari Active PO berdasarkan tanggal hari ini
		mockBatchPoRepo.On("GetActivePOByDate", mock.Anything, mock.AnythingOfType("time.Time")).
			Return(activePO, nil).Once()

		// 2. Ekspektasi: Filter yang masuk ke Repository.Fetch SUDAH terisi BatchPoID dari Active PO
		expectedFilter := inputFilter
		expectedFilter.BatchPoID = activePO.ID

		mockOrders := []domain.Order{{ID: "1", TotalQty: 10}, {ID: "2", TotalQty: 5}}
		mockOrderRepo.On("Fetch", mock.Anything, expectedFilter, 10, 0).
			Return(mockOrders, int64(2), nil).Once()

		orders, meta, err := uc.ListOrders(context.Background(), inputFilter, query)

		assert.NoError(t, err)
		assert.Len(t, orders, 2)
		assert.Equal(t, 1, meta.TotalPages)
		assert.Equal(t, int64(2), meta.TotalItems)
		assert.Equal(t, 10, meta.Limit)

		mockBatchPoRepo.AssertExpectations(t)
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("Success - Explicit Batch PO 'all' (Skip Active PO Lookup)", func(t *testing.T) {
		mockOrderRepo, _, _, _, mockBatchPoRepo, _, _, _, _, uc := setupOrderTest()

		// Filter dari FE mengirim batch_po_id = "all"
		explicitFilter := domain.OrderFilter{
			BatchPoID: "all",
		}

		mockOrders := []domain.Order{{ID: "1"}, {ID: "2"}}
		mockOrderRepo.On("Fetch", mock.Anything, explicitFilter, 10, 0).
			Return(mockOrders, int64(2), nil).Once()

		orders, meta, err := uc.ListOrders(context.Background(), explicitFilter, query)

		assert.NoError(t, err)
		assert.Len(t, orders, 2)
		assert.Equal(t, 1, meta.TotalPages)

		// Memastikan GetActivePOByDate TIDAK DIPANGGIL karena batch_po_id sudah diisi "all"
		mockBatchPoRepo.AssertNotCalled(t, "GetActivePOByDate")
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("Error_From_Repository", func(t *testing.T) {
		mockOrderRepo, _, _, _, mockBatchPoRepo, _, _, _, _, uc := setupOrderTest()

		filter := domain.OrderFilter{
			BatchPoID: "po-123", // Menggunakan PO ID eksplisit
		}
		expectedErr := errors.New("database error")

		mockOrderRepo.On("Fetch", mock.Anything, filter, 10, 0).
			Return(nil, int64(0), expectedErr).Once()

		orders, meta, err := uc.ListOrders(context.Background(), filter, query)

		assert.Error(t, err)
		assert.Nil(t, orders)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, 0, meta.TotalPages)

		mockBatchPoRepo.AssertNotCalled(t, "GetActivePOByDate")
		mockOrderRepo.AssertExpectations(t)
	})
}
func TestOrderUsecase_UpdateOrder(t *testing.T) {
	mockOrderRepo, _, _, _, _, _, _, _, _, uc := setupOrderTest()
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
	mockOrderRepo, _, _, _, _, _, _, _, _, uc := setupOrderTest()
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
	mockID := "ord-123"

	t.Run("Success - Quotation to Pending", func(t *testing.T) {
		// Pastikan mockBatchPoRepo ditangkap di sini
		mockOrderRepo, _, _, _, mockBatchPoRepo, _, mockTxManager, _, _, uc := setupOrderTest()

		mockTxManager.On("RunInTransaction", mock.Anything, mock.Anything).
			Return(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()

		mockOrder := &domain.Order{
			ID:            mockID,
			OrderStatus:   domain.OrderStatusQuotation,
			PaymentStatus: domain.PaymentStatusPartial,
			BatchPoID:     "po-lama",
		}

		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, mockID).Return(mockOrder, nil).Once()

		// 🚨 TAMBAHKAN MOCK INI: Karena Quotation -> Pending sekarang memicu pencarian PO aktif
		activePO := &domain.BatchPO{ID: "po-baru-agustus"}
		mockBatchPoRepo.On("GetActivePOByDate", mock.Anything, mock.Anything).Return(activePO, nil).Once()

		// 🆕 BARU: Fetch ulang Order dengan Items untuk hitung HPP Material (GetByIDForUpdate sengaja tidak preload Items)
		// Items kosong di sini -> HPPMaterialCost hasilnya 0, productMaterialUsecase tidak perlu di-mock sama sekali
		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(&domain.Order{ID: mockID, Items: []domain.OrderItem{}}, nil).Once()

		// Update ekspektasi: BatchPoID, ApprovedAt, dan HPP snapshot ikut ter-update
		mockOrderRepo.On("Update", mock.Anything, mock.MatchedBy(func(o *domain.Order) bool {
			return o.OrderStatus == domain.OrderStatusPending &&
				o.BatchPoID == "po-baru-agustus" &&
				o.ApprovedAt != nil &&
				o.HPPMaterialCost == 0 && // tidak ada item -> HPP 0
				o.HPPCalculatedAt != nil
		})).Return(nil).Once()

		err := uc.UpdateOrderStatus(context.Background(), mockID, domain.OrderStatusPending)

		assert.NoError(t, err)
		mockOrderRepo.AssertExpectations(t)
		mockBatchPoRepo.AssertExpectations(t) // Pastikan mock PO di-assert juga
	})

	t.Run("Success - Manual Approve (Quotation to Pending) & Auto Re-allocate", func(t *testing.T) {
		mockOrderRepo, _, _, _, mockBatchPoRepo, _, mockTxManager, _, _, uc := setupOrderTest()

		mockTxManager.On("RunInTransaction", mock.Anything, mock.Anything).
			Return(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()

		mockOrder := &domain.Order{
			ID:            mockID,
			OrderStatus:   domain.OrderStatusQuotation,
			PaymentStatus: domain.PaymentStatusPartial, // Syarat agar lolos ke Pending
			BatchPoID:     "po-lama",
		}

		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, mockID).Return(mockOrder, nil).Once()

		// MOCK: Proses Re-alokasi
		activePO := &domain.BatchPO{ID: "po-baru-agustus"}
		mockBatchPoRepo.On("GetActivePOByDate", mock.Anything, mock.Anything).Return(activePO, nil).Once()

		// 🆕 BARU: fetch ulang Order dengan Items untuk hitung HPP Material
		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(&domain.Order{ID: mockID, Items: []domain.OrderItem{}}, nil).Once()

		mockOrderRepo.On("Update", mock.Anything, mock.MatchedBy(func(o *domain.Order) bool {
			// Ekspektasikan order berubah menjadi Pending dan BatchPoID ter-update
			return o.OrderStatus == domain.OrderStatusPending &&
				o.BatchPoID == "po-baru-agustus" &&
				o.ApprovedAt != nil &&
				o.HPPCalculatedAt != nil
		})).Return(nil).Once()

		// 🚨 Panggil dengan target status Pending, sesuai aturan Satpam
		err := uc.UpdateOrderStatus(context.Background(), mockID, domain.OrderStatusPending)

		assert.NoError(t, err)
		mockOrderRepo.AssertExpectations(t)
		mockBatchPoRepo.AssertExpectations(t)
	})

	t.Run("Error - Quotation to Pending but Unpaid", func(t *testing.T) {
		mockOrderRepo, _, _, _, _, _, mockTxManager, _, _, uc := setupOrderTest()

		mockTxManager.On("RunInTransaction", mock.Anything, mock.Anything).
			Return(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()

		mockOrder := &domain.Order{
			ID:            mockID,
			OrderStatus:   domain.OrderStatusQuotation,
			PaymentStatus: domain.PaymentStatusUnpaid,
		}

		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, mockID).Return(mockOrder, nil).Once()

		err := uc.UpdateOrderStatus(context.Background(), mockID, domain.OrderStatusPending)

		assert.Error(t, err)
		mockOrderRepo.AssertNotCalled(t, "Update")
	})

	t.Run("Success - Pending to Production", func(t *testing.T) {
		mockOrderRepo, _, _, _, _, _, mockTxManager, _, _, uc := setupOrderTest()

		mockTxManager.On("RunInTransaction", mock.Anything, mock.Anything).
			Return(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()

		mockOrder := &domain.Order{
			ID:          mockID,
			OrderStatus: domain.OrderStatusPending,
		}

		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, mockID).Return(mockOrder, nil).Once()

		// Re-alokasi PO TIDAK dipanggil karena status sebelumnya Pending (bukan Quotation)
		mockOrderRepo.On("Update", mock.Anything, mock.MatchedBy(func(o *domain.Order) bool {
			return o.OrderStatus == domain.OrderStatusProduction
		})).Return(nil).Once()

		err := uc.UpdateOrderStatus(context.Background(), mockID, domain.OrderStatusProduction)

		assert.NoError(t, err)
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("Success - Ready to Completed", func(t *testing.T) {
		mockOrderRepo, _, _, _, _, _, mockTxManager, _, _, uc := setupOrderTest()

		mockTxManager.On("RunInTransaction", mock.Anything, mock.Anything).
			Return(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()

		mockOrder := &domain.Order{
			ID:            mockID,
			OrderStatus:   domain.OrderStatusReady,
			PaymentStatus: domain.PaymentStatusPaid,
		}

		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, mockID).Return(mockOrder, nil).Once()

		mockOrderRepo.On("Update", mock.Anything, mock.MatchedBy(func(o *domain.Order) bool {
			return o.OrderStatus == domain.OrderStatusCompleted
		})).Return(nil).Once()

		err := uc.UpdateOrderStatus(context.Background(), mockID, domain.OrderStatusCompleted)

		assert.NoError(t, err)
		mockOrderRepo.AssertExpectations(t)
	})

	t.Run("Error - Order Not Found", func(t *testing.T) {
		mockOrderRepo, _, _, _, _, _, mockTxManager, _, _, uc := setupOrderTest()

		mockTxManager.On("RunInTransaction", mock.Anything, mock.Anything).
			Return(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()

		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, mockID).Return(nil, domain.ErrNotFound).Once()

		err := uc.UpdateOrderStatus(context.Background(), mockID, domain.OrderStatusPending)

		assert.Error(t, err)
		mockOrderRepo.AssertNotCalled(t, "Update")
	})

	t.Run("Success - Approve Membekukan HPP Material Dari 2 Item", func(t *testing.T) {
		mockOrderRepo, _, _, _, mockBatchPoRepo, _, mockTxManager, mockProductMaterialUsecase, _, uc := setupOrderTest()

		mockTxManager.On("RunInTransaction", mock.Anything, mock.Anything).
			Return(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()

		mockOrder := &domain.Order{
			ID:            mockID,
			OrderStatus:   domain.OrderStatusQuotation,
			PaymentStatus: domain.PaymentStatusPartial,
		}
		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, mockID).Return(mockOrder, nil).Once()

		// Tidak ada Batch PO aktif hari ini -> BatchPoID order tidak berubah, tapi approve tetap jalan
		mockBatchPoRepo.On("GetActivePOByDate", mock.Anything, mock.Anything).Return(nil, domain.ErrNotFound).Once()

		// Order ini punya 2 item -> CalculateMaterialCost dipanggil 2x, masing-masing produk beda
		orderWithItems := &domain.Order{
			ID: mockID,
			Items: []domain.OrderItem{
				{ProductID: "prod-kaos-polo", Qty: 20},
				{ProductID: "prod-kemeja-pdh", Qty: 10},
			},
		}
		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(orderWithItems, nil).Once()

		mockProductMaterialUsecase.On("CalculateMaterialCost", mock.Anything, "prod-kaos-polo", 20).Return(float64(600000), nil).Once()
		mockProductMaterialUsecase.On("CalculateMaterialCost", mock.Anything, "prod-kemeja-pdh", 10).Return(float64(450000), nil).Once()

		// Total HPP Material harus 600.000 + 450.000 = 1.050.000, dan HPPCalculatedAt terisi
		mockOrderRepo.On("Update", mock.Anything, mock.MatchedBy(func(o *domain.Order) bool {
			return o.OrderStatus == domain.OrderStatusPending &&
				o.HPPMaterialCost == 1050000 &&
				o.HPPCalculatedAt != nil
		})).Return(nil).Once()

		err := uc.UpdateOrderStatus(context.Background(), mockID, domain.OrderStatusPending)

		assert.NoError(t, err)
		mockOrderRepo.AssertExpectations(t)
		mockProductMaterialUsecase.AssertExpectations(t)
	})

	t.Run("Error - CalculateMaterialCost Gagal Membatalkan Seluruh Approve", func(t *testing.T) {
		// Kalau hitung HPP gagal (misal resep produk korup), approve HARUS gagal juga -
		// jangan sampai Order pindah ke Pending dengan HPP yang salah/kosong diam-diam.
		mockOrderRepo, _, _, _, mockBatchPoRepo, _, mockTxManager, mockProductMaterialUsecase, _, uc := setupOrderTest()

		mockTxManager.On("RunInTransaction", mock.Anything, mock.Anything).
			Return(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()

		mockOrder := &domain.Order{
			ID:            mockID,
			OrderStatus:   domain.OrderStatusQuotation,
			PaymentStatus: domain.PaymentStatusPartial,
		}
		mockOrderRepo.On("GetByIDForUpdate", mock.Anything, mockID).Return(mockOrder, nil).Once()
		mockBatchPoRepo.On("GetActivePOByDate", mock.Anything, mock.Anything).Return(nil, domain.ErrNotFound).Once()

		orderWithItems := &domain.Order{
			ID:    mockID,
			Items: []domain.OrderItem{{ProductID: "prod-rusak", Qty: 5}},
		}
		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(orderWithItems, nil).Once()

		expectedErr := errors.New("resep produk tidak ditemukan")
		mockProductMaterialUsecase.On("CalculateMaterialCost", mock.Anything, "prod-rusak", 5).Return(float64(0), expectedErr).Once()

		err := uc.UpdateOrderStatus(context.Background(), mockID, domain.OrderStatusPending)

		assert.Error(t, err)
		mockOrderRepo.AssertNotCalled(t, "Update") // Order TIDAK BOLEH ke-update kalau HPP gagal dihitung
		mockProductMaterialUsecase.AssertExpectations(t)
	})
}

func TestOrderUsecase_GetOrderHPP(t *testing.T) {
	mockID := "ord-123"

	t.Run("Success - Gabungkan Material Beku dan Tenaga Kerja Live", func(t *testing.T) {
		mockOrderRepo, _, _, _, _, _, _, _, mockWorkLogRepo, uc := setupOrderTest()

		approvedAt := time.Now().Add(-24 * time.Hour)
		existingOrder := &domain.Order{
			ID:              mockID,
			HPPMaterialCost: 1050000, // sudah dibekukan saat approve
			HPPCalculatedAt: &approvedAt,
		}
		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(existingOrder, nil).Once()
		mockWorkLogRepo.On("GetTotalCostByOrder", mock.Anything, mockID).Return(float64(350000), nil).Once()

		result, err := uc.GetOrderHPP(context.Background(), mockID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, mockID, result.OrderID)
		assert.Equal(t, float64(1050000), result.MaterialCost)
		assert.Equal(t, float64(350000), result.LaborCost)
		assert.Equal(t, float64(1400000), result.TotalCost) // 1.050.000 + 350.000
		mockOrderRepo.AssertExpectations(t)
		mockWorkLogRepo.AssertExpectations(t)
	})

	t.Run("Success - Order Belum Pernah Disetujui, HPP Material Masih 0", func(t *testing.T) {
		mockOrderRepo, _, _, _, _, _, _, _, mockWorkLogRepo, uc := setupOrderTest()

		existingOrder := &domain.Order{ID: mockID} // Masih Quotation, HPPMaterialCost belum pernah diisi
		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(existingOrder, nil).Once()
		mockWorkLogRepo.On("GetTotalCostByOrder", mock.Anything, mockID).Return(float64(0), nil).Once()

		result, err := uc.GetOrderHPP(context.Background(), mockID)

		assert.NoError(t, err)
		assert.Equal(t, float64(0), result.MaterialCost)
		assert.Nil(t, result.MaterialCalculatedAt)
		assert.Equal(t, float64(0), result.TotalCost)
	})

	t.Run("Failed - Order Tidak Ditemukan", func(t *testing.T) {
		mockOrderRepo, _, _, _, _, _, _, _, mockWorkLogRepo, uc := setupOrderTest()

		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(nil, domain.ErrNotFound).Once()

		result, err := uc.GetOrderHPP(context.Background(), mockID)

		assert.ErrorIs(t, err, domain.ErrNotFound)
		assert.Nil(t, result)
		mockWorkLogRepo.AssertNotCalled(t, "GetTotalCostByOrder")
	})

	t.Run("Failed - WorkLogRepo Error", func(t *testing.T) {
		mockOrderRepo, _, _, _, _, _, _, _, mockWorkLogRepo, uc := setupOrderTest()

		existingOrder := &domain.Order{ID: mockID, HPPMaterialCost: 500000}
		mockOrderRepo.On("GetByID", mock.Anything, mockID).Return(existingOrder, nil).Once()

		expectedErr := errors.New("db error")
		mockWorkLogRepo.On("GetTotalCostByOrder", mock.Anything, mockID).Return(float64(0), expectedErr).Once()

		result, err := uc.GetOrderHPP(context.Background(), mockID)

		assert.Error(t, err)
		assert.Nil(t, result)
		mockOrderRepo.AssertExpectations(t)
		mockWorkLogRepo.AssertExpectations(t)
	})
}

func TestOrderUsecase_UpdatePaymentStatus(t *testing.T) {
	// Asumsi Anda memiliki fungsi setupOrderTest() untuk inisialisasi mock repo & usecase
	// mockOrderRepo, _, _, _, _, uc := setupOrderTest()
	mockOrderRepo, _, _, _, _, _, _, _, _, uc := setupOrderTest()
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
	mockBatchPORepo := new(mocks.BatchPORepository)
	mockProductMaterialUsecase := new(mocks.ProductMaterialUsecase)
	mockWorkLogRepo := new(mocks.WorkLogRepository)

	u := usecase.NewOrderUsecase(mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, mockBatchPORepo, mockPaymentRepo, mockTxManager, mockProductMaterialUsecase, mockWorkLogRepo, time.Second*2)

	orderID := "order-123"
	productID := "prod-123"
	input := domain.OrderItemInput{
		ProductID:  productID,
		CustomName: "Kemeja PDH Bank ERT",
		Qty:        2,
		Price:      15000,
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
				ctx := args.Get(0).(context.Context)
				fn := args.Get(1).(func(context.Context) error)
				err := fn(ctx)
				assert.NoError(t, err)
			}).
			Return(nil).Once()

		// =========================================================

		// 2. Buat item (Akan tereksekusi berkat Run() di atas)
		mockOrderRepo.On("CreateItem", mock.Anything, mock.MatchedBy(func(item *domain.OrderItem) bool {
			// Kita pastikan Usecase mem-passing CustomName ke entity OrderItem
			return item.CustomName == "Kemeja PDH Bank ERT" &&
				item.ProductID == productID &&
				item.Qty == 2
		})).Return(nil).Once()

		// 3. Masuk ke recalculateOrderTotal
		mockOrderRepo.On("GetByID", mock.Anything, orderID).Return(&domain.Order{
			ID: orderID,
			Items: []domain.OrderItem{
				{Price: 15000, Qty: 2},
			},
		}, nil).Once()

		mockPaymentRepo.On("GetByOrderID", mock.Anything, orderID).Return([]domain.Payment{}, nil).Once()
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
	mockBatchPORepo := new(mocks.BatchPORepository)
	mockProductMaterialUsecase := new(mocks.ProductMaterialUsecase)
	mockWorkLogRepo := new(mocks.WorkLogRepository)

	u := usecase.NewOrderUsecase(mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, mockBatchPORepo, mockPaymentRepo, mockTxManager, mockProductMaterialUsecase, mockWorkLogRepo, time.Second*2)

	orderID := "order-123"
	itemID := "item-123"
	productID := "prod-123"
	input := domain.OrderItemInput{
		ProductID:  productID,
		CustomName: "Kemeja PDH Revisi",
		Qty:        5, // Qty diubah jadi 5
		Price:      0, // Misal harga dikosongkan agar pakai BasePrice product
	}

	t.Run("Success", func(t *testing.T) {
		mockOrderRepo.ExpectedCalls = nil
		mockProductRepo.ExpectedCalls = nil
		mockPaymentRepo.ExpectedCalls = nil // Reset mock payment
		mockTxManager.ExpectedCalls = nil   // Reset mock transaction

		existingItem := &domain.OrderItem{ID: itemID, OrderID: orderID, ProductID: productID, CustomName: "Nama Lama", Qty: 2, Price: 10000}

		mockOrderRepo.On("GetItemByID", mock.Anything, orderID, itemID).Return(existingItem, nil).Once()
		mockProductRepo.On("GetByID", mock.Anything, productID).Return(&domain.Product{ID: productID, BasePrice: 20000}, nil).Once()

		// =========================================================
		// 🚨 PERBAIKAN: MOCK TRANSACTION MANAGER
		// =========================================================
		mockTxManager.On("RunInTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			Run(func(args mock.Arguments) {
				ctx := args.Get(0).(context.Context)
				fn := args.Get(1).(func(context.Context) error)
				err := fn(ctx)
				assert.NoError(t, err)
			}).
			Return(nil).Once()
		// =========================================================

		// 2. Update item (Dijalankan di DALAM transaksi)
		mockOrderRepo.On("UpdateItem", mock.Anything, mock.MatchedBy(func(item *domain.OrderItem) bool {
			return item.CustomName == "Kemeja PDH Revisi" && item.Qty == 5
		})).Return(nil).Once()

		// 3. Masuk ke recalculateOrderTotal (Dijalankan di DALAM transaksi)
		mockOrderRepo.On("GetByID", mock.Anything, orderID).Return(&domain.Order{
			ID: orderID,
			Items: []domain.OrderItem{
				{Price: 20000, Qty: 5},
			},
		}, nil).Once()

		// Skenario Partial: Customer sudah DP 40.000 (Tagihan 100.000)
		mockPaymentRepo.On("GetByOrderID", mock.Anything, orderID).Return([]domain.Payment{
			{Amount: 40000, Status: domain.PaymentVerificationVerified},
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
	mockBatchPORepo := new(mocks.BatchPORepository)
	mockProductMaterialUsecase := new(mocks.ProductMaterialUsecase)
	mockWorkLogRepo := new(mocks.WorkLogRepository)

	u := usecase.NewOrderUsecase(mockOrderRepo, mockCustomerRepo, mockUserRepo, mockProductRepo, mockBatchPORepo, mockPaymentRepo, mockTxManager, mockProductMaterialUsecase, mockWorkLogRepo, time.Second*2)

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
			{Amount: 15000, Status: domain.PaymentVerificationVerified},
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
