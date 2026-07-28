package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/utils"
	tests "github.com/faridlan/sikon-api/test"
)

func TestBatchPO_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()

	t.Run("Create Batch PO - Success", func(t *testing.T) {
		tests.ClearTables(db)

		reqBody := dto.BatchPOCreateRequest{
			Name:        "PO Lebaran 2026",
			TargetMonth: 3,    // <-- TAMBAHAN: Maret
			TargetYear:  2026, // <-- TAMBAHAN: 2026
			StartDate:   time.Now(),
			EndDate:     time.Now().AddDate(0, 0, 14),
			Quota:       500,
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/batch-pos", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.BatchPOResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.NotEmpty(t, response.Data.ID)
		assert.Equal(t, "PO Lebaran 2026", response.Data.Name)
		assert.Equal(t, 3, response.Data.TargetMonth)   // <-- ASSERTION BARU
		assert.Equal(t, 2026, response.Data.TargetYear) // <-- ASSERTION BARU
		assert.Equal(t, "draft", response.Data.Status)
		assert.Equal(t, 500, response.Data.Quota)
	})

	t.Run("Create Batch PO - Validation Error (Nama Kosong)", func(t *testing.T) {
		tests.ClearTables(db)

		reqBody := dto.BatchPOCreateRequest{
			Name:        "", // Dikosongkan agar gagal validasi
			TargetMonth: 3,
			TargetYear:  2026,
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/batch-pos", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	// --- SKENARIO BARU: VALIDASI BULAN TIDAK MASUK AKAL ---
	t.Run("Create Batch PO - Validation Error (Bulan Invalid)", func(t *testing.T) {
		tests.ClearTables(db)

		reqBody := dto.BatchPOCreateRequest{
			Name:        "PO Error",
			TargetMonth: 13, // <-- Gagal: Maksimal 12
			TargetYear:  2026,
			Quota:       100,
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/batch-pos", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Get Batch PO Detail - Success", func(t *testing.T) {
		tests.ClearTables(db)
		seededPO := tests.SeedBatchPO(db, "PO Reguler Juni", "active")

		req := httptest.NewRequest("GET", "/api/batch-pos/"+seededPO.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.BatchPOResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, seededPO.ID, response.Data.ID)
		assert.Equal(t, "PO Reguler Juni", response.Data.Name)
	})

	t.Run("List Active Batch POs - Dropdown Sales", func(t *testing.T) {
		tests.ClearTables(db)

		// Buat 3 PO dengan status berbeda
		tests.SeedBatchPO(db, "PO 1", "active")
		tests.SeedBatchPO(db, "PO 2", "closed")
		tests.SeedBatchPO(db, "PO 3", "active")

		req := httptest.NewRequest("GET", "/api/batch-pos/active", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[[]dto.BatchPOResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Harus hanya mengembalikan 2 data yang active
		assert.Len(t, response.Data, 2)
		for _, po := range response.Data {
			assert.Equal(t, "active", po.Status)
		}
	})

	t.Run("Update Batch PO Status - Activate", func(t *testing.T) {
		tests.ClearTables(db)
		seededPO := tests.SeedBatchPO(db, "PO Baru", "draft")

		reqBody := dto.BatchPOStatusUpdateRequest{
			Status: "active",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", "/api/batch-pos/"+seededPO.ID+"/status", bytes.NewBuffer(bodyJson))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Verifikasi perubahan di database
		var updatedStatus string
		db.Raw("SELECT status FROM batch_pos WHERE id = ?", seededPO.ID).Scan(&updatedStatus)
		assert.Equal(t, "active", updatedStatus)
	})

	t.Run("Delete Batch PO - Success (Soft Delete)", func(t *testing.T) {
		tests.ClearTables(db)
		seededPO := tests.SeedBatchPO(db, "PO Batal", "draft")

		req := httptest.NewRequest("DELETE", "/api/batch-pos/"+seededPO.ID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Verifikasi Soft Delete di database (harus tidak ditemukan dengan pencarian normal)
		var count int64
		db.Table("batch_pos").Where("id = ? AND deleted_at IS NULL", seededPO.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("Error - UUID Tidak Valid", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/batch-pos/bukan-uuid-123", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Error - Data Tidak Ditemukan", func(t *testing.T) {
		tests.ClearTables(db)
		fakeID := uuid.New().String()

		req := httptest.NewRequest("GET", "/api/batch-pos/"+fakeID, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
