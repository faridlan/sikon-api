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
	"github.com/faridlan/sikon-api/internal/repository/postgres"
	"github.com/faridlan/sikon-api/internal/utils"
	tests "github.com/faridlan/sikon-api/test"
)

// ==========================================
// 1. TEST CREATE PRODUCT (POST /api/products)
// ==========================================
func TestCreateProduct_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := tests.SeedCategory(db, "Kaos Sablon")

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.ProductCreateRequest{
			CategoryID:  category.ID,
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

	// 🚨 TAMBAHAN: Test Array Gambar
	t.Run("Success_With_Multiple_Images", func(t *testing.T) {
		reqBody := dto.ProductCreateRequest{
			CategoryID:  category.ID,
			Name:        "Kaos Polos Lengan Panjang",
			Description: "Bahan Combed 30s",
			BasePrice:   65000,
			ImageURLs:   []string{"https://example.com/depan.jpg", "https://example.com/belakang.jpg"},
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

		assert.Equal(t, "Kaos Polos Lengan Panjang", response.Data.Name)

		// Verifikasi bahwa 2 gambar tersimpan dan ter-mapping dengan benar
		assert.Equal(t, 2, len(response.Data.Images))
		assert.Equal(t, "https://example.com/depan.jpg", response.Data.Images[0].ImageURL)
		assert.True(t, response.Data.Images[0].IsPrimary) // Gambar pertama harus Primary
		assert.Equal(t, "https://example.com/belakang.jpg", response.Data.Images[1].ImageURL)
		assert.False(t, response.Data.Images[1].IsPrimary) // Gambar kedua bukan Primary
	})

	t.Run("Failed_Category_Not_Found", func(t *testing.T) {
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
		assert.NotEqual(t, fiber.StatusCreated, resp.StatusCode)
	})

	t.Run("Failed_Validation_Price_Zero", func(t *testing.T) {
		reqBody := dto.ProductCreateRequest{
			CategoryID: category.ID,
			Name:       "Barang Gratis",
			BasePrice:  0,
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

	// 1. Siapkan Data Seeder
	catAtasan := tests.SeedCategory(db, "Atasan")
	catBawahan := tests.SeedCategory(db, "Bawahan")

	// Produk Atasan
	tests.SeedProduct(db, catAtasan.ID, "Kemeja Polos Hitam", 100000)
	tests.SeedProduct(db, catAtasan.ID, "Kaos Sablon Premium", 75000)

	// Produk Bawahan
	tests.SeedProduct(db, catBawahan.ID, "Celana Jeans Denim", 150000)

	t.Run("Success_GetAll_TanpaFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/products", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Verifikasi total data harus 3
		var body map[string]any
		json.NewDecoder(resp.Body).Decode(&body)

		meta := body["meta"].(map[string]any)
		assert.Equal(t, float64(3), meta["total_items"]) // JSON number di-parse sebagai float64
	})

	t.Run("Success_FilterBySearch_SatuKata", func(t *testing.T) {
		// Cari kata "Kemeja" (case insensitive)
		req := httptest.NewRequest("GET", "/api/products?search=kemeja", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body map[string]any
		json.NewDecoder(resp.Body).Decode(&body)

		data := body["data"].([]any)
		meta := body["meta"].(map[string]any)

		assert.Len(t, data, 1) // Hanya ada 1 kemeja
		assert.Equal(t, float64(1), meta["total_items"])
	})

	t.Run("Success_FilterByCategory", func(t *testing.T) {
		// Filter semua produk dengan kategori "Atasan"
		req := httptest.NewRequest("GET", "/api/products?category_id="+catAtasan.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body map[string]any
		json.NewDecoder(resp.Body).Decode(&body)

		data := body["data"].([]any)
		assert.Len(t, data, 2) // Kemeja dan Kaos
	})

	t.Run("Success_KombinasiSearchDanCategory", func(t *testing.T) {
		// Cari kata "Celana" tapi memaksakan ID Kategorinya adalah "Atasan"
		// Harusnya tidak ketemu, karena Celana ada di kategori Bawahan
		req := httptest.NewRequest("GET", "/api/products?search=Celana&category_id="+catAtasan.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body map[string]any
		json.NewDecoder(resp.Body).Decode(&body)

		data := body["data"].([]any)
		assert.Len(t, data, 0) // Kosong (Empty State)
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

	t.Run("Success_Update_Image", func(t *testing.T) {
		// 🚨 Buat gambar lama secara manual di tabel relasi product_images
		// (Pastikan kamu meng-import package postgres tempat modelmu berada)
		oldImage := postgres.ProductImageModel{
			ProductID: product.ID,
			ImageURL:  "https://example.com/gambar-lama.jpg",
			IsPrimary: true,
		}
		db.Create(&oldImage)

		// Request update dengan daftar gambar baru
		reqBody := dto.ProductUpdateRequest{
			ImageURLs: []string{"https://example.com/gambar-baru.jpg"}, // Mengganti gambar lama
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/products/"+product.ID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.ProductResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Verifikasi response memiliki URL array yang baru dan hanya ada 1 gambar
		assert.Equal(t, 1, len(response.Data.Images))
		assert.Equal(t, "https://example.com/gambar-baru.jpg", response.Data.Images[0].ImageURL)
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

	// 🚨 TAMBAHAN: Skenario Hapus Produk dengan Gambar
	t.Run("Success_With_Image", func(t *testing.T) {
		// Buat produk baru khusus untuk tes ini
		productWithImage := tests.SeedProduct(db, category.ID, "Topi Taktikal", 45000)
		db.Model(&productWithImage).Update("image_url", "https://example.com/topi.jpg")

		req := httptest.NewRequest("DELETE", "/api/products/"+productWithImage.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode) // API harus tetap membalas 200 dengan cepat
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := httptest.NewRequest("DELETE", "/api/products/"+randomID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
