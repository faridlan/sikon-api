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
// 1. TEST REGISTER USER (POST /api/users/register)
// ==========================================
func TestRegisterUser_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	t.Run("Success_Register_Admin", func(t *testing.T) {
		reqBody := dto.UserRegisterRequest{
			Name:     "Budi Admin",
			Email:    "budi@admin.com",
			Password: "rahasia123",
			Role:     "admin",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/users/register", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.UserResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Budi Admin", response.Data.Name)
		assert.Equal(t, "budi@admin.com", response.Data.Email)
		// Pastikan password TIDAK dikembalikan di response!
		assert.NotEmpty(t, response.Data.ID)
	})

	t.Run("Failed_Email_Already_Exists", func(t *testing.T) {
		// 1. Kita buat dulu user dengan email tina@sales.com langsung ke DB
		tests.SeedUser(db, "Tina Asli", "tina@sales.com", "sales")

		// 2. Kita coba register pakai email yang sama
		reqBody := dto.UserRegisterRequest{
			Name:     "Tina Palsu",
			Email:    "tina@sales.com", // <-- Email Duplikat
			Password: "password123",
			Role:     "sales",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/users/register", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)

		// Harusnya Usecase/Repo menolak dan API mengembalikan 409 Conflict
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	})

	t.Run("Failed_Validation_Invalid_Role", func(t *testing.T) {
		reqBody := dto.UserRegisterRequest{
			Name:     "Role Aneh",
			Email:    "aneh@email.com",
			Password: "password123",
			Role:     "superadmin", // <-- Tidak ada di 'oneof=admin sales'
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/users/register", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// ==========================================
// 2. TEST LIST USERS (GET /api/users)
// ==========================================
func TestListUsers_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// Seed 3 data user ke database
	tests.SeedUser(db, "User Satu", "user1@sikon.com", "sales")
	tests.SeedUser(db, "User Dua", "user2@sikon.com", "admin")
	tests.SeedUser(db, "User Tiga", "user3@sikon.com", "sales")

	t.Run("Success_GetList", func(t *testing.T) {
		// Tembak endpoint dengan query parameter page=1 dan limit=10
		req := httptest.NewRequest("GET", "/api/users?page=1&limit=10", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Karena SendSuccessPaginated biasanya mengembalikan struktur JSON yang sedikit berbeda
		// (ada object 'meta' untuk pagination), kita buat struct anonim untuk menampungnya
		var response struct {
			Message string             `json:"message"`
			Data    []dto.UserResponse `json:"data"`
			Meta    any                `json:"meta"`
		}

		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Verifikasi bahwa ada 3 data user yang dikembalikan
		assert.Len(t, response.Data, 3)
		assert.Equal(t, "Berhasil mengambil daftar user", response.Message)
		assert.NotNil(t, response.Meta) // Pastikan object meta tidak nil
	})
}

// ==========================================
// 3. TEST GET USER PROFILE (GET /api/users/:id)
// ==========================================
func TestGetUser_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "Andi", "andi@sikon.com", "sales")

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/"+user.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.UserResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, user.ID, response.Data.ID)
		assert.Equal(t, "Andi", response.Data.Name)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := httptest.NewRequest("GET", "/api/users/"+randomID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// ==========================================
// 4. TEST UPDATE USER (PUT /api/users/:id)
// ==========================================
func TestUpdateUser_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "Siti Lama", "siti@sikon.com", "sales")

	t.Run("Success", func(t *testing.T) {
		// Update name dan role menjadi admin
		reqBody := dto.UserUpdateRequest{
			Name: "Siti Manajer",
			Role: "admin",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/users/"+user.ID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.UserResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Siti Manajer", response.Data.Name)
		assert.Equal(t, "admin", response.Data.Role)
	})

	t.Run("Failed_Validation_Invalid_Role", func(t *testing.T) {
		reqBody := dto.UserUpdateRequest{Role: "bos_besar"} // Role salah
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/users/"+user.ID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// ==========================================
// 5. TEST DELETE USER (DELETE /api/users/:id)
// ==========================================
func TestDeleteUser_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "User Resign", "resign@sikon.com", "sales")

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/users/"+user.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Verifikasi Terhapus
		reqCheck := httptest.NewRequest("GET", "/api/users/"+user.ID, nil)
		respCheck, _ := app.Test(reqCheck, -1)
		assert.Equal(t, fiber.StatusNotFound, respCheck.StatusCode)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := httptest.NewRequest("DELETE", "/api/users/"+randomID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
