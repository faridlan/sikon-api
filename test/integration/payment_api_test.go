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
func setupPaymentDependencies(db *gorm.DB) (orderID string, bankAccountID string, userID string) {
	sales := tests.SeedUser(db, "Sales Payment", "pay@sikon.com", "sales")
	cust := tests.SeedCustomer(db, "Cust Payment", "0812", "Jkt")
	cat := tests.SeedCategory(db, "Kategori Pay")
	prod := tests.SeedProduct(db, cat.ID, "Produk Pay", 50000)
	batchPo := tests.SeedBatchPO(db, "Batch PO Test", "active")

	order := tests.SeedOrder(db, batchPo.ID, cust.ID, sales.ID, prod.ID)
	bank := tests.SeedBankAccount(db, nil, "BCA", "123", "PT SIKOn")

	return order.ID, bank.ID, sales.ID
}

// ==========================================
// 1. TEST CREATE PAYMENT (POST /api/payments)
// ==========================================
func TestCreatePayment_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	orderID, bankID, userID := setupPaymentDependencies(db)

	t.Run("Success_Create_DP_Pending_Verification", func(t *testing.T) {
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
		req.Header.Set("X-User-Id", userID)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.PaymentResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Verifikasi bahwa status pembayaran diajukan dengan status "pending"
		assert.Equal(t, "pending", response.Data.Status)

		// Verifikasi bahwa sebelum di-approve Finance, order masih unpaid
		var updatedOrder postgres.OrderModel
		err = db.Table("orders").Where("id = ?", orderID).First(&updatedOrder).Error
		assert.NoError(t, err)
		assert.Equal(t, "unpaid", updatedOrder.PaymentStatus)
	})

	t.Run("Failed_Validation_Amount_Zero", func(t *testing.T) {
		reqBody := dto.PaymentCreateRequest{
			OrderID:         orderID,
			BankAccountID:   bankID,
			Amount:          0, // amount gt=0
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
			PaymentType:     "ngutang_dulu",
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
// 2. TEST VERIFY PAYMENT (PATCH /api/payments/:id/verify)
// ==========================================
func TestVerifyPayment_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	orderID, bankID, userID := setupPaymentDependencies(db)
	payment := tests.SeedPayment(db, orderID, bankID, 50000, "dp")

	t.Run("Success_Verify_Payment_And_Updates_Order_To_Partial", func(t *testing.T) {
		reqBody := dto.PaymentVerifyRequest{
			Status: "verified",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/api/payments/"+payment.ID+"/verify", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-Id", userID)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Verifikasi status Payment di database berubah jadi verified
		var updatedPayment postgres.PaymentModel
		db.Table("payments").Where("id = ?", payment.ID).First(&updatedPayment)
		assert.Equal(t, "verified", updatedPayment.Status)

		// Verifikasi PaymentStatus pada Order berubah otomatis jadi partial
		var updatedOrder postgres.OrderModel
		db.Table("orders").Where("id = ?", orderID).First(&updatedOrder)
		assert.Equal(t, "partial", updatedOrder.PaymentStatus)
	})

	t.Run("Success_Reject_Payment_Keeps_Order_Unpaid", func(t *testing.T) {
		tests.ClearTables(db)
		orderID2, bankID2, userID2 := setupPaymentDependencies(db)
		payment2 := tests.SeedPayment(db, orderID2, bankID2, 50000, "dp")

		reqBody := dto.PaymentVerifyRequest{
			Status: "rejected",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/api/payments/"+payment2.ID+"/verify", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-Id", userID2)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Verifikasi status order tetap unpaid
		var updatedOrder postgres.OrderModel
		db.Table("orders").Where("id = ?", orderID2).First(&updatedOrder)
		assert.Equal(t, "unpaid", updatedOrder.PaymentStatus)
	})

	t.Run("Failed_Verify_Invalid_Status", func(t *testing.T) {
		reqBody := dto.PaymentVerifyRequest{
			Status: "gajadi",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/api/payments/"+payment.ID+"/verify", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req, -1)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// ==========================================
// 3. TEST GET PAYMENT BY ID
// ==========================================
func TestGetPaymentByID_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	orderID, bankID, userID := setupPaymentDependencies(db)
	payment := tests.SeedPayment(db, orderID, bankID, 50000, "dp")

	t.Run("Success_Get_ByID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/payments/"+payment.ID, nil)
		req.Header.Set("X-User-Id", userID)
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
// 4. TEST LIST PAYMENTS & FILTER
// ==========================================
func TestListPayments_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	orderID, bankID, _ := setupPaymentDependencies(db)

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

		assert.Len(t, response.Data, 2)
		assert.Equal(t, int64(2), response.Meta.TotalItems)
	})

	t.Run("Success_List_DenganFilter_PaymentType", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/payments?payment_type=dp", nil)
		resp, _ := app.Test(req, -1)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.PaginatedResponse[dto.PaymentResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Len(t, response.Data, 1)
		assert.Equal(t, "dp", response.Data[0].PaymentType)
	})

	t.Run("Success_List_DenganFilter_StatusPending", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/payments?status=pending", nil)
		resp, _ := app.Test(req, -1)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.PaginatedResponse[dto.PaymentResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Len(t, response.Data, 2)
	})
}

// ==========================================
// 5. TEST UPDATE PAYMENT (PUT /api/payments/:id)
// ==========================================
func TestUpdatePayment_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	orderID, bankID, userID := setupPaymentDependencies(db)
	payment := tests.SeedPayment(db, orderID, bankID, 100000, "dp")

	t.Run("Success_Update", func(t *testing.T) {
		reqBody := dto.PaymentUpdateRequest{
			ReferenceNumber: "TRX-UPDATE-123",
			PaymentType:     "settlement",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/payments/"+payment.ID, bytes.NewBuffer(bodyJson))
		req.Header.Set("X-User-Id", userID)
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
// 6. TEST DELETE PAYMENT (DELETE /api/payments/:id)
// ==========================================
func TestDeletePayment_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()

	t.Run("Success_Delete_RevertsToUnpaid", func(t *testing.T) {
		tests.ClearTables(db)
		orderID, bankID, userID := setupPaymentDependencies(db)

		payment := tests.SeedPayment(db, orderID, bankID, 50000, "dp")

		// Verifikasi pembayaran terlebih dahulu agar memengaruhi order
		db.Table("payments").Where("id = ?", payment.ID).Update("status", "verified")
		db.Table("orders").Where("id = ?", orderID).Update("payment_status", "partial")

		req := httptest.NewRequest("DELETE", "/api/payments/"+payment.ID, nil)
		req.Header.Set("X-User-Id", userID)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var paymentCount int64
		db.Model(&domain.Payment{}).Where("id = ?", payment.ID).Count(&paymentCount)
		assert.Equal(t, int64(0), paymentCount)

		var order postgres.OrderModel
		db.First(&order, "id = ?", orderID)
		assert.Equal(t, "unpaid", order.PaymentStatus)
	})

	t.Run("Failed_Delete_NotFound", func(t *testing.T) {
		tests.ClearTables(db)
		req := httptest.NewRequest("DELETE", "/api/payments/"+uuid.New().String(), nil)
		resp, _ := app.Test(req, -1)

		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// ==========================================
// 7. TEST GET PAYMENTS BY ORDER ID (GET /api/payments/order/:order_id)
// ==========================================
func TestGetPaymentsByOrderID_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	orderID1, bankID, _ := setupPaymentDependencies(db)

	sales2 := tests.SeedUser(db, "Sales Kedua", "sales2@sikon.com", "sales")
	cust2 := tests.SeedCustomer(db, "Cust Kedua", "0812345", "Bandung")
	cat2 := tests.SeedCategory(db, "Kategori Lain")
	prod2 := tests.SeedProduct(db, cat2.ID, "Produk Lain", 100000)
	batchPo2 := tests.SeedBatchPO(db, "Batch PO Test 2", "active")
	order2 := tests.SeedOrder(db, batchPo2.ID, cust2.ID, sales2.ID, prod2.ID)

	tests.SeedPayment(db, orderID1, bankID, 50000, "dp")
	tests.SeedPayment(db, orderID1, bankID, 50000, "settlement")

	t.Run("Success_Get_Payments_For_Order", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/payments/order/"+orderID1, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[[]dto.PaymentResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Len(t, response.Data, 2)
		for _, p := range response.Data {
			assert.Equal(t, orderID1, p.OrderID)
		}
	})

	t.Run("Success_But_Empty_List", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/payments/order/"+order2.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[[]dto.PaymentResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Len(t, response.Data, 0)
	})

	t.Run("Failed_OrderNotFound", func(t *testing.T) {
		randomOrderID := uuid.New().String()
		req := httptest.NewRequest("GET", "/api/payments/order/"+randomOrderID, nil)
		resp, _ := app.Test(req, -1)

		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("Failed_Invalid_UUID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/payments/order/bukan-uuid", nil)
		resp, _ := app.Test(req, -1)

		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// ==========================================
// 8. TEST RACE CONDITION (CONCURRENCY)
// ==========================================
func TestRaceCondition_ProcessPayment_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	sales := tests.SeedUser(db, "Sales Race", "race@sikon.com", "sales")
	cust := tests.SeedCustomer(db, "Cust Race", "081999", "Bdg")
	cat := tests.SeedCategory(db, "Kategori Race")
	prod := tests.SeedProduct(db, cat.ID, "Kemeja Mahal", 1000000)
	batchPo := tests.SeedBatchPO(db, "Batch PO Test", "active")

	order := tests.SeedOrder(db, batchPo.ID, cust.ID, sales.ID, prod.ID)
	db.Table("orders").Where("id = ?", order.ID).Update("total_amount", 1000000)

	bank := tests.SeedBankAccount(db, nil, "BCA", "123", "PT SIKOn")

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	errorCount := 0

	totalRequests := 3

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

			resp, err := app.Test(req, -1)
			assert.NoError(t, err)

			mu.Lock()
			if resp.StatusCode == fiber.StatusCreated {
				successCount++
			} else {
				errorCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// 3 Pengajuan pembayaran berhasil disimpan dengan status 'pending'
	assert.Equal(t, 3, successCount)

	// Selanjutnya simulasi verifikasi oleh Divisi Finance
	var payments []postgres.PaymentModel
	db.Table("payments").Where("order_id = ?", order.ID).Find(&payments)

	// Ambil 2 pembayaran pertama untuk di-verify
	for i := 0; i < 2; i++ {
		verifyBody, _ := json.Marshal(dto.PaymentVerifyRequest{Status: "verified"})
		req := httptest.NewRequest("PATCH", fmt.Sprintf("/api/payments/%s/verify", payments[i].ID), bytes.NewBuffer(verifyBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-User-Id", sales.ID) // 👈 Header ID User Penanggung Jawab
		resp, _ := app.Test(req, -1)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	}

	// Setelah 2x 500rb di-verify, Order otomatis lunas (PAID)
	var testOrder postgres.OrderModel
	db.Table("orders").Where("id = ?", order.ID).First(&testOrder)
	assert.Equal(t, "paid", testOrder.PaymentStatus)
}
