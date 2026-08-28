package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/repository/postgres"
	"github.com/faridlan/sikon-api/internal/utils"
	tests "github.com/faridlan/sikon-api/test"
)

func TestCreateWorkLog_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	worker := postgres.WorkerModel{
		ID:         uuid.NewString(),
		Name:       "Mang Ade",
		Role:       "tailor",
		SalaryType: "piece_rate",
		Status:     "active",
	}
	db.Create(&worker)

	fmt.Println("Worker ID:", worker.ID) // Debugging line to print the worker ID

	batchPO := postgres.BatchPOModel{
		ID:        uuid.NewString(),
		Name:      "PO AUGUST 2026",
		Status:    "active",
		Quota:     500,
		StartDate: time.Now().AddDate(0, 0, -5),
		EndDate:   time.Now().AddDate(0, 0, 10),
	}
	db.Create(&batchPO)

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.WorkLogCreateRequest{
			WorkerID:   worker.ID,
			BatchPoID:  &batchPO.ID,
			JobType:    "jahit",
			Qty:        10,
			RatePerQty: 12000,
			WorkDate:   "2026-08-20",
			Notes:      "Variasi saku samping",
		}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/work-logs", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.WorkLogResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, worker.ID, response.Data.WorkerID)
		assert.Equal(t, 10, response.Data.Qty)
		assert.Equal(t, float64(12000), response.Data.RatePerQty)
		assert.Equal(t, float64(120000), response.Data.TotalAmount) // 10 * 12.000 = 120.000
		assert.Equal(t, "2026-08-20", response.Data.WorkDate)
		assert.NotEmpty(t, response.Data.ID)
	})

	t.Run("Failed_Validation", func(t *testing.T) {
		reqBody := dto.WorkLogCreateRequest{
			WorkerID:   worker.ID,
			Qty:        0, // Qty harus > 0
			RatePerQty: 12000,
			WorkDate:   "2026-08-20",
		}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/work-logs", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestDistributeWorkLog_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	sales := tests.SeedUser(db, "Sales Distribusi", "sales-distribute@example.com", "sales")
	customer := tests.SeedCustomerWithSales(db, "Customer Distribusi", "081234567890", "Bandung", sales.ID)
	category := tests.SeedCategory(db, "Kategori Distribusi")
	product := tests.SeedProduct(db, category.ID, "Produk Distribusi", 100000)
	worker1 := postgres.WorkerModel{ID: uuid.NewString(), Name: "Worker Satu", Role: "tailor", SalaryType: "piece_rate", Status: "active"}
	worker2 := postgres.WorkerModel{ID: uuid.NewString(), Name: "Worker Dua", Role: "tailor", SalaryType: "piece_rate", Status: "active"}
	db.Create(&worker1)
	db.Create(&worker2)
	batchPO := postgres.BatchPOModel{ID: uuid.NewString(), Name: "PO DISTRIBUSI", Status: "active", Quota: 10, StartDate: time.Now().AddDate(0, 0, -1), EndDate: time.Now().AddDate(0, 0, 10)}
	db.Create(&batchPO)
	order := postgres.OrderModel{
		ID: uuid.NewString(), OrderNumber: "ORD-DISTRIBUTE-001", BatchPoID: &batchPO.ID,
		CustomerID: customer.ID, SalesID: sales.ID, TotalAmount: 1000000,
		OrderStatus: string(domain.OrderStatusProduction), PaymentStatus: string(domain.PaymentStatusPartial),
	}
	db.Create(&order)
	db.Create(&postgres.OrderItemModel{ID: uuid.NewString(), OrderID: order.ID, ProductID: product.ID, Qty: 10, Price: 100000})

	t.Run("Success", func(t *testing.T) {
		body, _ := json.Marshal(dto.WorkLogDistributeRequest{
			BatchPOID: batchPO.ID, JobType: "jahit", WorkerIDs: []string{worker1.ID, worker2.ID},
			RatePerQty: 12000, WorkDate: "2026-08-20", Notes: "Distribusi test",
		})
		req := tests.AuthenticatedRequest("POST", "/api/work-logs/distribute", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[[]dto.WorkLogResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)
		assert.Len(t, response.Data, 2)
		if assert.Len(t, response.Data, 2) {
			assert.Equal(t, 10, response.Data[0].Qty+response.Data[1].Qty)
			assert.Equal(t, worker1.ID, response.Data[0].WorkerID)
			assert.Equal(t, worker2.ID, response.Data[1].WorkerID)
		}
	})

	t.Run("Failed_Validation", func(t *testing.T) {
		body, _ := json.Marshal(dto.WorkLogDistributeRequest{BatchPOID: batchPO.ID, JobType: "jahit", RatePerQty: 12000})
		req := tests.AuthenticatedRequest("POST", "/api/work-logs/distribute", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestGetWorkLog_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "HR Admin", "hr@example.com", "accounting")
	worker := postgres.WorkerModel{ID: uuid.NewString(), Name: "Mang Wahyu", Role: "cutter", SalaryType: "piece_rate", Status: "active"}
	db.Create(&worker)

	workLog := postgres.WorkLogModel{
		ID:          uuid.NewString(),
		WorkerID:    worker.ID,
		JobType:     "potong",
		Qty:         20,
		RatePerQty:  3000,
		TotalAmount: 60000,
		WorkDate:    time.Now(),
		CreatedByID: user.ID,
	}
	db.Create(&workLog)

	t.Run("Success", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/work-logs/"+workLog.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.WorkLogResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, workLog.ID, response.Data.ID)
		assert.Equal(t, 20, response.Data.Qty)
		assert.Equal(t, float64(60000), response.Data.TotalAmount)
		assert.NotNil(t, response.Data.Worker)
		assert.Equal(t, "Mang Wahyu", response.Data.Worker.Name)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := tests.AuthenticatedRequest("GET", "/api/work-logs/"+randomID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestListWorkLogs_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "HR Admin", "hr2@example.com", "accounting")
	worker1 := postgres.WorkerModel{ID: uuid.NewString(), Name: "Mang Ade", Role: "tailor", SalaryType: "piece_rate", Status: "active"}
	worker2 := postgres.WorkerModel{ID: uuid.NewString(), Name: "Kang Agus", Role: "finishing", SalaryType: "piece_rate", Status: "active"}
	db.Create(&worker1)
	db.Create(&worker2)

	log1 := postgres.WorkLogModel{ID: uuid.NewString(), WorkerID: worker1.ID, JobType: "jahit", Qty: 10, RatePerQty: 12000, TotalAmount: 120000, WorkDate: time.Now(), CreatedByID: user.ID}
	log2 := postgres.WorkLogModel{ID: uuid.NewString(), WorkerID: worker2.ID, JobType: "finishing", Qty: 50, RatePerQty: 1000, TotalAmount: 50000, WorkDate: time.Now(), CreatedByID: user.ID}
	db.Create(&log1)
	db.Create(&log2)

	t.Run("Success_List_All", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/work-logs?page=1&limit=10", nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Data []dto.WorkLogResponse `json:"data"`
		}
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Len(t, response.Data, 2)
	})

	t.Run("Success_Filter_WorkerID", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/work-logs?worker_id="+worker1.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Data []dto.WorkLogResponse `json:"data"`
		}
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Len(t, response.Data, 1)
		assert.Equal(t, worker1.ID, response.Data[0].WorkerID)
	})
}

func TestUpdateWorkLog_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "HR Admin", "hr3@example.com", "accounting")
	worker := postgres.WorkerModel{ID: uuid.NewString(), Name: "Mang Ade", Role: "tailor", SalaryType: "piece_rate", Status: "active"}
	db.Create(&worker)

	workLog := postgres.WorkLogModel{
		ID:          uuid.NewString(),
		WorkerID:    worker.ID,
		JobType:     "jahit",
		Qty:         5,
		RatePerQty:  12000,
		TotalAmount: 60000,
		WorkDate:    time.Now(),
		CreatedByID: user.ID,
	}
	db.Create(&workLog)

	t.Run("Success_Update_Qty_Recalculate", func(t *testing.T) {
		reqBody := dto.WorkLogUpdateRequest{
			Qty:        10, // Diubah dari 5 ke 10
			RatePerQty: 12000,
		}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("PUT", "/api/work-logs/"+workLog.ID, bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.WorkLogResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, 10, response.Data.Qty)
		assert.Equal(t, float64(120000), response.Data.TotalAmount) // Auto-recalculate 10 * 12.000 = 120.000
	})
}

func TestDeleteWorkLog_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "HR Admin", "hr4@example.com", "accounting")
	worker := postgres.WorkerModel{ID: uuid.NewString(), Name: "Mang Ade", Role: "tailor", SalaryType: "piece_rate", Status: "active"}
	db.Create(&worker)

	workLog := postgres.WorkLogModel{
		ID:          uuid.NewString(),
		WorkerID:    worker.ID,
		JobType:     "jahit",
		Qty:         5,
		RatePerQty:  12000,
		TotalAmount: 60000,
		WorkDate:    time.Now(),
		CreatedByID: user.ID,
	}
	db.Create(&workLog)

	t.Run("Success", func(t *testing.T) {
		req := tests.AuthenticatedRequest("DELETE", "/api/work-logs/"+workLog.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		checkReq := tests.AuthenticatedRequest("GET", "/api/work-logs/"+workLog.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		checkResp, err := app.Test(checkReq, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, checkResp.StatusCode)
	})
}
