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

	// 1. SKENARIO CREATE SURAT PENAWARAN (Default Status)
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
			OrderStatus:     "quotation", // <-- Frontend mengirim status quotation untuk membuat Surat Penawaran
			Items: []dto.OrderItemRequest{
				{
					ProductID: product.ID,
					Qty:       2,
					Price:     150000,
					Details:   map[string]any{"Benang": "Benang Bordir Menggunakan Benang Polyster", "Bordir": "Bordir Menggunakan Sistem Komputerisasi", "Jahitan": "Jahit Rapi", "Bahan": map[string]any{"Name": "Katun Baby Canvas", "Spec": "menggunakan baby canvas", "Color": "Hitam"}}, // Contoh variasi produk yang lebih kompleks
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
	})

	// 2. SKENARIO BARU: CREATE ORDER LANGSUNG (Status Pending dari Frontend)
	t.Run("Success_Create_Order_Directly_As_Pending", func(t *testing.T) {
		reqBody := dto.OrderCreateRequest{
			CustomerID:      customer.ID,
			SalesID:         sales.ID,
			OrderStatus:     "pending", // <-- Frontend secara eksplisit mengirim status pending
			ShippingCost:    20000,
			CourierName:     "JNT",
			ShippingAddress: "Alamat Langsung",
			Items: []dto.OrderItemRequest{
				{
					ProductID: product.ID,
					Qty:       1,
					Price:     150000,
					Details:   map[string]any{"Benang": "Benang Bordir Menggunakan Benang Polyster", "Bordir": "Bordir Menggunakan Sistem Komputerisasi", "Jahitan": "Jahit Rapi", "Bahan": map[string]any{"Name": "Katun Baby Canvas", "Spec": "menggunakan baby canvas", "Color": "Hitam"}}, // Contoh variasi produk yang lebih kompleks
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

		// PASTIKAN: Sistem mencatatnya sebagai 'pending' (Pesanan Resmi), bukan 'quotation'
		assert.Equal(t, "pending", response.Data.OrderStatus)
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

		var response utils.ErrorResponse
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)
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
			ShippingCost:    50000,
			CourierName:     "SiCepat",
			ShippingAddress: "Alamat Update",
			Notes:           "Catatan Update",
			ValidUntil:      &validUntilUpdate,     // <-- TAMBAHAN
			TermsConditions: "Lunas Sebelum Kirim", // <-- TAMBAHAN
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

	t.Run("Success_Update_To_Production", func(t *testing.T) {
		reqBody := dto.OrderStatusUpdateRequest{
			OrderStatus: "production",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/api/orders/"+order.ID+"/status", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Success_Update_To_Quotation", func(t *testing.T) {
		reqBody := dto.OrderStatusUpdateRequest{
			OrderStatus: "quotation", // Status valid lainnya
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/api/orders/"+order.ID+"/status", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Failed_Invalid_Status_Enum", func(t *testing.T) {
		reqBody := dto.OrderStatusUpdateRequest{
			OrderStatus: "status_ngasal", // Harus ditolak oleh validator "oneof"
		}
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
