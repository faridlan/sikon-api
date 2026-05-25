package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
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

	order := tests.SeedOrder(db, cust.ID, sales.ID, prod.ID)
	bank := tests.SeedBankAccount(db, nil, "BCA", "123", "PT SIKOn")

	return order.ID, bank.ID
}

// ==========================================
// 1. TEST CREATE PAYMENT (POST /api/payments)
// ==========================================
func TestCreatePayment_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	orderID, bankID := setupPaymentDependencies(db)

	t.Run("Success_Create_DP", func(t *testing.T) {
		reqBody := dto.PaymentCreateRequest{
			OrderID:         orderID,
			BankAccountID:   bankID,
			Amount:          50000,
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

	// --- SKENARIO GAGAL ---

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
			PaymentType:     "ngutang_dulu", // <-- Gagal: Bukan bagian dari enum (dp, settlement, installment)
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/payments", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// ==========================================
// 2. TEST GET & LIST PAYMENT
// ==========================================
func TestGetPayment_Integration(t *testing.T) {
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

	t.Run("Success_List_Payments", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/payments", nil)
		resp, _ := app.Test(req, -1)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

// ==========================================
// 3. TEST UPDATE PAYMENT (PUT /api/payments/:id)
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
// 4. TEST DELETE PAYMENT (DELETE /api/payments/:id)
// ==========================================
func TestDeletePayment_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	orderID, bankID := setupPaymentDependencies(db)
	payment := tests.SeedPayment(db, orderID, bankID, 50000, "dp")

	t.Run("Success_Delete", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/payments/"+payment.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Failed_Delete_NotFound", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/payments/"+uuid.New().String(), nil)
		resp, _ := app.Test(req, -1)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// ==========================================
// 5. TEST GET PAYMENTS BY ORDER ID (GET /api/payments/order/:order_id)
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
	order2 := tests.SeedOrder(db, cust2.ID, sales2.ID, prod2.ID)

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
