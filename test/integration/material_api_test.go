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

func TestCreateMaterial_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.MaterialCreateRequest{
			Name:      "Kain Ripstop",
			Unit:      "meter",
			UnitPrice: 25000,
			Category:  "kain",
		}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/materials", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.MaterialResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, "Kain Ripstop", response.Data.Name)
		assert.Equal(t, float64(25000), response.Data.UnitPrice)
		assert.NotEmpty(t, response.Data.ID)
	})

	t.Run("Failed_Validation", func(t *testing.T) {
		reqBody := dto.MaterialCreateRequest{Name: "", Unit: "meter", UnitPrice: 1000}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/materials", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestListMaterials_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	_ = tests.SeedMaterial(db, "Kain Ripstop", "meter", 25000, "kain")
	_ = tests.SeedMaterial(db, "Kancing", "pcs", 500, "aksesoris")

	t.Run("Success_List", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/materials?page=1&limit=10", nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Data []dto.MaterialResponse `json:"data"`
		}
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Len(t, response.Data, 2)
	})
}

func TestGetMaterial_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	material := tests.SeedMaterial(db, "Benang Jahit", "roll", 15000, "aksesoris")

	t.Run("Success", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/materials/"+material.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.MaterialResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, material.ID, response.Data.ID)
		assert.Equal(t, "Benang Jahit", response.Data.Name)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := tests.AuthenticatedRequest("GET", "/api/materials/"+randomID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestUpdateMaterial_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	material := tests.SeedMaterial(db, "Sleting", "pcs", 2000, "aksesoris")

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.MaterialUpdateRequest{UnitPrice: 2500}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("PUT", "/api/materials/"+material.ID, bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.MaterialResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, float64(2500), response.Data.UnitPrice)
		assert.Equal(t, "Sleting", response.Data.Name) // field lain tidak berubah
	})
}

func TestDeleteMaterial_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	material := tests.SeedMaterial(db, "Label", "pcs", 300, "aksesoris")

	t.Run("Success", func(t *testing.T) {
		req := tests.AuthenticatedRequest("DELETE", "/api/materials/"+material.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		checkReq := tests.AuthenticatedRequest("GET", "/api/materials/"+material.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		checkResp, err := app.Test(checkReq, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, checkResp.StatusCode)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := tests.AuthenticatedRequest("DELETE", "/api/materials/"+randomID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// ==========================================
// TESTS: PRODUCT MATERIAL (Resep / BOM)
// ==========================================

func TestSetProductMaterials_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := tests.SeedCategory(db, "Kaos")
	product := tests.SeedProduct(db, category.ID, "Kaos Polo", 150000)
	material := tests.SeedMaterial(db, "Kain Ripstop", "meter", 25000, "kain")

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.SetProductMaterialsRequest{
			Items: []dto.ProductMaterialItemRequest{
				{MaterialID: material.ID, QtyPerUnit: 1.2},
			},
		}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("PUT", "/api/products/"+product.ID+"/materials", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[[]dto.ProductMaterialResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Len(t, response.Data, 1)
		assert.Equal(t, material.ID, response.Data[0].MaterialID)
		assert.Equal(t, 1.2, response.Data[0].QtyPerUnit)
	})

	t.Run("Success_Replace_Bukan_Tambah", func(t *testing.T) {
		// Panggil lagi dengan resep berbeda -> resep lama harus HILANG, bukan ketambahan
		material2 := tests.SeedMaterial(db, "Kancing", "pcs", 500, "aksesoris")

		reqBody := dto.SetProductMaterialsRequest{
			Items: []dto.ProductMaterialItemRequest{
				{MaterialID: material2.ID, QtyPerUnit: 3},
			},
		}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("PUT", "/api/products/"+product.ID+"/materials", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[[]dto.ProductMaterialResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		// Cuma 1 baris (resep Kain Ripstop dari test sebelumnya sudah tergantikan)
		assert.Len(t, response.Data, 1)
		assert.Equal(t, material2.ID, response.Data[0].MaterialID)
	})
}

func TestGetProductMaterials_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := tests.SeedCategory(db, "Kemeja")
	product := tests.SeedProduct(db, category.ID, "Kemeja PDH", 200000)
	material := tests.SeedMaterial(db, "Kain American Drill", "meter", 30000, "kain")
	_ = tests.SeedProductMaterial(db, product.ID, material.ID, 1.5)

	t.Run("Success", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/products/"+product.ID+"/materials", nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[[]dto.ProductMaterialResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Len(t, response.Data, 1)
		assert.Equal(t, 1.5, response.Data[0].QtyPerUnit)
		// Pastikan relasi Material ikut ke-preload (bukan cuma ID doang)
		assert.NotNil(t, response.Data[0].Material)
		assert.Equal(t, "Kain American Drill", response.Data[0].Material.Name)
	})
}
