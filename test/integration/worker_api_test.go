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
	"github.com/faridlan/sikon-api/internal/repository/postgres"
	"github.com/faridlan/sikon-api/internal/utils"
	tests "github.com/faridlan/sikon-api/test"
)

func TestCreateWorker_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.WorkerCreateRequest{
			Name:       "Mang Ade",
			Phone:      "081234567890",
			Role:       "tailor",
			SalaryType: "piece_rate",
		}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/workers", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.WorkerResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, "Mang Ade", response.Data.Name)
		assert.Equal(t, "tailor", response.Data.Role)
		assert.Equal(t, "piece_rate", response.Data.SalaryType)
		assert.Equal(t, "active", response.Data.Status)
		assert.NotEmpty(t, response.Data.ID)
	})

	t.Run("Failed_Validation", func(t *testing.T) {
		reqBody := dto.WorkerCreateRequest{
			Name:       "", // Mandatory
			Role:       "tailor",
			SalaryType: "piece_rate",
		}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/workers", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestGetWorker_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	workerModel := postgres.WorkerModel{
		ID:         uuid.NewString(),
		Name:       "Mang Wahyu",
		Phone:      "081987654321",
		Role:       "cutter",
		SalaryType: "daily",
		Status:     "active",
	}
	db.Create(&workerModel)

	t.Run("Success", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/workers/"+workerModel.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.WorkerResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, workerModel.ID, response.Data.ID)
		assert.Equal(t, "Mang Wahyu", response.Data.Name)
		assert.Equal(t, "cutter", response.Data.Role)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := tests.AuthenticatedRequest("GET", "/api/workers/"+randomID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestListWorkers_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	worker1 := postgres.WorkerModel{ID: uuid.NewString(), Name: "Mang Ade", Role: "tailor", SalaryType: "piece_rate", Status: "active"}
	worker2 := postgres.WorkerModel{ID: uuid.NewString(), Name: "Kang Agus", Role: "finishing", SalaryType: "monthly", Status: "active"}
	db.Create(&worker1)
	db.Create(&worker2)

	t.Run("Success_List_All", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/workers?page=1&limit=10", nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Message string               `json:"message"`
			Data    []dto.WorkerResponse `json:"data"`
		}
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Len(t, response.Data, 2)
	})

	t.Run("Success_Filter_Role", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/workers?role=tailor", nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Data []dto.WorkerResponse `json:"data"`
		}
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Len(t, response.Data, 1)
		assert.Equal(t, "Mang Ade", response.Data[0].Name)
	})
}

func TestUpdateWorker_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	workerModel := postgres.WorkerModel{
		ID:         uuid.NewString(),
		Name:       "Mang Ade",
		Phone:      "081234567890",
		Role:       "tailor",
		SalaryType: "piece_rate",
		Status:     "active",
	}
	db.Create(&workerModel)

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.WorkerUpdateRequest{
			Name:   "Mang Ade Supriatna",
			Status: "inactive",
		}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("PUT", "/api/workers/"+workerModel.ID, bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.WorkerResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, "Mang Ade Supriatna", response.Data.Name)
		assert.Equal(t, "inactive", response.Data.Status)
	})
}

func TestDeleteWorker_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	workerModel := postgres.WorkerModel{
		ID:         uuid.NewString(),
		Name:       "Worker To Delete",
		Role:       "helper",
		SalaryType: "daily",
		Status:     "active",
	}
	db.Create(&workerModel)

	t.Run("Success", func(t *testing.T) {
		req := tests.AuthenticatedRequest("DELETE", "/api/workers/"+workerModel.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		checkReq := tests.AuthenticatedRequest("GET", "/api/workers/"+workerModel.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		checkResp, err := app.Test(checkReq, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, checkResp.StatusCode)
	})
}
