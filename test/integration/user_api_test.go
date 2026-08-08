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
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/repository/postgres"
	"github.com/faridlan/sikon-api/internal/utils"
	tests "github.com/faridlan/sikon-api/test"
)

// ==========================================
// 1. TEST REGISTER USER (POST /api/users/register)
// ==========================================
func TestRegisterUser_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)
	token := tests.GenerateTestJWTToken("user-owner-123", "owner@sikon.com", domain.RoleOwner)

	t.Run("Success_Register_Admin", func(t *testing.T) {
		reqBody := dto.UserRegisterRequest{
			Name:     "Budi Admin",
			Email:    "budi@admin.com",
			Password: "rahasia123",
			Role:     "accounting",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/users/register", bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.UserResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Budi Admin", response.Data.Name)
		assert.Equal(t, "budi@admin.com", response.Data.Email)
		assert.NotEmpty(t, response.Data.ID)
	})

	t.Run("Success_Register_Sales_With_WhatsApp", func(t *testing.T) {
		reqBody := dto.UserRegisterRequest{
			Name:       "Rina Sales",
			Email:      "rina@sales.com",
			Password:   "rahasia123",
			Role:       "sales",
			Phone:      "6281200000001",
			StatusText: "Online sekarang",
			ImageURL:   "https://example.com/rina.jpg",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/users/register", bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.UserResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Rina Sales", response.Data.Name)
		assert.Equal(t, "rina@sales.com", response.Data.Email)
		assert.Equal(t, "6281200000001", response.Data.Phone)
		assert.Equal(t, "Online sekarang", response.Data.StatusText)
		assert.True(t, response.Data.IsActive)
	})

	t.Run("Failed_Email_Already_Exists", func(t *testing.T) {
		tests.SeedUser(db, "Tina Asli", "tina@sales.com", "sales")

		reqBody := dto.UserRegisterRequest{
			Name:     "Tina Palsu",
			Email:    "tina@sales.com",
			Password: "password123",
			Role:     "sales",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/users/register", bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	})

	t.Run("Failed_Validation_Invalid_Role", func(t *testing.T) {
		reqBody := dto.UserRegisterRequest{
			Name:     "Role Aneh",
			Email:    "aneh@email.com",
			Password: "password123",
			Role:     "superadmin",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/users/register", bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// ==========================================
// 2. TEST GET PUBLIC SALES DIRECTORY (GET /api/users/public/sales)
// ==========================================
func TestGetPublicSales_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// 1. Seed sales aktif
	sales1 := tests.SeedUser(db, "Rina Pratiwi", "rina@sikon.com", "sales", "https://example.com/rina.jpg")
	db.Model(&postgres.UserModel{}).Where("id = ?", sales1.ID).Updates(map[string]any{
		"phone":       "6281200000001",
		"status_text": "Online sekarang",
		"is_active":   true,
		"sort_order":  1,
	})

	sales2 := tests.SeedUser(db, "Dimas Aditya", "dimas@sikon.com", "sales", "https://example.com/dimas.jpg")
	db.Model(&postgres.UserModel{}).Where("id = ?", sales2.ID).Updates(map[string]any{
		"phone":       "6281200000002",
		"status_text": "Balas dalam 1 jam",
		"is_active":   true,
		"sort_order":  2,
	})

	// 2. Seed sales non-aktif (tidak boleh muncul di response)
	salesInactive := tests.SeedUser(db, "Sales Nonaktif", "off@sikon.com", "sales")
	db.Model(&postgres.UserModel{}).Where("id = ?", salesInactive.ID).Update("is_active", false)

	// 3. Seed accounting (tidak boleh muncul di response public sales)
	tests.SeedUser(db, "Admin SIKOn", "admin@sikon.com", "accounting")

	t.Run("Success_Get_Public_Sales_Directory", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/users/public/sales", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[[]dto.PublicSalesResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Harus hanya mengembalikan 2 sales aktif
		assert.Len(t, response.Data, 2)

		// Verifikasi urutan sort_order & format link WhatsApp
		assert.Equal(t, "Rina Pratiwi", response.Data[0].Name)
		assert.Equal(t, "6281200000001", response.Data[0].Phone)
		assert.Contains(t, response.Data[0].WhatsAppURL, "api.whatsapp.com/send/?phone=6281200000001")

		assert.Equal(t, "Dimas Aditya", response.Data[1].Name)
		assert.Equal(t, "6281200000002", response.Data[1].Phone)
	})
}

// ==========================================
// 3. TEST LIST USERS (GET /api/users)
// ==========================================
func TestListUsers_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	tests.SeedUser(db, "User Satu", "user1@sikon.com", "sales", "https://example.com/image1.jpg")
	tests.SeedUser(db, "User Dua", "user2@sikon.com", "accounting", "https://example.com/image2.jpg")
	tests.SeedUser(db, "User Tiga", "user3@sikon.com", "sales", "https://example.com/image3.jpg")

	t.Run("Success_GetList_TanpaFilter", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/users?page=1&limit=10", nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Message string             `json:"message"`
			Data    []dto.UserResponse `json:"data"`
			Meta    any                `json:"meta"`
		}

		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Len(t, response.Data, 4)
		assert.NotNil(t, response.Meta)
	})

	t.Run("Success_GetList_FilterRoleSales", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/users?role=sales", nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Data []dto.UserResponse `json:"data"`
		}

		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Len(t, response.Data, 2)
		assert.Equal(t, "sales", response.Data[0].Role)
	})
}

// ==========================================
// 4. TEST GET USER PROFILE (GET /api/users/:id)
// ==========================================
func TestGetUser_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "Andi", "andi@sikon.com", "sales", "https://example.com/andi.jpg")

	t.Run("Success", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/users/"+user.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
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
		req := tests.AuthenticatedRequest("GET", "/api/users/"+randomID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// ==========================================
// 5. TEST UPDATE USER (PUT /api/users/:id)
// ==========================================
func TestUpdateUser_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "Siti Lama", "siti@sikon.com", "sales")

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.UserUpdateRequest{
			Name:       "Siti Manajer",
			Role:       "accounting",
			Phone:      "6281233334444",
			StatusText: "Balas cepat",
			ImageURL:   "https://example.com/new_image.jpg",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("PUT", "/api/users/"+user.ID, bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.UserResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Siti Manajer", response.Data.Name)
		assert.Equal(t, "accounting", response.Data.Role)
		assert.Equal(t, "6281233334444", response.Data.Phone)
		assert.Equal(t, "Balas cepat", response.Data.StatusText)
	})

	t.Run("Failed_Validation_Invalid_Role", func(t *testing.T) {
		reqBody := dto.UserUpdateRequest{Role: "bos_besar"}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("PUT", "/api/users/"+user.ID, bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// ==========================================
// 6. TEST DELETE USER (DELETE /api/users/:id)
// ==========================================
func TestDeleteUser_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "User Resign", "resign@sikon.com", "sales")

	t.Run("Success", func(t *testing.T) {
		req := tests.AuthenticatedRequest("DELETE", "/api/users/"+user.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		reqCheck := tests.AuthenticatedRequest("GET", "/api/users/"+user.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		respCheck, _ := app.Test(reqCheck, -1)
		assert.Equal(t, fiber.StatusNotFound, respCheck.StatusCode)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := tests.AuthenticatedRequest("DELETE", "/api/users/"+randomID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
