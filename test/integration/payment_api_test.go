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
