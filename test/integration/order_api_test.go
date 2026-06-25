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

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/repository/postgres"
	"github.com/faridlan/sikon-api/internal/utils"
	tests "github.com/faridlan/sikon-api/test"
)

// ==========================================
// 1. TEST CREATE ORDER (POST /api/orders)
// ==========================================
func TestCreateOrder_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// --- SETUP DATA PRASYARAT (SANGAT PENTING) ---
	sales := tests.SeedUser(db, "Sales Order", "sales.ord@sikon.com", "sales")
	customer := tests.SeedCustomer(db, "Cust Order", "0811", "Jakarta")
	category := tests.SeedCategory(db, "Kemeja")
	product := tests.SeedProduct(db, category.ID, "Kemeja PDH", 150000)

	// 1. SKENARIO CREATE SURAT PENAWARAN (Normal Flow)
	t.Run("Success_Create_Order", func(t *testing.T) {
		validUntil := time.Now().AddDate(0, 0, 7)
		reqBody := dto.OrderCreateRequest{
			CustomerID:      customer.ID,
			SalesID:         sales.ID,
			ShippingCost:    20000,
			CourierName:     "JNE",
			ShippingAddress: "Alamat Kirim",
			ValidUntil:      &validUntil,
			TermsConditions: "Syarat 1 2 3",
			OrderStatus:     "quotation",
			Items: []dto.OrderItemRequest{
				{
					ProductID: product.ID,
					Qty:       2,
					Price:     150000,
					Details:   map[string]any{"Benang": "Benang Bordir Menggunakan Benang Polyster", "Bordir": "Bordir Menggunakan Sistem Komputerisasi", "Jahitan": "Jahit Rapi", "Bahan": map[string]any{"Name": "Katun Baby Canvas", "Spec": "menggunakan baby canvas", "Color": "Hitam"}},
				},
			},
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/orders", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.OrderResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "quotation", response.Data.OrderStatus)
		assert.Equal(t, "Syarat 1 2 3", response.Data.TermsConditions)
		assert.NotNil(t, response.Data.ValidUntil)
		assert.Equal(t, 150000.0, response.Data.Items[0].Price)
		assert.Equal(t, "Hitam", response.Data.Items[0].Details["Bahan"].(map[string]any)["Color"])

		// --- TAMBAHAN ASSERSI PERHITUNGAN KEUANGAN ---
		assert.Equal(t, 300000.0, response.Data.Subtotal)
		assert.Equal(t, 320000.0, response.Data.TotalAmount)
	})

	// 2. 🚨 REVISI SKENARIO: ANTI-BYPASS STATUS ORDER
	t.Run("Success_Create_Order_But_Forced_As_Quotation", func(t *testing.T) {
		reqBody := dto.OrderCreateRequest{
			CustomerID:      customer.ID,
			SalesID:         sales.ID,
			OrderStatus:     "pending", // <-- Frontend mencoba mengirim status pending secara paksa
			ShippingCost:    20000,
			CourierName:     "JNT",
			ShippingAddress: "Alamat Langsung",
			Items: []dto.OrderItemRequest{
				{
					ProductID: product.ID,
					Qty:       1,
					Price:     150000,
					Details:   map[string]any{"Benang": "Benang Bordir Menggunakan Benang Polyster", "Bordir": "Bordir Menggunakan Sistem Komputerisasi", "Jahitan": "Jahit Rapi", "Bahan": map[string]any{"Name": "Katun Baby Canvas", "Spec": "menggunakan baby canvas", "Color": "Hitam"}},
				},
			},
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/orders", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode) // Tetap sukses membuat entitas data

		var response utils.SuccessResponse[dto.OrderResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// 🚨 EKSPEKTASI BARU: Sistem Backend HARUS memotong manipulasi data dari luar
		// dan memaksanya tetap lahir sebagai 'quotation' dan 'unpaid'
		assert.Equal(t, "quotation", response.Data.OrderStatus, "Sistem kebobolan! Harusnya status dipaksa menjadi quotation")
		assert.Equal(t, "unpaid", response.Data.PaymentStatus)
	})

	// 3. SKENARIO GAGAL VALIDASI
	t.Run("Failed_Validation_Empty_Items", func(t *testing.T) {
		reqBody := dto.OrderCreateRequest{
			CustomerID: customer.ID,
			SalesID:    sales.ID,
			Items:      []dto.OrderItemRequest{}, // Items kosong
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/orders", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// ==========================================
// 2. TEST GET ORDER (GET /api/orders/:id)
// ==========================================
func TestGetOrder_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	sales := tests.SeedUser(db, "Sales", "s@s.com", "sales")
	cust := tests.SeedCustomer(db, "Cust", "08", "Jkt")
	cat := tests.SeedCategory(db, "Cat")
	prod := tests.SeedProduct(db, cat.ID, "Prod", 100)

	order := tests.SeedOrder(db, cust.ID, sales.ID, prod.ID)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/orders/"+order.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.OrderResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, order.ID, response.Data.ID)
		assert.Equal(t, cust.ID, response.Data.CustomerID)
		assert.Len(t, response.Data.Items, 1) // Pastikan Itemnya ikut terbawa (Eager Loading)
	})
}

// ==========================================
// 3. TEST UPDATE ORDER DETAILS (PUT /api/orders/:id)
// ==========================================
func TestUpdateOrder_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	sales := tests.SeedUser(db, "S", "s@s.com", "sales")
	cust := tests.SeedCustomer(db, "C", "08", "Jkt")
	cat := tests.SeedCategory(db, "Cat")
	prod := tests.SeedProduct(db, cat.ID, "Prod", 100)
	order := tests.SeedOrder(db, cust.ID, sales.ID, prod.ID)

	t.Run("Success_Update_Shipping_And_Terms", func(t *testing.T) {
		validUntilUpdate := time.Now().AddDate(0, 0, 14)
		reqBody := dto.OrderUpdateRequest{
			ShippingCost:    50000, // Ongkir DIUBAH jadi 50.000
			CourierName:     "SiCepat",
			ShippingAddress: "Alamat Update",
			Notes:           "Catatan Update",
			ValidUntil:      &validUntilUpdate,
			TermsConditions: "Lunas Sebelum Kirim",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/orders/"+order.ID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.OrderResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// --- ASSERSI TAMBAHAN ---
		assert.Equal(t, "Lunas Sebelum Kirim", response.Data.TermsConditions)
		assert.Equal(t, 50000.0, response.Data.ShippingCost)

		// Kunci Uji Coba: Subtotal dari DB Seeder adalah 100.000
		// Ketika ongkir di-update jadi 50.000, Grand Total BARU harus menjadi 150.000
		assert.Equal(t, 150000.0, response.Data.TotalAmount)
	})

	// --- SKENARIO GAGAL (NEGATIVE PATH) ---

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		reqBody := dto.OrderUpdateRequest{CourierName: "J&T"}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/orders/"+randomID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode) // Harusnya 404

		var response utils.ErrorResponse
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)
	})

	t.Run("Failed_Invalid_UUID", func(t *testing.T) {
		reqBody := dto.OrderUpdateRequest{CourierName: "J&T"}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/orders/bukan-uuid", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode) // Harusnya 400

		var response utils.ErrorResponse
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)
	})

	t.Run("Failed_Validation_Negative_Shipping", func(t *testing.T) {
		reqBody := dto.OrderUpdateRequest{
			ShippingCost: -15000, // <-- Invalid: Tidak boleh kurang dari 0 (gte=0)
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/orders/"+order.ID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode) // Harusnya 400

		var response utils.ErrorResponse
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)
	})
}

// ==========================================
// 4. TEST UPDATE ORDER STATUS (PATCH /api/orders/:id/status)
// ==========================================
func TestUpdateOrderStatus_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	sales := tests.SeedUser(db, "S", "s@s.com", "sales")
	cust := tests.SeedCustomer(db, "C", "08", "Jkt")
	cat := tests.SeedCategory(db, "Cat")
	prod := tests.SeedProduct(db, cat.ID, "Prod", 100)
	order := tests.SeedOrder(db, cust.ID, sales.ID, prod.ID)

	// =========================================================================
	// 1. TAHAP QUOTATION -> PENDING
	// =========================================================================
	t.Run("Failed_Update_Quotation_To_Pending_No_DP", func(t *testing.T) {
		// SETUP: Kondisi awal Quotation tapi Customer belum transfer sama sekali
		db.Exec("UPDATE orders SET order_status = ?, payment_status = ? WHERE id = ?", "quotation", "unpaid", order.ID)

		reqBody := dto.OrderStatusUpdateRequest{OrderStatus: "pending"}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/api/orders/"+order.ID+"/status", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		// ASERSI: Ditolak masuk antrean karena belum ada uang (409 Conflict)
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	})

	t.Run("Success_Update_Quotation_To_Pending_With_DP", func(t *testing.T) {
		// SETUP: Customer sudah bayar DP (partial)
		db.Exec("UPDATE orders SET order_status = ?, payment_status = ? WHERE id = ?", "quotation", "partial", order.ID)

		reqBody := dto.OrderStatusUpdateRequest{OrderStatus: "pending"}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/api/orders/"+order.ID+"/status", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	// =========================================================================
	// 2. TAHAP PENDING -> PRODUCTION -> READY
	// =========================================================================
	t.Run("Success_Update_Pending_To_Production", func(t *testing.T) {
		db.Exec("UPDATE orders SET order_status = ?, payment_status = ? WHERE id = ?", "pending", "partial", order.ID)

		reqBody := dto.OrderStatusUpdateRequest{OrderStatus: "production"}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/api/orders/"+order.ID+"/status", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Success_Update_Production_To_Ready", func(t *testing.T) {
		// SETUP: Kain sudah beres dijahit, masuk ke gudang
		db.Exec("UPDATE orders SET order_status = ?, payment_status = ? WHERE id = ?", "production", "partial", order.ID)

		reqBody := dto.OrderStatusUpdateRequest{OrderStatus: "ready"}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/api/orders/"+order.ID+"/status", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	// =========================================================================
	// 3. TAHAP READY -> COMPLETED (PENGAMBILAN BARANG)
	// =========================================================================
	t.Run("Failed_Update_Ready_To_Completed_Not_Paid", func(t *testing.T) {
		// SETUP: Barang siap kirim, tapi baru dibayar DP (Barang ditahan)
		db.Exec("UPDATE orders SET order_status = ?, payment_status = ? WHERE id = ?", "ready", "partial", order.ID)

		reqBody := dto.OrderStatusUpdateRequest{OrderStatus: "completed"}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/api/orders/"+order.ID+"/status", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		// ASERSI: Ditolak karena belum lunas
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	})

	t.Run("Success_Update_Ready_To_Completed_Paid", func(t *testing.T) {
		// SETUP: Barang siap kirim, sisa tagihan sudah dilunasi
		db.Exec("UPDATE orders SET order_status = ?, payment_status = ? WHERE id = ?", "ready", "paid", order.ID)

		reqBody := dto.OrderStatusUpdateRequest{OrderStatus: "completed"}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/api/orders/"+order.ID+"/status", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	// =========================================================================
	// 4. UJI VALIDASI INPUT
	// =========================================================================
	t.Run("Failed_Invalid_Status_Enum", func(t *testing.T) {
		reqBody := dto.OrderStatusUpdateRequest{OrderStatus: "status_ngasal"}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/api/orders/"+order.ID+"/status", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// ==========================================
// 5. TEST DELETE ORDER (DELETE /api/orders/:id)
// ==========================================
func TestDeleteOrder_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	sales := tests.SeedUser(db, "S", "s@s.com", "sales")
	cust := tests.SeedCustomer(db, "C", "08", "Jkt")
	cat := tests.SeedCategory(db, "Cat")
	prod := tests.SeedProduct(db, cat.ID, "Prod", 100)
	order := tests.SeedOrder(db, cust.ID, sales.ID, prod.ID)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/orders/"+order.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Opsional: Verifikasi data benar-benar hilang
		reqCheck := httptest.NewRequest("GET", "/api/orders/"+order.ID, nil)
		respCheck, _ := app.Test(reqCheck, -1)
		assert.Equal(t, fiber.StatusNotFound, respCheck.StatusCode)
	})

	// --- SKENARIO GAGAL (NEGATIVE PATH) ---

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := httptest.NewRequest("DELETE", "/api/orders/"+randomID, nil)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode) // Harusnya 404
	})

	t.Run("Failed_Invalid_UUID", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/orders/ini-bukan-uuid", nil)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode) // Harusnya 400
	})
}

// ==========================================
// 6. TEST LIST ORDERS (GET /api/orders)
// ==========================================
func TestListOrders_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// --- 1. SETUP DATA PRASYARAT ---
	salesA := tests.SeedUser(db, "Sales A", "salesA@sikon.com", "sales")
	salesB := tests.SeedUser(db, "Sales B", "salesB@sikon.com", "sales")
	customer := tests.SeedCustomer(db, "Cust List", "08112233", "Jakarta")
	category := tests.SeedCategory(db, "Kaos")
	product := tests.SeedProduct(db, category.ID, "Kaos Polos", 50000)

	// Buat 3 Order dengan kombinasi Sales dan Status yang berbeda untuk test filter
	order1 := tests.SeedOrder(db, customer.ID, salesA.ID, product.ID)
	order2 := tests.SeedOrder(db, customer.ID, salesA.ID, product.ID)
	order3 := tests.SeedOrder(db, customer.ID, salesB.ID, product.ID)

	// Kita update statusnya secara manual via raw query GORM agar sesuai skenario filter
	db.Exec("UPDATE orders SET order_status = 'quotation' WHERE id = ?", order1.ID)
	db.Exec("UPDATE orders SET order_status = 'pending' WHERE id = ?", order2.ID)
	db.Exec("UPDATE orders SET order_status = 'pending' WHERE id = ?", order3.ID)

	// Struct untuk menangkap response JSON bertipe Paginated
	type PaginatedOrderResponse struct {
		Message string              `json:"message"`
		Data    []dto.OrderResponse `json:"data"`
		Meta    struct {
			CurrentPage int   `json:"current_page"`
			Limit       int   `json:"limit"`
			TotalItems  int64 `json:"total_items"`
			TotalPages  int   `json:"total_pages"`
		} `json:"meta"`
	}

	// --- 2. SKENARIO PENGUJIAN ---

	t.Run("Success_Get_All_Tanpa_Filter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/orders", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response PaginatedOrderResponse
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Harus mengembalikan semua order (Total 3)
		assert.Len(t, response.Data, 3)
		assert.Equal(t, int64(3), response.Meta.TotalItems)
	})

	t.Run("Success_Filter_By_OrderStatus", func(t *testing.T) {
		// Test mencari order yang statusnya 'pending' saja
		req := httptest.NewRequest("GET", "/api/orders?order_status=pending", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response PaginatedOrderResponse
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Hanya ada 2 order dengan status pending
		assert.Len(t, response.Data, 2)
		assert.Equal(t, int64(2), response.Meta.TotalItems)
		assert.Equal(t, "pending", response.Data[0].OrderStatus)
		assert.Equal(t, "pending", response.Data[1].OrderStatus)
	})

	t.Run("Success_Filter_By_SalesID", func(t *testing.T) {
		// Test mencari order milik Sales B saja
		req := httptest.NewRequest("GET", "/api/orders?sales_id="+salesB.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response PaginatedOrderResponse
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Sales B hanya punya 1 order
		assert.Len(t, response.Data, 1)
		assert.Equal(t, int64(1), response.Meta.TotalItems)
		assert.Equal(t, salesB.ID, response.Data[0].SalesID)
	})

	t.Run("Success_Pagination_Limit", func(t *testing.T) {
		// Test Pagination: Ambil halaman 1, tapi batasi hanya 2 data per halaman
		req := httptest.NewRequest("GET", "/api/orders?page=1&limit=2", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response PaginatedOrderResponse
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Karena dilimit 2, maka data yang keluar harus 2
		assert.Len(t, response.Data, 2)
		// Namun TotalItems keseluruhan tetap harus 3
		assert.Equal(t, int64(3), response.Meta.TotalItems)
		// Karena total 3 dibagi limit 2, maka TotalPages harus 2 (Ceil)
		assert.Equal(t, 2, response.Meta.TotalPages)
	})
}

func TestUpdatePaymentStatus_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// Persiapan Data (Seeding)
	sales := tests.SeedUser(db, "S", "s@s.com", "sales")
	cust := tests.SeedCustomer(db, "C", "08", "Jkt")
	cat := tests.SeedCategory(db, "Cat")
	prod := tests.SeedProduct(db, cat.ID, "Prod", 100)
	order := tests.SeedOrder(db, cust.ID, sales.ID, prod.ID)

	t.Run("Success_Update_To_Paid", func(t *testing.T) {
		reqBody := dto.PaymentStatusUpdateRequest{
			PaymentStatus: "paid",
		}
		bodyJson, _ := json.Marshal(reqBody)

		// Perhatikan: Endpoint-nya adalah /payment-status
		req := httptest.NewRequest("PATCH", "/api/orders/"+order.ID+"/payment-status", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Opsional & Sangat Disarankan untuk Integration Test:
		// Verifikasi langsung ke Database apakah nilainya benar-benar berubah
		/*
		   var updatedOrder order_model.OrderModel // Sesuaikan dengan nama model GORM Anda
		   db.First(&updatedOrder, "id = ?", order.ID)
		   assert.Equal(t, "paid", updatedOrder.PaymentStatus)
		*/
	})

	t.Run("Success_Update_To_Partial", func(t *testing.T) {
		reqBody := dto.PaymentStatusUpdateRequest{
			PaymentStatus: "partial",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/api/orders/"+order.ID+"/payment-status", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Failed_Invalid_Status_Enum", func(t *testing.T) {
		reqBody := dto.PaymentStatusUpdateRequest{
			PaymentStatus: "status_ngasal", // Harus ditolak oleh validator "oneof=unpaid partial paid"
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/api/orders/"+order.ID+"/payment-status", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode) // Ekspektasi Error 400
	})
}

func TestOrderItems_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()

	// --- 1. SKENARIO ADD ITEM ---
	t.Run("Success_AddOrderItem", func(t *testing.T) {
		tests.ClearTables(db) // Bersihkan DB sebelum test ini jalan

		sales := tests.SeedUser(db, "S1", "s1@s.com", "sales")
		cust := tests.SeedCustomer(db, "C1", "081", "Jkt")
		cat := tests.SeedCategory(db, "Cat1")

		// Buat 2 produk. Produk 1 disisipkan saat SeedOrder, Produk 2 kita tambahkan via API
		prod1 := tests.SeedProduct(db, cat.ID, "Prod1", 100000)
		prod2 := tests.SeedProduct(db, cat.ID, "Prod2", 25000)

		// Order dibuat dengan 1 item (Prod1 qty 1 = 100.000)
		order := tests.SeedOrder(db, cust.ID, sales.ID, prod1.ID)

		// Payload untuk menambah Prod2 (Qty 2 x 25.000 = 50.000)
		reqBody := dto.OrderItemRequest{
			ProductID: prod2.ID,
			Qty:       2,
			Price:     25000,
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/orders/"+order.ID+"/items", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.OrderResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// ASERSI:
		// 1. Jumlah item di order sekarang harus ada 2
		assert.Len(t, response.Data.Items, 2)
		// 2. Subtotal dan TotalAmount baru harus otomatis terhitung: 100.000 + 50.000 = 150.000

		ongkir := order.ShippingCost

		expectedSubtotal := 150000.0 // 100rb (awal) + 50rb (baru)
		expectedTotal := expectedSubtotal + ongkir

		assert.Equal(t, expectedSubtotal, response.Data.Subtotal)
		assert.Equal(t, expectedTotal, response.Data.TotalAmount)
	})

	// --- 2. SKENARIO UPDATE ITEM ---
	t.Run("Success_UpdateOrderItem", func(t *testing.T) {
		tests.ClearTables(db)
		sales := tests.SeedUser(db, "S2", "s2@s.com", "sales")
		cust := tests.SeedCustomer(db, "C2", "082", "Bdg")
		cat := tests.SeedCategory(db, "Cat2")
		prod := tests.SeedProduct(db, cat.ID, "Prod3", 100000)

		// Order dibuat dengan 1 item (Prod3 qty 1 = 100.000)
		order := tests.SeedOrder(db, cust.ID, sales.ID, prod.ID)

		// Kita perlu mengambil ID dari Item yang baru saja dibuat oleh SeedOrder
		var existingItem postgres.OrderItemModel
		db.Where("order_id = ?", order.ID).First(&existingItem)

		// Payload: Kita ubah qty-nya jadi 5, dan harganya kita diskon jadi 90.000 per item
		reqBody := dto.OrderItemRequest{
			ProductID: prod.ID,
			Qty:       5,
			Price:     90000,
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/orders/"+order.ID+"/items/"+existingItem.ID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.OrderResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// ASERSI:
		// 1. Jumlah item tetap 1 (karena update, bukan tambah baru)
		assert.Len(t, response.Data.Items, 1)
		assert.Equal(t, 5, response.Data.Items[0].Qty)
		assert.Equal(t, 90000.0, response.Data.Items[0].Price)
		// 2. Subtotal baru harus 5 x 90.000 = 450.000
		assert.Equal(t, 450000.0, response.Data.Subtotal)
	})

	// --- 3. SKENARIO DELETE ITEM ---
	t.Run("Success_DeleteOrderItem", func(t *testing.T) {
		tests.ClearTables(db)
		sales := tests.SeedUser(db, "S3", "s3@s.com", "sales")
		cust := tests.SeedCustomer(db, "C3", "083", "Sby")
		cat := tests.SeedCategory(db, "Cat3")
		prod := tests.SeedProduct(db, cat.ID, "Prod4", 100000)

		// Order dibuat dengan 1 item (Prod4 qty 1 = 100.000)
		order := tests.SeedOrder(db, cust.ID, sales.ID, prod.ID)

		// Cari ID Itemnya
		var existingItem postgres.OrderItemModel
		db.Where("order_id = ?", order.ID).First(&existingItem)

		// Hit endpoint DELETE
		req := httptest.NewRequest("DELETE", "/api/orders/"+order.ID+"/items/"+existingItem.ID, nil)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.OrderResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// ASERSI API:
		// Item di keranjang jadi kosong (0), dan Total ter-reset menjadi 0
		assert.Len(t, response.Data.Items, 0)
		assert.Equal(t, 0.0, response.Data.Subtotal)

		// ASERSI DATABASE (Pastikan benar-benar terhapus secara fisik di db)
		var count int64
		db.Model(&postgres.OrderItemModel{}).Where("id = ?", existingItem.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	// --- 4. SKENARIO GAGAL VALIDASI QTY ---
	t.Run("Failed_AddOrderItem_ZeroQty", func(t *testing.T) {
		tests.ClearTables(db)
		sales := tests.SeedUser(db, "S4", "s4@s.com", "sales")
		cust := tests.SeedCustomer(db, "C4", "084", "Sby")
		cat := tests.SeedCategory(db, "Cat4")
		prod := tests.SeedProduct(db, cat.ID, "Prod5", 100000)
		order := tests.SeedOrder(db, cust.ID, sales.ID, prod.ID)

		// Sengaja kirim Qty: 0 agar gagal di DTO Validator
		reqBody := dto.OrderItemRequest{
			ProductID: prod.ID,
			Qty:       0,
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/orders/"+order.ID+"/items", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)

		// Harusnya bad request karena validator menolak Qty 0
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}
