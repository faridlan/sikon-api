package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/utils"
	tests "github.com/faridlan/sikon-api/test"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestMaterial_And_DynamicHPP_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	sales := tests.SeedUser(db, "Sales", "sales@sikon.com", "sales")
	cust := tests.SeedCustomer(db, "Customer Dinas", "0812345678", "Jakarta")
	cat := tests.SeedCategory(db, "Kemeja")
	batchPo := tests.SeedBatchPO(db, "PO September 2026", "active")
	db.Exec("UPDATE batch_pos SET open_date = ?, close_date = ? WHERE id = ?", time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour), batchPo.ID)

	var createdFabricID string
	var createdColorID string

	t.Run("1. Create Fabric Material with Specs and Colors", func(t *testing.T) {
		reqBody := dto.MaterialCreateRequest{
			Name:            "American Drill 1919",
			Unit:            "meter",
			UnitPrice:       35000,
			Category:        "kain",
			Description:     "Kain tebal, kuat dan berserat halus",
			Composition:     "65% Polyester / 35% Viscose",
			CareInstruction: "Cuci suhu ruang, jangan gunakan pemutih klorin",
			GSMInfo:         "210 gsm",
			Colors: []dto.FabricColorRequest{
				{Name: "Navy Blue", HexCode: "#000080"},
				{Name: "Khaki", HexCode: "#C3B091"},
			},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/materials", bytes.NewBuffer(bodyBytes), "admin", "admin@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var res utils.SuccessResponse[dto.MaterialResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &res)

		assert.Equal(t, "American Drill 1919", res.Data.Name)
		assert.Equal(t, float64(35000), res.Data.UnitPrice)
		assert.Equal(t, "210 gsm", res.Data.GSMInfo)
		assert.Len(t, res.Data.Colors, 2)
		assert.Equal(t, "Navy Blue", res.Data.Colors[0].Name)

		createdFabricID = res.Data.ID
		createdColorID = res.Data.Colors[0].ID
	})

	t.Run("2. Get Fabric Material By ID", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/materials/"+createdFabricID, nil, "admin", "admin@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var res utils.SuccessResponse[dto.MaterialResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &res)

		assert.Equal(t, createdFabricID, res.Data.ID)
		assert.Equal(t, "American Drill 1919", res.Data.Name)
		assert.Len(t, res.Data.Colors, 2)
	})

	var productID string

	t.Run("3. Create Product with Fabric Linking to Master Material", func(t *testing.T) {
		reqBody := dto.ProductCreateRequest{
			CategoryID:  cat.ID,
			Name:        "Kemeja PDH Dinas",
			Description: "Kemeja dinas resmi lengan pendek",
			BasePrice:   125000,
			Fabrics: []dto.ProductFabricRequest{
				{
					FabricID:   &createdFabricID,
					QtyPerUnit: 1.5,
					BasePrice:  125000,
					IsDefault:  true,
				},
			},
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/products", bytes.NewBuffer(bodyBytes), "admin", "admin@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var res utils.SuccessResponse[dto.ProductResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &res)

		assert.Equal(t, "Kemeja PDH Dinas", res.Data.Name)
		assert.Len(t, res.Data.Fabrics, 1)
		assert.Equal(t, &createdFabricID, res.Data.Fabrics[0].FabricID)
		assert.Equal(t, 1.5, res.Data.Fabrics[0].QtyPerUnit)
		// Nama dan colors harus otomatis ter-fallback dari Master Material!
		assert.Equal(t, "American Drill 1919", res.Data.Fabrics[0].Name)
		assert.Len(t, res.Data.Fabrics[0].Colors, 2)

		productID = res.Data.ID
	})

	t.Run("4. Set Product Materials for Accessories (BOM)", func(t *testing.T) {
		// Buat material kancing (aksesoris)
		kancingReq := dto.MaterialCreateRequest{
			Name:      "Kancing Kemeja",
			Unit:      "pcs",
			UnitPrice: 500,
			Category:  "aksesoris",
		}
		kbBytes, _ := json.Marshal(kancingReq)
		reqK := tests.AuthenticatedRequest("POST", "/api/materials", bytes.NewBuffer(kbBytes), "admin", "admin@sikon.com", domain.RoleOwner)
		reqK.Header.Set("Content-Type", "application/json")
		respK, _ := app.Test(reqK, -1)
		var kancingRes utils.SuccessResponse[dto.MaterialResponse]
		kBytes, _ := io.ReadAll(respK.Body)
		json.Unmarshal(kBytes, &kancingRes)

		// Set BOM produk: 8 pcs kancing per kemeja = 8 * 500 = 4.000 per pcs
		bomReq := dto.SetProductMaterialsRequest{
			Items: []dto.ProductMaterialItemRequest{
				{
					MaterialID: kancingRes.Data.ID,
					QtyPerUnit: 8,
				},
			},
		}
		bBytes, _ := json.Marshal(bomReq)
		reqBOM := tests.AuthenticatedRequest("PUT", "/api/products/"+productID+"/materials", bytes.NewBuffer(bBytes), "admin", "admin@sikon.com", domain.RoleOwner)
		reqBOM.Header.Set("Content-Type", "application/json")
		respBOM, err := app.Test(reqBOM, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, respBOM.StatusCode)
	})

	t.Run("5. Create Order with Fabric & Color, Transition to Production, Verify Dynamic HPP", func(t *testing.T) {
		orderQty := 10
		// Ekspektasi HPP:
		// Kain = 10 pcs * 1.5 meter * 35.000 = 525.000
		// Kancing = 10 pcs * 8 pcs * 500 = 40.000
		// Total HPP Material = 565.000

		orderReq := dto.OrderCreateRequest{
			BatchPoID:  batchPo.ID,
			CustomerID: cust.ID,
			SalesID:    sales.ID,
			Items: []dto.OrderItemRequest{
				{
					ProductID:     productID,
					FabricID:      &createdFabricID,
					FabricColorID: &createdColorID,
					CustomName:    "Kemeja PDH Lengan Pendek",
					Qty:           orderQty,
					Price:         125000,
				},
			},
		}
		oBytes, _ := json.Marshal(orderReq)
		reqOrder := tests.AuthenticatedRequest("POST", "/api/orders", bytes.NewBuffer(oBytes), "admin", "admin@sikon.com", domain.RoleOwner)
		reqOrder.Header.Set("Content-Type", "application/json")
		respOrder, err := app.Test(reqOrder, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, respOrder.StatusCode)

		var orderRes utils.SuccessResponse[dto.OrderResponse]
		oRespBody, _ := io.ReadAll(respOrder.Body)
		json.Unmarshal(oRespBody, &orderRes)

		orderID := orderRes.Data.ID
		assert.Equal(t, &createdFabricID, orderRes.Data.Items[0].FabricID)
		assert.Equal(t, "American Drill 1919", orderRes.Data.Items[0].FabricName)
		assert.Equal(t, "Navy Blue", orderRes.Data.Items[0].FabricColorName)

		// Set payment_status = partial agar bisa masuk pending (minimal sudah ada DP)
		db.Exec("UPDATE orders SET payment_status = ? WHERE id = ?", "partial", orderID)

		// Ubah status ke pending — HPP Material dihitung di sini (quotation → pending)
		statusReq := dto.OrderStatusUpdateRequest{OrderStatus: "pending"}
		sBytes, _ := json.Marshal(statusReq)
		reqStatus := tests.AuthenticatedRequest("PATCH", "/api/orders/"+orderID+"/status", bytes.NewBuffer(sBytes), "admin", "admin@sikon.com", domain.RoleOwner)
		reqStatus.Header.Set("Content-Type", "application/json")
		respStatus, err := app.Test(reqStatus, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, respStatus.StatusCode, "Transisi quotation → pending harus berhasil")

		// Set payment_status = paid agar bisa masuk production
		db.Exec("UPDATE orders SET payment_status = ? WHERE id = ?", "paid", orderID)

		// Ubah status ke production (pending → production)
		statusReq2 := dto.OrderStatusUpdateRequest{OrderStatus: "production"}
		s2Bytes, _ := json.Marshal(statusReq2)
		reqStatus2 := tests.AuthenticatedRequest("PATCH", "/api/orders/"+orderID+"/status", bytes.NewBuffer(s2Bytes), "admin", "admin@sikon.com", domain.RoleOwner)
		reqStatus2.Header.Set("Content-Type", "application/json")
		respStatus2, err := app.Test(reqStatus2, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, respStatus2.StatusCode, "Transisi pending → production harus berhasil")

		// Cek Endpoint HPP: GET /api/orders/:id/hpp
		reqHPP := tests.AuthenticatedRequest("GET", "/api/orders/"+orderID+"/hpp", nil, "admin", "admin@sikon.com", domain.RoleOwner)
		respHPP, err := app.Test(reqHPP, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, respHPP.StatusCode)

		var hppRes utils.SuccessResponse[dto.OrderHPPResponse]
		hBytes, _ := io.ReadAll(respHPP.Body)
		json.Unmarshal(hBytes, &hppRes)

		assert.Equal(t, orderID, hppRes.Data.OrderID)
		assert.Equal(t, float64(565000), hppRes.Data.MaterialCost, "HPP Material Cost harus 565.000 (525.000 kain + 40.000 kancing)")
		assert.NotNil(t, hppRes.Data.MaterialCalculatedAt)
	})
}
