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
// 1. TEST CREATE CATEGORY (POST /api/categories)
// ==========================================
func TestCreateCategory_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.CategoryRequest{Name: "Kain Katun Premium", ImageURL: "https://example.com/kain-katun.jpg"}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/categories", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.CategoryResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Kain Katun Premium", response.Data.Name)
		assert.Equal(t, "https://example.com/kain-katun.jpg", response.Data.ImageURL)
		assert.NotEmpty(t, response.Data.ID)
	})

	t.Run("Failed_Validation", func(t *testing.T) {
		reqBody := dto.CategoryRequest{Name: ""} // Name kosong (harus error)
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/categories", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// ==========================================
// 2. TEST GET CATEGORY (GET /api/categories/:id)
// ==========================================
func TestGetCategory_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// Seed data kategori ke DB terlebih dahulu
	category := tests.SeedCategory(db, "Kemeja Flanel", "https://example.com/flanel.jpg")

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/categories/"+category.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.CategoryResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, category.ID, response.Data.ID)
		assert.Equal(t, "Kemeja Flanel", response.Data.Name)
		assert.Equal(t, "https://example.com/flanel.jpg", response.Data.ImageURL)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String() // Generate UUID acak yang tidak ada di DB
		req := httptest.NewRequest("GET", "/api/categories/"+randomID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// ==========================================
// 3. TEST LIST CATEGORIES (GET /api/categories)
// ==========================================
func TestListCategories_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// Seed 3 data kategori
	tests.SeedCategory(db, "Kategori 1")
	tests.SeedCategory(db, "Kategori 2")
	tests.SeedCategory(db, "Kategori 3")

	t.Run("Success_GetList", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/categories?page=1&limit=10", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Gunakan struct anonim jika Utils tidak punya generic untuk pagination,
		// atau gunakan utils.PaginatedResponse jika sudah ada.
		var response struct {
			Message string                 `json:"message"`
			Data    []dto.CategoryResponse `json:"data"`
			Meta    any                    `json:"meta"`
		}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Len(t, response.Data, 3) // Pastikan ada 3 data yang kembali
	})
}

// ==========================================
// 4. TEST UPDATE CATEGORY (PUT /api/categories/:id)
// ==========================================
func TestUpdateCategory_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := tests.SeedCategory(db, "Nama Lama", "https://example.com/old.jpg")

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.CategoryRequest{Name: "Nama Baru Terupdate", ImageURL: "https://example.com/updated-nama.jpg"}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/categories/"+category.ID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.CategoryResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Nama Baru Terupdate", response.Data.Name)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String() // ID acak yang pasti tidak ada di DB
		reqBody := dto.CategoryRequest{Name: "Coba Update Data Gaib"}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/categories/"+randomID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode) // Harusnya 404
	})

	t.Run("Failed_Validation_Body", func(t *testing.T) {
		reqBody := dto.CategoryRequest{Name: ""} // Nama tidak boleh kosong
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/categories/"+category.ID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode) // Harusnya 400
	})

	t.Run("Failed_Invalid_UUID", func(t *testing.T) {
		reqBody := dto.CategoryRequest{Name: "Test Invalid ID"}
		bodyJson, _ := json.Marshal(reqBody)

		// Tembak dengan format ID yang salah
		req := httptest.NewRequest("PUT", "/api/categories/id-bukan-uuid", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode) // Harusnya 400 dicegat oleh utils.ValidateUUID
	})
}

// ==========================================
// 5. TEST DELETE CATEGORY (DELETE /api/categories/:id)
// ==========================================
func TestDeleteCategory_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := tests.SeedCategory(db, "Kategori Untuk Dihapus")

	t.Run("Success", func(t *testing.T) {
		// 1. Eksekusi Delete
		req := httptest.NewRequest("DELETE", "/api/categories/"+category.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// 2. Verifikasi data benar-benar hilang (Tembak GET by ID)
		reqCheck := httptest.NewRequest("GET", "/api/categories/"+category.ID, nil)
		respCheck, _ := app.Test(reqCheck, -1)

		// Harusnya API mengembalikan 404 NotFound karena data sudah dihapus
		assert.Equal(t, fiber.StatusNotFound, respCheck.StatusCode)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := httptest.NewRequest("DELETE", "/api/categories/"+randomID, nil)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode) // Harusnya 404
	})

	t.Run("Failed_Invalid_UUID", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/categories/ini-id-ngasal", nil)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode) // Harusnya 400
	})
}
