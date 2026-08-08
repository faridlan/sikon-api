package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
	tests "github.com/faridlan/sikon-api/test"
)

// ==========================================
// 1. TEST CREATE CUSTOMER (POST /api/customers)
// ==========================================
func TestCreateCustomer_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// Buat user sales terlebih dahulu untuk dites
	salesUser := tests.SeedUser(db, "Joko Sales", "joko@sikon.com", "sales")

	t.Run("Success_With_SalesID", func(t *testing.T) {
		reqBody := dto.CustomerCreateRequest{
			Name:    "PT Maju Bersama",
			Phone:   "081234567890",
			Address: "Jl. Sudirman No. 1",
			SalesID: salesUser.ID, // Menambahkan SalesID yang valid
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/customers", bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.CustomerResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "PT Maju Bersama", response.Data.Name)
		assert.NotEmpty(t, response.Data.ID)
		assert.Equal(t, salesUser.ID, response.Data.SalesID) // Verifikasi SalesID tersimpan
	})

	t.Run("Failed_Validation_Missing_Phone", func(t *testing.T) {
		reqBody := dto.CustomerCreateRequest{
			Name:    "Pelanggan Tanpa Nomor",
			Address: "Jl. Buntu",
			// Phone kosong, padahal di DTO wajib (required)
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/customers", bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
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

	salesUser := tests.SeedUser(db, "Rini Sales", "rini@sikon.com", "sales")
	customer := tests.SeedCustomerWithSales(db, "Budi Pelanggan", "0899999", "Jakarta", salesUser.ID)

	t.Run("Success", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/customers/"+customer.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.CustomerResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, customer.ID, response.Data.ID)
		assert.Equal(t, "Budi Pelanggan", response.Data.Name)

		// Verifikasi relasi Sales berhasil di-preload dan ditampilkan
		assert.Equal(t, salesUser.ID, response.Data.SalesID)
		assert.NotNil(t, response.Data.Sales)
		assert.Equal(t, "Rini Sales", response.Data.Sales.Name)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := tests.AuthenticatedRequest("GET", "/api/customers/"+randomID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
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

	// 1. Buat 2 Sales berbeda
	sales1 := tests.SeedUser(db, "Deni Sales", "deni@sikon.com", "sales")
	sales2 := tests.SeedUser(db, "Rudi Sales", "rudi@sikon.com", "sales")

	// 2. Buat data Customer
	tests.SeedCustomerWithSales(db, "Maju Jaya", "08111111", "Jakarta", sales1.ID)
	tests.SeedCustomerWithSales(db, "Maju Terus", "08222222", "Bandung", sales1.ID)
	tests.SeedCustomerWithSales(db, "Mundur Alon", "08333333", "Surabaya", sales2.ID)

	t.Run("Success_Get_All", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/customers?page=1&limit=10", nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Data []dto.CustomerResponse `json:"data"`
			Meta any                    `json:"meta"`
		}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Tanpa filter, harus kembali 3
		assert.Len(t, response.Data, 3)
	})

	t.Run("Success_Search_By_Name", func(t *testing.T) {
		// Kata kunci "Maju", harus mengembalikan "Maju Jaya" & "Maju Terus"
		req := tests.AuthenticatedRequest("GET", "/api/customers?search=Maju", nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Data []dto.CustomerResponse `json:"data"`
		}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Len(t, response.Data, 2)
	})

	t.Run("Success_Search_By_Phone", func(t *testing.T) {
		// Kata kunci "0833", harus mengembalikan "Mundur Alon"
		req := tests.AuthenticatedRequest("GET", "/api/customers?search=0833", nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Data []dto.CustomerResponse `json:"data"`
		}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Len(t, response.Data, 1)
		assert.Equal(t, "Mundur Alon", response.Data[0].Name)
	})

	t.Run("Success_Filter_By_SalesID", func(t *testing.T) {
		// Filter menggunakan ID milik sales1 (Deni), harus kembali 2
		req := tests.AuthenticatedRequest("GET", "/api/customers?sales_id="+sales1.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Data []dto.CustomerResponse `json:"data"`
		}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Len(t, response.Data, 2)
		for _, c := range response.Data {
			assert.Equal(t, sales1.ID, c.SalesID) // Pastikan benar-benar milik Deni
		}
	})
}

// ==========================================
// 4. TEST UPDATE CUSTOMER (PUT /api/customers/:id)
// ==========================================
func TestUpdateCustomer_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	customer := tests.SeedCustomer(db, "Cust Lama", "0000", "Alamat Lama")
	salesUserBaru := tests.SeedUser(db, "Rudi Sales", "rudi@sikon.com", "sales")

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.CustomerUpdateRequest{
			Name:    "Cust Baru",
			Phone:   "1111",
			Address: "Alamat Baru",
			SalesID: salesUserBaru.ID, // Assign sales_id baru
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("PUT", "/api/customers/"+customer.ID, bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.CustomerResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Cust Baru", response.Data.Name)
		assert.Equal(t, "1111", response.Data.Phone)
		assert.Equal(t, salesUserBaru.ID, response.Data.SalesID) // Verifikasi update ID Sales
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		reqBody := dto.CustomerUpdateRequest{Name: "Coba Update"}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("PUT", "/api/customers/"+randomID, bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
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
		req := tests.AuthenticatedRequest("DELETE", "/api/customers/"+customer.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Verifikasi Terhapus
		reqCheck := tests.AuthenticatedRequest("GET", "/api/customers/"+customer.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		respCheck, _ := app.Test(reqCheck, -1)
		assert.Equal(t, fiber.StatusNotFound, respCheck.StatusCode)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := tests.AuthenticatedRequest("DELETE", "/api/customers/"+randomID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
