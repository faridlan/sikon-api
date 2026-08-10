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
// 1. TEST CREATE SPEC TEMPLATE (POST /api/spec-templates)
// ==========================================
func TestCreateSpecTemplate_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.SpecTemplateRequest{
			Name: "Bahan Rompi Premium",
			Spec: "Drill Premium, Furing Asahi, Resleting YKK",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/spec-templates", bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.SpecTemplateResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Bahan Rompi Premium", response.Data.Name)
		assert.Equal(t, "Drill Premium, Furing Asahi, Resleting YKK", response.Data.Spec)
		assert.NotEmpty(t, response.Data.ID)
	})

	t.Run("Failed_Validation_Missing_Name", func(t *testing.T) {
		reqBody := dto.SpecTemplateRequest{
			Name: "", // Name kosong (harus error)
			Spec: "Drill Premium",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/spec-templates", bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Failed_Validation_Missing_Spec", func(t *testing.T) {
		reqBody := dto.SpecTemplateRequest{
			Name: "Bahan Rompi Premium",
			Spec: "", // Spec kosong (harus error)
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/spec-templates", bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// ==========================================
// 2. TEST GET SPEC TEMPLATE (GET /api/spec-templates/:id)
// ==========================================
func TestGetSpecTemplate_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// Seed data spec template ke DB terlebih dahulu
	specTpl := tests.SeedSpecTemplate(db, "Kemeja PDL", "Ripstop, Bordir Komputer")

	t.Run("Success", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/spec-templates/"+specTpl.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.SpecTemplateResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, specTpl.ID, response.Data.ID)
		assert.Equal(t, "Kemeja PDL", response.Data.Name)
		assert.Equal(t, "Ripstop, Bordir Komputer", response.Data.Spec)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String() // Generate UUID acak yang tidak ada di DB
		req := tests.AuthenticatedRequest("GET", "/api/spec-templates/"+randomID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// ==========================================
// 3. TEST LIST SPEC TEMPLATES (GET /api/spec-templates)
// ==========================================
func TestListSpecTemplates_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// Seed 3 data spec template
	tests.SeedSpecTemplate(db, "Template 1", "Spec A")
	tests.SeedSpecTemplate(db, "Template 2", "Spec B")
	tests.SeedSpecTemplate(db, "Template 3", "Spec C")

	t.Run("Success_GetList", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/spec-templates?page=1&limit=10", nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// Gunakan struct anonim jika Utils tidak punya generic untuk pagination
		var response struct {
			Message string                     `json:"message"`
			Data    []dto.SpecTemplateResponse `json:"data"`
			Meta    any                        `json:"meta"`
		}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Len(t, response.Data, 3) // Pastikan ada 3 data yang kembali
	})
}

// ==========================================
// 4. TEST UPDATE SPEC TEMPLATE (PUT /api/spec-templates/:id)
// ==========================================
func TestUpdateSpecTemplate_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	specTpl := tests.SeedSpecTemplate(db, "Nama Lama", "Spec Lama")

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.SpecTemplateRequest{
			Name: "Nama Baru Terupdate",
			Spec: "Spec Baru Terupdate",
		}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("PUT", "/api/spec-templates/"+specTpl.ID, bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.SpecTemplateResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		assert.Equal(t, "Nama Baru Terupdate", response.Data.Name)
		assert.Equal(t, "Spec Baru Terupdate", response.Data.Spec)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		reqBody := dto.SpecTemplateRequest{Name: "Coba Update Data Gaib", Spec: "Ngasal"}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("PUT", "/api/spec-templates/"+randomID, bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("Failed_Validation_Body", func(t *testing.T) {
		reqBody := dto.SpecTemplateRequest{Name: "", Spec: ""} // Tidak boleh kosong
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("PUT", "/api/spec-templates/"+specTpl.ID, bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Failed_Invalid_UUID", func(t *testing.T) {
		reqBody := dto.SpecTemplateRequest{Name: "Test Invalid ID", Spec: "Test"}
		bodyJson, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("PUT", "/api/spec-templates/id-bukan-uuid", bytes.NewBuffer(bodyJson), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode) // Dicegat oleh utils.ValidateUUID
	})
}

// ==========================================
// 5. TEST DELETE SPEC TEMPLATE (DELETE /api/spec-templates/:id)
// ==========================================
func TestDeleteSpecTemplate_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	specTpl := tests.SeedSpecTemplate(db, "Kategori Untuk Dihapus", "Spec Hapus")

	t.Run("Success", func(t *testing.T) {
		// 1. Eksekusi Delete
		req := tests.AuthenticatedRequest("DELETE", "/api/spec-templates/"+specTpl.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		// 2. Verifikasi data benar-benar hilang (Tembak GET by ID)
		reqCheck := tests.AuthenticatedRequest("GET", "/api/spec-templates/"+specTpl.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		respCheck, _ := app.Test(reqCheck, -1)

		// Harusnya API mengembalikan 404 NotFound
		assert.Equal(t, fiber.StatusNotFound, respCheck.StatusCode)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := tests.AuthenticatedRequest("DELETE", "/api/spec-templates/"+randomID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("Failed_Invalid_UUID", func(t *testing.T) {
		req := tests.AuthenticatedRequest("DELETE", "/api/spec-templates/ini-id-ngasal", nil, "test-user", "test-user@sikon.com", domain.RoleOwner)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}
