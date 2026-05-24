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
// 1. TEST CREATE CUSTOMER (POST /api/customers)
// ==========================================
func TestCreateCustomer_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.CustomerCreateRequest{
			Name:    "PT Maju Bersama",
			Phone:   "081234567890",
			Address: "Jl. Sudirman No. 1",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/customers", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.CustomerResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "PT Maju Bersama", response.Data.Name)
		assert.NotEmpty(t, response.Data.ID)
	})

	t.Run("Failed_Validation_Missing_Phone", func(t *testing.T) {
		reqBody := dto.CustomerCreateRequest{
			Name:    "Pelanggan Tanpa Nomor",
			Address: "Jl. Buntu",
			// Phone kosong, padahal di DTO wajib (required)
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/customers", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// ==========================================
// 2. TEST GET CUSTOMER (GET /api/customers/:id)
// ==========================================
func TestGetCustomer_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	customer := tests.SeedCustomer(db, "Budi Pelanggan", "0899999", "Jakarta")

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/customers/"+customer.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.CustomerResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, customer.ID, response.Data.ID)
		assert.Equal(t, "Budi Pelanggan", response.Data.Name)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := httptest.NewRequest("GET", "/api/customers/"+randomID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// ==========================================
// 3. TEST LIST CUSTOMERS (GET /api/customers)
// ==========================================
func TestListCustomers_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	tests.SeedCustomer(db, "Cust A", "111", "Alamat A")
	tests.SeedCustomer(db, "Cust B", "222", "Alamat B")

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/customers?page=1&limit=10", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Data []dto.CustomerResponse `json:"data"`
			Meta any                    `json:"meta"`
		}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Len(t, response.Data, 2)
		assert.NotNil(t, response.Meta)
	})
}

// ==========================================
// 4. TEST UPDATE CUSTOMER (PUT /api/customers/:id)
// ==========================================
func TestUpdateCustomer_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	customer := tests.SeedCustomer(db, "Cust Lama", "0000", "Alamat Lama")

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.CustomerUpdateRequest{
			Name:    "Cust Baru",
			Phone:   "1111",
			Address: "Alamat Baru",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/customers/"+customer.ID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.CustomerResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Cust Baru", response.Data.Name)
		assert.Equal(t, "1111", response.Data.Phone)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		reqBody := dto.CustomerUpdateRequest{Name: "Coba Update"}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PUT", "/api/customers/"+randomID, bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// ==========================================
// 5. TEST DELETE CUSTOMER (DELETE /api/customers/:id)
// ==========================================
func TestDeleteCustomer_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	customer := tests.SeedCustomer(db, "Cust Hapus", "999", "Jakarta")

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/customers/"+customer.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Verifikasi Terhapus
		reqCheck := httptest.NewRequest("GET", "/api/customers/"+customer.ID, nil)
		respCheck, _ := app.Test(reqCheck, -1)
		assert.Equal(t, fiber.StatusNotFound, respCheck.StatusCode)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := httptest.NewRequest("DELETE", "/api/customers/"+randomID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
