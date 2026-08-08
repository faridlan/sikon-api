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
	"github.com/faridlan/sikon-api/internal/utils"
	tests "github.com/faridlan/sikon-api/test"
)

func TestCreateExpenseCategory_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.ExpenseCategoryCreateRequest{
			Name:        "Bahan Baku",
			Type:        "HPP",
			Description: "Pembelian kain utama",
		}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/expenses/categories", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.ExpenseCategoryResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, "Bahan Baku", response.Data.Name)
		assert.Equal(t, "HPP", response.Data.Type)
		assert.NotEmpty(t, response.Data.ID)
	})

	t.Run("Failed_Validation", func(t *testing.T) {
		reqBody := dto.ExpenseCategoryCreateRequest{Name: "", Type: "HPP"}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/expenses/categories", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestCreateExpense_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "Admin Keuangan", "keuangan@example.com", "accounting")
	category := tests.SeedExpenseCategory(db, "Bahan Baku", "HPP", "Pembelian kain")

	t.Run("Success", func(t *testing.T) {
		reqBody := dto.ExpenseCreateRequest{
			ExpenseCategoryID: category.ID,
			CreatedByID:       user.ID,
			Title:             "Beli Kain Ripstop",
			Amount:            1500000,
			ExpenseDate:       time.Now(),
			Notes:             "Pembelian di Toko Maju",
		}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/expenses", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var response utils.SuccessResponse[dto.ExpenseResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, "Beli Kain Ripstop", response.Data.Title)
		assert.Equal(t, category.ID, response.Data.ExpenseCategoryID)
		assert.NotEmpty(t, response.Data.CreatedByID)
		assert.NotEmpty(t, response.Data.ID)
	})

	t.Run("Failed_Validation", func(t *testing.T) {
		reqBody := dto.ExpenseCreateRequest{Title: "", Amount: 0, ExpenseCategoryID: category.ID, ExpenseDate: time.Now()}
		body, _ := json.Marshal(reqBody)

		req := tests.AuthenticatedRequest("POST", "/api/expenses", bytes.NewBuffer(body), "test-user", "test-user@sikon.com", domain.RoleOwner)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestListExpenses_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "Admin Keuangan", "keuangan2@example.com", "accounting")
	category := tests.SeedExpenseCategory(db, "Operasional", "OPEX", "Biaya kantor")
	_ = tests.SeedExpense(db, category.ID, user.ID, nil, "Gaji Staff", 5000000, time.Now())
	_ = tests.SeedExpense(db, category.ID, user.ID, nil, "Listrik", 750000, time.Now().AddDate(0, 0, -1))

	t.Run("Success_List", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/expenses?page=1&limit=10", nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Message string                `json:"message"`
			Data    []dto.ExpenseResponse `json:"data"`
			Meta    any                   `json:"meta"`
		}
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Len(t, response.Data, 2)
	})
}

func TestGetExpense_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "Admin Keuangan", "keuangan3@example.com", "accounting")
	category := tests.SeedExpenseCategory(db, "Bahan Baku", "HPP", "Pembelian kain")
	expense := tests.SeedExpense(db, category.ID, user.ID, nil, "Beli Benang", 1200000, time.Now())

	t.Run("Success", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/expenses/"+expense.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.ExpenseResponse]
		bodyResp, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(bodyResp, &response)

		assert.Equal(t, expense.ID, response.Data.ID)
		assert.Equal(t, "Beli Benang", response.Data.Title)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := tests.AuthenticatedRequest("GET", "/api/expenses/"+randomID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestDeleteExpense_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	user := tests.SeedUser(db, "Admin Keuangan", "keuangan4@example.com", "accounting")
	category := tests.SeedExpenseCategory(db, "Bahan Baku", "HPP", "Pembelian kain")
	expense := tests.SeedExpense(db, category.ID, user.ID, nil, "Hapus Data", 900000, time.Now())

	t.Run("Success", func(t *testing.T) {
		req := tests.AuthenticatedRequest("DELETE", "/api/expenses/"+expense.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		checkReq := tests.AuthenticatedRequest("GET", "/api/expenses/"+expense.ID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		checkResp, err := app.Test(checkReq, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, checkResp.StatusCode)
	})

	t.Run("Failed_NotFound", func(t *testing.T) {
		randomID := uuid.New().String()
		req := tests.AuthenticatedRequest("DELETE", "/api/expenses/"+randomID, nil, "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
