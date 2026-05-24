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
// 1. TEST CREATE BANK ACCOUNT (POST /api/bank-accounts)
// ==========================================
func TestCreateBankAccount_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "Sales Budi", "budi@sikon.com", "sales")

	t.Run("Success_Create_Global_Account", func(t *testing.T) {
		reqBody := dto.BankAccountCreateRequest{
			UserID:        nil, // Nil berarti ini rekening perusahaan
			BankName:      "BCA",
			AccountNumber: "1234567890",
			AccountName:   "PT SIKOn Global",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/bank-accounts", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.BankAccountResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Nil(t, response.Data.UserID)
		assert.Equal(t, "PT SIKOn Global", response.Data.AccountName)
	})

	t.Run("Success_Create_Sales_Account", func(t *testing.T) {
		reqBody := dto.BankAccountCreateRequest{
			UserID:        tests.StringPtr(user.ID), // Memasukkan ID User Sales
			BankName:      "Mandiri",
			AccountNumber: "0987654321",
			AccountName:   "Budi Haryanto",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/bank-accounts", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.BankAccountResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.NotNil(t, response.Data.UserID)
		assert.Equal(t, user.ID, *response.Data.UserID)
	})

	// --- SKENARIO GAGAL (NEGATIVE PATH) ---

	t.Run("Failed_Validation_Missing_Required_Fields", func(t *testing.T) {
		reqBody := dto.BankAccountCreateRequest{
			UserID:   nil,
			BankName: "BCA",
			// AccountNumber dan AccountName sengaja dikosongkan (padahal required)
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/bank-accounts", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Failed_Invalid_UserID_Format", func(t *testing.T) {
		reqBody := dto.BankAccountCreateRequest{
			UserID:        tests.StringPtr("bukan-uuid-valid"), // Harus ditolak oleh validator "omitempty,uuid"
			BankName:      "BRI",
			AccountNumber: "111",
			AccountName:   "Ngasal",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/bank-accounts", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// ==========================================
// 2. TEST UPDATE BANK ACCOUNT (PUT /api/bank-accounts/:id)
// ==========================================
func TestUpdateBankAccount_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	account := tests.SeedBankAccount(db, nil, "BRI", "111", "Nama Lama")

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.BankAccountUpdateRequest{
			BankName:      "BRI Update",
			AccountNumber: "222",
			AccountName:   "Nama Baru",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/bank-accounts/"+account.ID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.BankAccountResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Nama Baru", response.Data.AccountName)
	})

	// --- SKENARIO GAGAL (NEGATIVE PATH) ---

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String() // UUID yang tidak ada di DB
		reqBody := dto.BankAccountUpdateRequest{
			BankName: "Bank Gaib",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/bank-accounts/"+randomID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode) // Harusnya 404
	})

	t.Run("Failed_Invalid_UUID", func(t *testing.T) {
		reqBody := dto.BankAccountUpdateRequest{
			BankName: "Bank Gaib",
		}
		bodyJson, _ := json.Marshal(reqBody)

		// Tembak dengan format ID di URL yang salah
		req := httptest.NewRequest("PUT", "/api/bank-accounts/id-ngasal", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode) // Harusnya 400
	})
}

// ==========================================
// 3. TEST GET GLOBAL ACCOUNTS (GET /api/bank-accounts/global)
// ==========================================
func TestGetGlobalAccounts_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "Sales A", "a@sikon.com", "sales")

	// Kita sengaja mencampur datanya
	// 2 Rekening Global (UserID nil)
	tests.SeedBankAccount(db, nil, "BCA", "111", "Global 1")
	tests.SeedBankAccount(db, nil, "BNI", "222", "Global 2")

	// 1 Rekening Sales (UserID tidak nil)
	tests.SeedBankAccount(db, tests.StringPtr(user.ID), "Mandiri", "333", "Sales 1")

	t.Run("Success_Only_Return_Global", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/bank-accounts/global", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[[]dto.BankAccountResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Harus hanya mengembalikan 2 data, karena yang ke-3 milik sales
		assert.Len(t, response.Data, 2)

		// Pastikan semua data yang dikembalikan UserID-nya nil
		for _, acc := range response.Data {
			assert.Nil(t, acc.UserID)
		}
	})
}

// ==========================================
// 4. TEST LIST ALL ACCOUNTS (GET /api/bank-accounts)
// ==========================================
func TestListBankAccounts_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	tests.SeedBankAccount(db, nil, "BCA", "111", "Global")
	user := tests.SeedUser(db, "Sales B", "b@sikon.com", "sales")
	tests.SeedBankAccount(db, tests.StringPtr(user.ID), "BRI", "222", "Sales B")

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/bank-accounts", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Data []dto.BankAccountResponse `json:"data"`
		}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Len(t, response.Data, 2) // Semua ditarik
	})
}

// ==========================================
// 5. TEST DELETE BANK ACCOUNT (DELETE /api/bank-accounts/:id)
// ==========================================
func TestDeleteBankAccount_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	account := tests.SeedBankAccount(db, nil, "Hapus Bank", "000", "Akan Dihapus")

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/bank-accounts/"+account.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := httptest.NewRequest("DELETE", "/api/bank-accounts/"+randomID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
