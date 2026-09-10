package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
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

// setupOrderHPPFixture menyiapkan data dasar: kategori, produk dengan resep BOM,
// customer, sales, dan Batch PO aktif. Order-nya sendiri dibuat per test case
// karena tiap test butuh skenario qty/produk yang beda.
func setupOrderHPPFixture(db *gorm.DB) (product postgres.ProductModel, customer postgres.CustomerModel, sales postgres.UserModel, batchPO postgres.BatchPOModel) {
	category := tests.SeedCategory(db, "Kaos")
	product = tests.SeedProduct(db, category.ID, "Kaos Polo", 150000)

	// Resep: 1.2 meter kain @ Rp25.000 + 3 kancing @ Rp1.000 = Rp30.000 + Rp3.000 = Rp33.000 per pcs
	material1 := tests.SeedMaterial(db, "Kain Ripstop", "meter", 25000, "kain")
	material2 := tests.SeedMaterial(db, "Kancing", "pcs", 1000, "aksesoris")
	tests.SeedProductMaterial(db, product.ID, material1.ID, 1.2)
	tests.SeedProductMaterial(db, product.ID, material2.ID, 3)

	sales = tests.SeedUser(db, "Sales Dummy", "sales-hpp@sikon.com", "sales")
	customer = tests.SeedCustomer(db, "Customer Dummy", "081234567890", "Jl. Dummy No.1")
	batchPO = tests.SeedBatchPO(db, "PO HPP Test", "active")

	return
}

func TestOrderApprove_MembekukanHPPMaterial_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	product, customer, sales, batchPO := setupOrderHPPFixture(db)

	t.Run("Success - HPP Material Ter-snapshot Sesuai Resep x Qty", func(t *testing.T) {
		// SeedOrder membuat 1 item dengan Qty=2 untuk productID yang diberikan
		order := tests.SeedOrder(db, batchPO.ID, customer.ID, sales.ID, product.ID)

		// Order butuh PaymentStatus != unpaid supaya lolos state machine Quotation -> Pending
		db.Model(&postgres.OrderModel{}).Where("id = ?", order.ID).Update("payment_status", "partial")

		reqBody := dto.OrderStatusUpdateRequest{OrderStatus: "pending"}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("PATCH", "/api/orders/"+order.ID+"/status", bytes.NewBuffer(body), "test-sales", "sales-approver@sikon.com", domain.RoleSales)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Verifikasi langsung ke database: HPP Material harus 33.000/pcs x 2 pcs = 66.000
		var updated postgres.OrderModel
		db.Where("id = ?", order.ID).First(&updated)

		assert.Equal(t, "pending", updated.OrderStatus)
		assert.Equal(t, float64(66000), updated.HPPMaterialCost)
		assert.NotNil(t, updated.HPPCalculatedAt)
	})

	t.Run("Gagal Approve Kalau Masih Unpaid, HPP Tidak Ikut Dihitung", func(t *testing.T) {
		order := tests.SeedOrder(db, batchPO.ID, customer.ID, sales.ID, product.ID) // default payment_status = unpaid

		reqBody := dto.OrderStatusUpdateRequest{OrderStatus: "pending"}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("PATCH", "/api/orders/"+order.ID+"/status", bytes.NewBuffer(body), "test-sales", "sales-approver@sikon.com", domain.RoleSales)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)

		var stillQuotation postgres.OrderModel
		db.Where("id = ?", order.ID).First(&stillQuotation)
		assert.Equal(t, "quotation", stillQuotation.OrderStatus)
		assert.Equal(t, float64(0), stillQuotation.HPPMaterialCost) // Tidak sempat dihitung sama sekali
	})
}

func TestGetOrderHPP_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	product, customer, sales, batchPO := setupOrderHPPFixture(db)

	t.Run("Success - Material Beku + Labor 0 (Belum Ada Work Log)", func(t *testing.T) {
		order := tests.SeedOrder(db, batchPO.ID, customer.ID, sales.ID, product.ID)
		db.Model(&postgres.OrderModel{}).Where("id = ?", order.ID).
			Updates(map[string]any{"hpp_material_cost": 66000, "hpp_calculated_at": time.Now()})

		req := tests.AuthenticatedRequest("GET", "/api/orders/"+order.ID+"/hpp", nil, "test-accounting", "accounting-hpp@sikon.com", domain.RoleAccounting)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.OrderHPPResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, float64(66000), response.Data.MaterialCost)
		assert.Equal(t, float64(0), response.Data.LaborCost)
		assert.Equal(t, float64(66000), response.Data.TotalCost)
	})

	t.Run("Success - Material Beku + Labor Live Dari Work Log Aktual", func(t *testing.T) {
		order := tests.SeedOrder(db, batchPO.ID, customer.ID, sales.ID, product.ID)
		db.Model(&postgres.OrderModel{}).Where("id = ?", order.ID).
			Updates(map[string]any{"hpp_material_cost": 66000, "hpp_calculated_at": time.Now()})

		// Buat worker + 2 baris Work Log langsung (belum ada helper Seed khusus untuk ini)
		worker := postgres.WorkerModel{
			ID: uuid.New().String(), Name: "Tukang Jahit Dummy", Role: "jahit",
			SalaryType: "borongan", Status: "active",
		}
		db.Create(&worker)

		db.Create(&postgres.WorkLogModel{
			ID: uuid.New().String(), WorkerID: worker.ID, OrderID: &order.ID,
			JobType: "jahit", Qty: 2, RatePerQty: 10000, TotalAmount: 20000,
			WorkDate: time.Now(), CreatedByID: sales.ID,
		})
		db.Create(&postgres.WorkLogModel{
			ID: uuid.New().String(), WorkerID: worker.ID, OrderID: &order.ID,
			JobType: "potong", Qty: 2, RatePerQty: 5000, TotalAmount: 10000,
			WorkDate: time.Now(), CreatedByID: sales.ID,
		})

		req := tests.AuthenticatedRequest("GET", "/api/orders/"+order.ID+"/hpp", nil, "test-accounting", "accounting-hpp@sikon.com", domain.RoleAccounting)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.OrderHPPResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, float64(66000), response.Data.MaterialCost) // tetap beku, tidak ikut berubah
		assert.Equal(t, float64(30000), response.Data.LaborCost)    // 20.000 + 10.000, live dari Work Log
		assert.Equal(t, float64(96000), response.Data.TotalCost)    // 66.000 + 30.000
	})

	t.Run("Failed - Order Tidak Ditemukan", func(t *testing.T) {
		randomID := uuid.New().String()
		req := tests.AuthenticatedRequest("GET", "/api/orders/"+randomID+"/hpp", nil, "test-accounting", "accounting-hpp@sikon.com", domain.RoleAccounting)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
