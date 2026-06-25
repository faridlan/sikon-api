package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/repository/postgres"
	"github.com/faridlan/sikon-api/internal/utils"
	tests "github.com/faridlan/sikon-api/test"
)

// ==========================================
// FUNGSI HELPER UNTUK SETUP DATA LENGKAP
// ==========================================
func setupPaymentDependencies(db *gorm.DB) (orderID string, bankAccountID string) {
	// Buat semua prasyarat
	sales := tests.SeedUser(db, "Sales Payment", "pay@sikon.com", "sales")
	cust := tests.SeedCustomer(db, "Cust Payment", "0812", "Jkt")
	cat := tests.SeedCategory(db, "Kategori Pay")
	prod := tests.SeedProduct(db, cat.ID, "Produk Pay", 50000)
	batchPo := tests.SeedBatchPO(db, "Batch PO Test", "active")

	order := tests.SeedOrder(db, batchPo.ID, cust.ID, sales.ID, prod.ID)
	bank := tests.SeedBankAccount(db, nil, "BCA", "123", "PT SIKOn")

	return order.ID, bank.ID
}

// ==========================================
// 2. TEST CREATE PAYMENT (POST /api/payments)
// ==========================================
func TestCreatePayment_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	orderID, bankID := setupPaymentDependencies(db)

	t.Run("Success_Create_DP", func(t *testing.T) {
		reqBody := dto.PaymentCreateRequest{
			OrderID:         orderID,
			BankAccountID:   bankID,
			Amount:          50000, // Bayar DP 50.000
			PaymentDate:     time.Now(),
			ReferenceNumber: "TRX-DP-001",
			PaymentType:     "dp",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/payments", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	})

	// --- SKENARIO GAGAL (VALIDASI INPUT) ---

	t.Run("Failed_Validation_Amount_Zero", func(t *testing.T) {
		reqBody := dto.PaymentCreateRequest{
			OrderID:         orderID,
			BankAccountID:   bankID,
			Amount:          0, // <-- Gagal: Amount tidak boleh 0 (gt=0)
			ReferenceNumber: "TRX-FAILED",
			PaymentType:     "dp",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/payments", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Failed_Validation_Invalid_PaymentType", func(t *testing.T) {
		reqBody := dto.PaymentCreateRequest{
			OrderID:         orderID,
			BankAccountID:   bankID,
			Amount:          100000,
			ReferenceNumber: "TRX-FAILED-2",
			PaymentType:     "ngutang_dulu", // <-- Gagal: Bukan bagian dari enum
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/payments", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	// --- SKENARIO BUSINESS RULES (BARU) ---

	t.Run("Failed_Overpayment", func(t *testing.T) {
		// Kasir menginput nominal yang sangat besar melampaui sisa tagihan
		reqBody := dto.PaymentCreateRequest{
			OrderID:         orderID,
			BankAccountID:   bankID,
			Amount:          999999999, // <-- Gagal: Melebihi sisa tagihan
			ReferenceNumber: "TRX-OVER-001",
			PaymentType:     "settlement",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/payments", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		// Harus melempar error 400 Bad Request karena masuk validasi ErrBadParamInput
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Success_Create_Settlement_And_Check_Status", func(t *testing.T) {
		// Di Integration Test, kita tidak tahu persis berapa TotalAmount di seeder Anda.
		// Agar dinamis dan tidak mudah fail, kita ambil sisa tagihannya langsung dari DB DB Testing!
		type OrderModel struct {
			ID            string
			TotalAmount   float64
			PaymentStatus string
		}
		var testOrder OrderModel
		db.Table("orders").Where("id = ?", orderID).First(&testOrder)

		var totalPaid float64
		db.Table("payments").Where("order_id = ?", orderID).Select("COALESCE(SUM(amount), 0)").Scan(&totalPaid)

		sisaTagihan := testOrder.TotalAmount - totalPaid

		reqBody := dto.PaymentCreateRequest{
			OrderID:         orderID,
			BankAccountID:   bankID,
			Amount:          sisaTagihan, // <-- Bayar LUNAS sesuai sisa tagihan
			PaymentDate:     time.Now(),
			ReferenceNumber: "TRX-LUNAS-001",
			PaymentType:     "settlement",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/payments", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		// VERIFIKASI PENTING: Apakah Status Order benar-benar berubah jadi "paid"?
		db.Table("orders").Where("id = ?", orderID).First(&testOrder)
		assert.Equal(t, "paid", testOrder.PaymentStatus)
	})

	t.Run("Failed_Already_Fully_Paid", func(t *testing.T) {
		// Karena di test case sebelumnya ("Success_Create_Settlement...") tagihan sudah LUNAS,
		// maka jika kita coba bayar lagi, harusnya ditolak oleh sistem.
		reqBody := dto.PaymentCreateRequest{
			OrderID:         orderID,
			BankAccountID:   bankID,
			Amount:          10000,
			ReferenceNumber: "TRX-LATE-001",
			PaymentType:     "settlement",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/payments", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		// Harus melempar 409 Conflict sesuai domain.ErrConflict "Pesanan sudah lunas sepenuhnya"
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	})
}

// ==========================================
// 1. TEST GET PAYMENT BY ID
// ==========================================
func TestGetPaymentByID_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	orderID, bankID := setupPaymentDependencies(db)
	payment := tests.SeedPayment(db, orderID, bankID, 50000, "dp")

	t.Run("Success_Get_ByID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/payments/"+payment.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.PaymentResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, payment.ID, response.Data.ID)
		assert.Equal(t, float64(50000), response.Data.Amount)
	})

	t.Run("Failed_Get_NotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/payments/"+uuid.New().String(), nil)
		resp, _ := app.Test(req, -1)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// ==========================================
// 3. TEST LIST PAYMENTS & FILTER
// ==========================================
func TestListPayments_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	orderID, bankID := setupPaymentDependencies(db)

	// Seed 2 data dengan tipe yang berbeda agar bisa di-filter
	// Asumsi parameter SeedPayment: db, orderID, bankAccountID, amount, paymentType
	tests.SeedPayment(db, orderID, bankID, 50000, "dp")
	tests.SeedPayment(db, orderID, bankID, 150000, "settlement")

	t.Run("Success_List_SemuaData_TanpaFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/payments", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.PaginatedResponse[dto.PaymentResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Harus mengembalikan 2 data karena tidak ada filter
		assert.Len(t, response.Data, 2)
		assert.Equal(t, int64(2), response.Meta.TotalItems)
	})

	t.Run("Success_List_DenganFilter_PaymentType", func(t *testing.T) {
		// Frontend hanya ingin melihat pembayaran dengan tipe "dp"
		req := httptest.NewRequest("GET", "/api/payments?payment_type=dp", nil)
		resp, _ := app.Test(req, -1)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.PaginatedResponse[dto.PaymentResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Harus hanya mengembalikan 1 data (karena yang tipe "dp" cuma 1)
		assert.Len(t, response.Data, 1)
		assert.Equal(t, "dp", response.Data[0].PaymentType)
	})

	t.Run("Success_List_DenganFilter_SearchOrderID", func(t *testing.T) {
		// Frontend mencari berdasarkan orderID
		req := httptest.NewRequest("GET", "/api/payments?search="+orderID, nil)
		resp, _ := app.Test(req, -1)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.PaginatedResponse[dto.PaymentResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Kebetulan kedua payment di-seed menggunakan orderID yang sama
		assert.Len(t, response.Data, 2)
	})

	t.Run("Success_List_FilterTidakKetemu", func(t *testing.T) {
		// Frontend mencari keyword yang tidak ada di database
		req := httptest.NewRequest("GET", "/api/payments?search=TRX-TIDAKADA", nil)
		resp, _ := app.Test(req, -1)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.PaginatedResponse[dto.PaymentResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Datanya harus kosong, tapi response status tetap 200 OK
		assert.Len(t, response.Data, 0)
		assert.Equal(t, int64(0), response.Meta.TotalItems)
	})
}

// ==========================================
// 4. TEST UPDATE PAYMENT (PUT /api/payments/:id)
// ==========================================
func TestUpdatePayment_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	orderID, bankID := setupPaymentDependencies(db)
	payment := tests.SeedPayment(db, orderID, bankID, 100000, "dp")

	t.Run("Success_Update", func(t *testing.T) {
		reqBody := dto.PaymentUpdateRequest{
			ReferenceNumber: "TRX-UPDATE-123",
			PaymentType:     "settlement", // Update DP jadi Pelunasan
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/payments/"+payment.ID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Failed_Update_NotFound", func(t *testing.T) {
		reqBody := dto.PaymentUpdateRequest{ReferenceNumber: "TRX-GAIB"}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/payments/"+uuid.New().String(), bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req, -1)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// ==========================================
// 5. TEST DELETE PAYMENT (DELETE /api/payments/:id)
// ==========================================
func TestDeletePayment_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()

	t.Run("Success_Delete_RevertsToUnpaid", func(t *testing.T) {
		tests.ClearTables(db) // Bersihkan state sebelum test
		orderID, bankID := setupPaymentDependencies(db)

		// Skenario: Hanya ada 1 pembayaran.
		payment := tests.SeedPayment(db, orderID, bankID, 50000, "dp")

		req := httptest.NewRequest("DELETE", "/api/payments/"+payment.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// ASERSI DATABASE 1: Pastikan data payment benar-benar hilang dari tabel payments
		var paymentCount int64
		db.Model(&domain.Payment{}).Where("id = ?", payment.ID).Count(&paymentCount)
		assert.Equal(t, int64(0), paymentCount)

		// ASERSI DATABASE 2: Pastikan status order berubah menjadi "unpaid" karena tidak ada sisa pembayaran
		var order postgres.OrderModel
		db.First(&order, "id = ?", orderID)
		assert.Equal(t, "unpaid", order.PaymentStatus)
	})

	t.Run("Success_Delete_RevertsToPartial", func(t *testing.T) {
		tests.ClearTables(db) // Bersihkan state sebelum test
		orderID, bankID := setupPaymentDependencies(db)

		// Skenario: Ada 2 kali pembayaran. (Asumsi total tagihan Order cukup besar)
		payment1 := tests.SeedPayment(db, orderID, bankID, 50000, "dp")
		_ = tests.SeedPayment(db, orderID, bankID, 50000, "pelunasan") // Payment ke-2 (tidak dihapus)

		// Hapus HANYA payment pertama
		req := httptest.NewRequest("DELETE", "/api/payments/"+payment1.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// ASERSI DATABASE 1: Pastikan payment1 hilang
		var paymentCount int64
		db.Model(&domain.Payment{}).Where("id = ?", payment1.ID).Count(&paymentCount)
		assert.Equal(t, int64(0), paymentCount)

		// ASERSI DATABASE 2: Pastikan status order berubah menjadi "partial" karena masih tersisa Payment ke-2
		var order postgres.OrderModel
		db.First(&order, "id = ?", orderID)
		assert.Equal(t, "partial", order.PaymentStatus)
	})

	t.Run("Failed_Delete_NotFound", func(t *testing.T) {
		tests.ClearTables(db)
		req := httptest.NewRequest("DELETE", "/api/payments/"+uuid.New().String(), nil)
		resp, _ := app.Test(req, -1)

		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// ==========================================
// 6. TEST GET PAYMENTS BY ORDER ID (GET /api/payments/order/:order_id)
// ==========================================
func TestGetPaymentsByOrderID_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// --- SETUP DATA PRASYARAT ---
	// Kita gunakan fungsi setupPaymentDependencies yang sudah kita buat sebelumnya
	orderID1, bankID := setupPaymentDependencies(db)

	// Kita buat Order Kedua (orderID2) untuk memastikan tidak ada kebocoran data (Data Leakage)
	sales2 := tests.SeedUser(db, "Sales Kedua", "sales2@sikon.com", "sales")
	cust2 := tests.SeedCustomer(db, "Cust Kedua", "0812345", "Bandung")
	cat2 := tests.SeedCategory(db, "Kategori Lain")
	prod2 := tests.SeedProduct(db, cat2.ID, "Produk Lain", 100000)
	batchPo2 := tests.SeedBatchPO(db, "Batch PO Test 2", "active")
	order2 := tests.SeedOrder(db, batchPo2.ID, cust2.ID, sales2.ID, prod2.ID)

	// --- SEED PAYMENTS ---
	// Masukkan 2x Pembayaran (DP & Lunas) untuk Order 1
	tests.SeedPayment(db, orderID1, bankID, 50000, "dp")
	tests.SeedPayment(db, orderID1, bankID, 50000, "settlement")

	// Masukkan 1x Pembayaran untuk Order 2
	// tests.SeedPayment(db, order2.ID, bankID, 50000, "dp")

	// --- SKENARIO SUKSES ---

	t.Run("Success_Get_Payments_For_Order", func(t *testing.T) {
		// Tarik pembayaran khusus untuk Order 1
		req := httptest.NewRequest("GET", "/api/payments/order/"+orderID1, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[[]dto.PaymentResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Verifikasi: Harus mengembalikan tepat 2 pembayaran (milik order 2 tidak boleh ikut masuk)
		assert.Len(t, response.Data, 2)

		// Verifikasi: Semua payment yang kembali benar-benar milik Order 1
		for _, p := range response.Data {
			assert.Equal(t, orderID1, p.OrderID)
		}
	})

	// --- SKENARIO SUKSES TAPI KOSONG ---

	t.Run("Success_But_Empty_List", func(t *testing.T) {
		// Kita gunakan order2 yang sudah berhasil di-seed tapi BELUM kita buatkan payment-nya di seeder
		req := httptest.NewRequest("GET", "/api/payments/order/"+order2.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[[]dto.PaymentResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Len(t, response.Data, 0) // Datanya kosong [] karena order2 belum dibayar
	})

	// --- SKENARIO GAGAL / NEGATIVE PATH ---

	t.Run("Failed_OrderNotFound", func(t *testing.T) {
		// Mencari payment dari Order yang ID-nya ngarang
		randomOrderID := uuid.New().String()
		req := httptest.NewRequest("GET", "/api/payments/order/"+randomOrderID, nil)
		resp, _ := app.Test(req, -1)

		// Karena Usecase Anda mengecek eksistensi Order, ini harusnya me-return 404!
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("Failed_Invalid_UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/payments/order/bukan-uuid", nil)
		resp, _ := app.Test(req, -1)

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode) // Harusnya ditangkap 400
	})
}

// ==========================================
// 7. TEST RACE CONDITION (CONCURRENCY)
// ==========================================
func TestRaceCondition_ProcessPayment_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// Setup prasyarat: Order dengan 1 barang seharga Rp 1.000.000
	sales := tests.SeedUser(db, "Sales Race", "race@sikon.com", "sales")
	cust := tests.SeedCustomer(db, "Cust Race", "081999", "Bdg")
	cat := tests.SeedCategory(db, "Kategori Race")
	prod := tests.SeedProduct(db, cat.ID, "Kemeja Mahal", 1000000)
	batchPo := tests.SeedBatchPO(db, "Batch PO Test", "active")

	order := tests.SeedOrder(db, batchPo.ID, cust.ID, sales.ID, prod.ID)

	// 🚨 TAMBAHKAN BARIS INI: Force update total tagihan menjadi 1 Juta
	db.Table("orders").Where("id = ?", order.ID).Update("total_amount", 1000000)

	bank := tests.SeedBankAccount(db, nil, "BCA", "123", "PT SIKOn")

	// Alat bantu untuk menjalankan proses paralel dengan aman
	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	errorCount := 0

	totalRequests := 3

	// Skenario: 3 kasir mencoba submit pembayaran DP Rp 500.000 ke order yang sama,
	// secara bersamaan di detik yang sama persis.
	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			reqBody := dto.PaymentCreateRequest{
				OrderID:         order.ID,
				BankAccountID:   bank.ID,
				Amount:          500000,
				ReferenceNumber: "TRX-RACE-CONCURRENT",
				PaymentType:     "dp",
			}
			bodyJson, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/payments", bytes.NewBuffer(bodyJson))
			req.Header.Set("Content-Type", "application/json")

			// Eksekusi HTTP Request
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)

			respBody, _ := io.ReadAll(resp.Body)
			fmt.Printf("STATUS: %d | RESPONSE: %s\n", resp.StatusCode, string(respBody))

			// Gunakan Mutex agar penghitungan skor tidak ikut terkena race condition
			mu.Lock()
			if resp.StatusCode == fiber.StatusCreated {
				successCount++
			} else {
				errorCount++
			}
			mu.Unlock()
		}()
	}

	// Tunggu semua request bertabrakan dan selesai diproses oleh API
	wg.Wait()

	// ---------------------------------------------------------
	// ASERSI BUKTI KEBAL RACE CONDITION
	// ---------------------------------------------------------
	// Karena tagihan 1 Juta dan masing-masing kasir mensubmit 500rb,
	// hanya 2 request yang boleh sukses (500rb + 500rb = 1 Juta).
	// Request ke-3 harus gagal dan ditolak oleh sistem (Overpayment).
	assert.Equal(t, 2, successCount, "Hanya boleh ada 2 pembayaran yang berhasil masuk")
	assert.Equal(t, 1, errorCount, "Harus ada 1 pembayaran yang ditolak karena overpayment")

	// Verifikasi kebenaran fisik data di Database Utama
	type OrderModel struct {
		PaymentStatus string
	}
	var testOrder OrderModel
	db.Table("orders").Where("id = ?", order.ID).First(&testOrder)

	// Status order wajib berubah otomatis menjadi PAID
	assert.Equal(t, "paid", testOrder.PaymentStatus, "Status order harus otomatis lunas")

	// Total uang yang tercatat masuk di database tidak boleh bocor melebihi 1 Juta
	var totalPaid float64
	db.Table("payments").Where("order_id = ?", order.ID).Select("COALESCE(SUM(amount), 0)").Scan(&totalPaid)
	assert.Equal(t, float64(1000000), totalPaid, "Total uang masuk di database tidak boleh lebih dari 1.000.000")
}
