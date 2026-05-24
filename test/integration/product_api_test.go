package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/utils"
	tests "github.com/faridlan/sikon-api/test"
)

// ==========================================
// 1. TEST CREATE PRODUCT (POST /api/products)
// ==========================================
func TestCreateProduct_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// Persiapan: Buat data kategori yang valid
	category := tests.SeedCategory(db, "Kaos Sablon")

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.ProductCreateRequest{
			CategoryID:  category.ID, // Menggunakan ID yang valid dari seeder
			Name:        "Kaos Sablon Custom",
			Description: "Bahan Combed 30s",
			BasePrice:   55000,
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/products", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.ProductResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Kaos Sablon Custom", response.Data.Name)
		assert.Equal(t, category.ID, response.Data.CategoryID)
		assert.NotEmpty(t, response.Data.ID)
	})

	t.Run("Failed_Category_Not_Found", func(t *testing.T) {
		// Menggunakan UUID yang valid secara format, tapi TIDAK ADA di database
		randomCatID := uuid.New().String()

		reqBody := dto.ProductCreateRequest{
			CategoryID: randomCatID,
			Name:       "Produk Gaib",
			BasePrice:  10000,
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/products", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)

		// Harusnya 404 atau 400 (tergantung bagaimana Usecase Anda menangani error Foreign Key)
		// Jika Usecase Anda melakukan GetCategoryByID dulu, biasanya akan me-return 404
		// Jika Usecase Anda langsung insert dan mengandalkan error GORM constraint, mungkin 400/500
		assert.NotEqual(t, fiber.StatusCreated, resp.StatusCode)
	})

	t.Run("Failed_Validation_Price_Zero", func(t *testing.T) {
		reqBody := dto.ProductCreateRequest{
			CategoryID: category.ID,
			Name:       "Barang Gratis",
			BasePrice:  0, // Validator "gt=0" harusnya menolak ini
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/products", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// ==========================================
// 2. TEST GET PRODUCT (GET /api/products/:id)
// ==========================================
func TestGetProduct_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := tests.SeedCategory(db, "Topi")
	product := tests.SeedProduct(db, category.ID, "Topi Baseball", 25000)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/products/"+product.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.ProductResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, product.ID, response.Data.ID)
		assert.Equal(t, "Topi Baseball", response.Data.Name)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := httptest.NewRequest("GET", "/api/products/"+randomID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// ==========================================
// 3. TEST LIST PRODUCTS (GET /api/products)
// ==========================================
func TestListProducts_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := tests.SeedCategory(db, "Jaket")
	tests.SeedProduct(db, category.ID, "Jaket Bomber", 150000)
	tests.SeedProduct(db, category.ID, "Jaket Parka", 175000)

	t.Run("Success_GetList", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/products", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Data []dto.ProductResponse `json:"data"`
		}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Len(t, response.Data, 2)
	})
}

// ==========================================
// 4. TEST UPDATE PRODUCT (PUT /api/products/:id)
// ==========================================
func TestUpdateProduct_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := tests.SeedCategory(db, "Celana")
	product := tests.SeedProduct(db, category.ID, "Celana Jeans Lama", 100000)

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.ProductUpdateRequest{
			CategoryID: category.ID,
			Name:       "Celana Jeans Baru",
			BasePrice:  120000,
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/products/"+product.ID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		reqBody := dto.ProductUpdateRequest{Name: "Update Gaib"}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/products/"+randomID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// ==========================================
// 5. TEST DELETE PRODUCT (DELETE /api/products/:id)
// ==========================================
func TestDeleteProduct_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := tests.SeedCategory(db, "Aksesoris")
	product := tests.SeedProduct(db, category.ID, "Gantungan Kunci", 5000)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/products/"+product.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Verifikasi Terhapus
		reqCheck := httptest.NewRequest("GET", "/api/products/"+product.ID, nil)
		respCheck, _ := app.Test(reqCheck, -1)
		assert.Equal(t, fiber.StatusNotFound, respCheck.StatusCode)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := httptest.NewRequest("DELETE", "/api/products/"+randomID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
