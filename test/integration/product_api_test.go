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

	category := tests.SeedCategory(db, "Kemeja Taktikal")

	t.Run("Success_Basic", func(t *testing.T) {
		reqBody := dto.ProductCreateRequest{
			CategoryID:  category.ID,
			Name:        "Kemeja Taktikal Basic 5000",
			Description: "Bahan Combed 30s",
			BasePrice:   150000,
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

		assert.Equal(t, "Kemeja Taktikal Basic 5000", response.Data.Name)
		assert.Equal(t, category.ID, response.Data.CategoryID)
		assert.Equal(t, "kemeja-taktikal-basic-5000", response.Data.Slug)
		assert.NotEmpty(t, response.Data.ID)
	})

	t.Run("Success_Full_Custom_Product", func(t *testing.T) {
		reqBody := dto.ProductCreateRequest{
			CategoryID:    category.ID,
			Name:          "Kemeja Taktikal Premium 7200",
			Description:   "Bahan Ripstop Cotton 65/35",
			BasePrice:     185000,
			GSMInfo:       "210gsm",
			FabricSummary: "Ripstop",
			KeyFeatures:   []string{"Bahan anti robek", "Dual chest pocket velcro"},
			ImageURLs:     []string{"https://example.com/front.jpg", "https://example.com/back.jpg"},
			Fabrics: []dto.ProductFabricRequest{
				{
					Name:            "Ripstop Cotton 65/35",
					Description:     "Kuat & anti robek, 210gsm",
					Composition:     "65% Cotton / 35% Polyester",
					CareInstruction: "Cuci mesin air dingin",
					BasePrice:       185000,
					IsDefault:       true,
					Colors: []dto.FabricColorRequest{
						{Name: "Olive", HexCode: "#4b5320"},
						{Name: "Navy", HexCode: "#1b263b"},
					},
				},
			},
			Wholesale: []dto.WholesalePriceRequest{
				{MinQty: 6, UnitPrice: 175000},
			},
			DesignModel: &dto.ProductModelRequest{
				Name:        "Series 1 — Lengan Panjang",
				Type:        "long_sleeve",
				Description: "Template kemeja taktikal lengan panjang",
				Views: []dto.ProductModelViewRequest{
					{Side: "front", ArtURL: "https://example.com/art-front.png", MaskURL: "https://example.com/mask-front.png", Width: 1756, Height: 1920},
					{Side: "back", ArtURL: "https://example.com/art-back.png", MaskURL: "https://example.com/mask-back.png", Width: 1738, Height: 1920},
				},
			},
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

		assert.Equal(t, "Kemeja Taktikal Premium 7200", response.Data.Name)
		assert.Equal(t, 2, len(response.Data.Images))
		assert.Equal(t, 1, len(response.Data.Fabrics))
		assert.Equal(t, 2, len(response.Data.Fabrics[0].Colors))
		assert.Equal(t, 1, len(response.Data.Wholesale))
		assert.NotNil(t, response.Data.DesignModel)
		assert.Equal(t, 2, len(response.Data.DesignModel.Views))
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
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
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
// 2. TEST GET PRODUCT (GET /api/products/:id & /slug/:slug)
// ==========================================
func TestGetProduct_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := tests.SeedCategory(db, "Kemeja")
	fullProduct := tests.SeedFullCustomProduct(db, category.ID, "Kemeja PDL Lapangan", 175000)

	t.Run("Success_GetByID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/products/"+fullProduct.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.ProductResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, fullProduct.ID, response.Data.ID)
		assert.Equal(t, "Kemeja PDL Lapangan", response.Data.Name)
		assert.NotEmpty(t, response.Data.Fabrics)
		assert.NotEmpty(t, response.Data.Wholesale)
		assert.NotNil(t, response.Data.DesignModel)
	})

	t.Run("Success_GetBySlug", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/products/slug/"+fullProduct.Slug, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.ProductResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, fullProduct.Slug, response.Data.Slug)
		assert.Equal(t, "Kemeja PDL Lapangan", response.Data.Name)
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

	catAtasan := tests.SeedCategory(db, "Atasan")
	catBawahan := tests.SeedCategory(db, "Bawahan")

	tests.SeedProduct(db, catAtasan.ID, "Kemeja Polos Hitam", 100000)
	tests.SeedProduct(db, catAtasan.ID, "Kaos Sablon Premium", 75000)
	tests.SeedProduct(db, catBawahan.ID, "Celana Jeans Denim", 150000)

	t.Run("Success_GetAll_TanpaFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/products", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body map[string]any
		json.NewDecoder(resp.Body).Decode(&body)

		meta := body["meta"].(map[string]any)
		assert.Equal(t, float64(3), meta["total_items"])
	})

	t.Run("Success_FilterBySearch", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/products?search=kemeja", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body map[string]any
		json.NewDecoder(resp.Body).Decode(&body)

		data := body["data"].([]any)
		meta := body["meta"].(map[string]any)

		assert.Len(t, data, 1)
		assert.Equal(t, float64(1), meta["total_items"])
	})

	t.Run("Success_FilterByPriceRange", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/products?min_price=80000&max_price=120000", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body map[string]any
		json.NewDecoder(resp.Body).Decode(&body)

		data := body["data"].([]any)
		assert.Len(t, data, 1) // Kemeja Polos Hitam (100k)
	})

	t.Run("Success_SortingByPriceLow", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/products?sort_by=price_low", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var body map[string]any
		json.NewDecoder(resp.Body).Decode(&body)

		data := body["data"].([]any)
		firstItem := data[0].(map[string]any)
		assert.Equal(t, "Kaos Sablon Premium", firstItem["name"]) // 75k termurah
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

	t.Run("Success_Basic_Update", func(t *testing.T) {
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

	t.Run("Success_Update_With_Fabrics_And_Images", func(t *testing.T) {
		oldImage := postgres.ProductImageModel{
			ProductID: product.ID,
			ImageURL:  "https://example.com/gambar-lama.jpg",
			IsPrimary: true,
		}
		db.Create(&oldImage)

		reqBody := dto.ProductUpdateRequest{
			ImageURLs: []string{"https://example.com/gambar-baru.jpg"},
			Fabrics: []dto.ProductFabricRequest{
				{
					Name:      "Drill Premium",
					BasePrice: 175000,
					Colors: []dto.FabricColorRequest{
						{Name: "Black", HexCode: "#000000"},
					},
				},
			},
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

		assert.Equal(t, 1, len(response.Data.Images))
		assert.Equal(t, "https://example.com/gambar-baru.jpg", response.Data.Images[0].ImageURL)
		assert.Equal(t, 1, len(response.Data.Fabrics))
		assert.Equal(t, "Drill Premium", response.Data.Fabrics[0].Name)
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
