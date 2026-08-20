package integration_test

import (
	"bytes"
	"encoding/json"
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

func TestCreatePayroll_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "HR Admin", "hr_payroll@example.com", "accounting")
	worker := postgres.WorkerModel{ID: uuid.NewString(), Name: "Mang Ade", Role: "tailor", SalaryType: "piece_rate", Status: "active"}
	db.Create(&worker)

	log1 := postgres.WorkLogModel{ID: uuid.NewString(), WorkerID: worker.ID, JobType: "jahit", Qty: 10, RatePerQty: 12000, TotalAmount: 120000, WorkDate: time.Now(), CreatedByID: user.ID}
	log2 := postgres.WorkLogModel{ID: uuid.NewString(), WorkerID: worker.ID, JobType: "jahit", Qty: 5, RatePerQty: 12000, TotalAmount: 60000, WorkDate: time.Now(), CreatedByID: user.ID}
	db.Create(&log1)
	db.Create(&log2)

	t.Run("Success_Create_Draft_Payroll", func(t *testing.T) {
		reqBody := dto.PayrollCreateRequest{
			StartDate:  "2026-08-14",
			EndDate:    "2026-08-20",
			WorkLogIDs: []string{log1.ID, log2.ID},
		}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/payrolls", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.PayrollResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, "draft", response.Data.Status)
		assert.Equal(t, float64(180000), response.Data.TotalAmount) // 120.000 + 60.000 = 180.000
		assert.Len(t, response.Data.WorkLogs, 2)
		assert.NotEmpty(t, response.Data.PayrollNumber)
	})

	t.Run("Failed_Validation_Empty_WorkLogs", func(t *testing.T) {
		reqBody := dto.PayrollCreateRequest{
			StartDate:  "2026-08-14",
			EndDate:    "2026-08-20",
			WorkLogIDs: []string{},
		}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/payrolls", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestGetPayroll_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "HR Admin", "hr_get_payroll@example.com", "accounting")
	payroll := postgres.PayrollModel{
		ID:            uuid.NewString(),
		PayrollNumber: "PAY-202608-001",
		StartDate:     time.Now().AddDate(0, 0, -7),
		EndDate:       time.Now(),
		TotalAmount:   250000,
		Status:        "draft",
		CreatedByID:   user.ID,
	}
	db.Create(&payroll)

	t.Run("Success", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/payrolls/"+payroll.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.PayrollResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, payroll.ID, response.Data.ID)
		assert.Equal(t, "PAY-202608-001", response.Data.PayrollNumber)
		assert.Equal(t, float64(250000), response.Data.TotalAmount)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := tests.AuthenticatedRequest("GET", "/api/payrolls/"+randomID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestProcessPayrollPayment_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "Accounting Admin", "acc_pay@example.com", "accounting")
	category := tests.SeedExpenseCategory(db, "Ongkos Jahit / Borongan", "HPP", "Pembayaran upah jahit borongan")

	worker := postgres.WorkerModel{ID: uuid.NewString(), Name: "Mang Ade", Role: "tailor", SalaryType: "piece_rate", Status: "active"}
	db.Create(&worker)

	batchPO := postgres.BatchPOModel{ID: uuid.NewString(), Name: "PO AUGUST 2026", Status: "active", Quota: 500, StartDate: time.Now(), EndDate: time.Now().AddDate(0, 0, 10)}
	db.Create(&batchPO)

	workLog := postgres.WorkLogModel{
		ID:          uuid.NewString(),
		WorkerID:    worker.ID,
		BatchPoID:   &batchPO.ID,
		JobType:     "jahit",
		Qty:         20,
		RatePerQty:  12000,
		TotalAmount: 240000,
		WorkDate:    time.Now(),
		CreatedByID: user.ID,
	}
	db.Create(&workLog)

	payroll := postgres.PayrollModel{
		ID:            uuid.NewString(),
		PayrollNumber: "PAY-202608-002",
		StartDate:     time.Now().AddDate(0, 0, -7),
		EndDate:       time.Now(),
		TotalAmount:   240000,
		Status:        "draft",
		CreatedByID:   user.ID,
	}
	db.Create(&payroll)

	// Bind workLog ke Payroll
	db.Model(&workLog).Update("payroll_id", payroll.ID)

	t.Run("Success_Process_Payment_And_Auto_Create_Expense_HPP", func(t *testing.T) {
		req := tests.AuthenticatedRequest("POST", "/api/payrolls/"+payroll.ID+"/pay", nil, "test-user", "test-user@sikon.com", domain.RoleOwner)

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.PayrollResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		// 1. Verifikasi Status Payroll berubah jadi PAID
		assert.Equal(t, "paid", response.Data.Status)
		assert.NotEmpty(t, response.Data.ExpenseID)
		assert.NotNil(t, response.Data.PaidAt)

		// 2. Verifikasi Record Expense HPP Otomatis Terbuat di DB
		var autoExpense postgres.ExpenseModel
		errExpense := db.Where("id = ?", *response.Data.ExpenseID).First(&autoExpense).Error
		assert.NoError(t, errExpense)
		assert.Equal(t, category.ID, autoExpense.ExpenseCategoryID)
		assert.Equal(t, float64(240000), autoExpense.Amount)
		assert.Equal(t, &batchPO.ID, autoExpense.BatchPoID) // Linked ke Batch PO!
	})
}

func TestDeletePayroll_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "HR Admin", "hr_del_payroll@example.com", "accounting")
	payroll := postgres.PayrollModel{
		ID:            uuid.NewString(),
		PayrollNumber: "PAY-202608-DEL",
		StartDate:     time.Now().AddDate(0, 0, -7),
		EndDate:       time.Now(),
		TotalAmount:   100000,
		Status:        "draft",
		CreatedByID:   user.ID,
	}
	db.Create(&payroll)

	t.Run("Success_Delete_Draft_Payroll", func(t *testing.T) {
		req := tests.AuthenticatedRequest("DELETE", "/api/payrolls/"+payroll.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		checkReq := tests.AuthenticatedRequest("GET", "/api/payrolls/"+payroll.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		checkResp, err := app.Test(checkReq, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, checkResp.StatusCode)
	})
}
