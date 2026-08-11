package integration_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/domain"
	postgresRepo "github.com/faridlan/sikon-api/internal/repository/postgres"
	tests "github.com/faridlan/sikon-api/test"
)

// ============================================================================
// 1. TEST ACCOUNTING REPORT ENDPOINT
// ============================================================================
func TestAccountingReportEndpoint(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := postgresRepo.CategoryModel{ID: uuid.NewString(), Name: "Jaket"}
	db.Create(&category)

	product := postgresRepo.ProductModel{ID: uuid.NewString(), CategoryID: category.ID, Name: "Jaket Outdoor", BasePrice: 100000}
	db.Create(&product)

	sales := postgresRepo.UserModel{ID: uuid.NewString(), Name: "Sales Akuntansi", Email: "acc@example.com", Role: "sales"}
	db.Create(&sales)

	customer := postgresRepo.CustomerModel{ID: uuid.NewString(), Name: "PT Akuntansi", CreatedBy: sales.ID, SalesID: &sales.ID}
	db.Create(&customer)

	batchPO := postgresRepo.BatchPOModel{
		ID:        uuid.NewString(),
		Name:      "PO BEBAS",
		Status:    string(domain.BatchPOStatusActive),
		Quota:     100,
		StartDate: time.Now().Add(-72 * time.Hour),
		EndDate:   time.Now().Add(72 * time.Hour),
	}
	db.Create(&batchPO)

	bank := postgresRepo.BankAccountModel{ID: uuid.NewString(), BankName: "BCA", AccountNumber: "123", AccountName: "Bos"}
	db.Create(&bank)

	juliTime := time.Date(2026, 7, 15, 10, 0, 0, 0, time.Local)

	order1 := postgresRepo.OrderModel{
		ID:            uuid.NewString(),
		OrderNumber:   "ORD-ACC-001",
		BatchPoID:     &batchPO.ID,
		CustomerID:    customer.ID,
		SalesID:       sales.ID,
		TotalAmount:   1000000,
		OrderStatus:   string(domain.OrderStatusProduction),
		PaymentStatus: string(domain.PaymentStatusPartial),
		CreatedAt:     juliTime,
		UpdatedAt:     juliTime,
	}
	db.Create(&order1)
	db.Exec("UPDATE orders SET approved_at = ? WHERE id = ?", "2026-07-15 10:00:00", order1.ID)

	orderItem1 := postgresRepo.OrderItemModel{ID: uuid.NewString(), OrderID: order1.ID, ProductID: product.ID, Qty: 10, Price: 100000}
	db.Create(&orderItem1)

	payment1 := postgresRepo.PaymentModel{
		ID:              uuid.NewString(),
		OrderID:         order1.ID,
		BankAccountID:   bank.ID,
		Amount:          500000,
		PaymentDate:     juliTime,
		ReferenceNumber: "DP-ACC",
		Status:          string(domain.PaymentVerificationVerified),
	}
	db.Create(&payment1)

	// 1. Seed Expense HPP (Bahan Baku) = 400.000
	hppCategory := postgresRepo.ExpenseCategoryModel{ID: uuid.NewString(), Name: "Bahan Baku", Type: "HPP"}
	db.Create(&hppCategory)

	hppExpense := postgresRepo.ExpenseModel{
		ID:                uuid.NewString(),
		ExpenseCategoryID: hppCategory.ID,
		Title:             "Beli Kain Jaket",
		Amount:            400000,
		ExpenseDate:       juliTime,
		CreatedByID:       sales.ID,
	}
	db.Create(&hppExpense)

	// 2. Seed Expense OPEX (Operasional) = 200.000
	opexCategory := postgresRepo.ExpenseCategoryModel{ID: uuid.NewString(), Name: "Operasional", Type: "OPEX"}
	db.Create(&opexCategory)

	opexExpense := postgresRepo.ExpenseModel{
		ID:                uuid.NewString(),
		ExpenseCategoryID: opexCategory.ID,
		Title:             "Bayar Listrik",
		Amount:            200000,
		ExpenseDate:       juliTime,
		CreatedByID:       sales.ID,
	}
	db.Create(&opexExpense)

	t.Run("Success Get Accounting Report", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/reports/accounting?start_date=2026-07-01&end_date=2026-07-31", bytes.NewBuffer(nil), "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Data dto.AccountingReportResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		report := response.Data
		assert.Equal(t, "2026-07-01", report.StartDate)
		assert.Equal(t, "2026-07-31", report.EndDate)

		// Verification Metrics
		assert.Equal(t, float64(1000000), report.Summary.TotalOmset)
		assert.Equal(t, float64(500000), report.Summary.TotalCashIn)
		assert.Equal(t, float64(500000), report.Summary.TotalReceivable)

		// Financial Breakdown
		assert.Equal(t, float64(400000), report.Summary.TotalHPP)
		assert.Equal(t, float64(600000), report.Summary.GrossProfit) // Omset (1jt) - HPP (400k) = 600k
		assert.Equal(t, float64(200000), report.Summary.TotalOPEX)
		assert.Equal(t, float64(400000), report.Summary.NetProfit)    // GrossProfit (600k) - OPEX (200k) = 400k
		assert.Equal(t, float64(-100000), report.Summary.NetCashflow) // CashIn (500k) - (HPP 400k + OPEX 200k) = -100k

		assert.Equal(t, 1, report.Summary.TotalOrderCount)
		assert.Equal(t, 10, report.Summary.TotalItemQty)
	})
}

// ============================================================================
// 2. TEST PRODUCTION REPORT ENDPOINT
// ============================================================================
func TestProductionReportEndpoint(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := postgresRepo.CategoryModel{ID: uuid.NewString(), Name: "Kemeja"}
	db.Create(&category)

	product := postgresRepo.ProductModel{ID: uuid.NewString(), CategoryID: category.ID, Name: "Kemeja Tactical", BasePrice: 150000}
	db.Create(&product)

	sales := postgresRepo.UserModel{ID: uuid.NewString(), Name: "Sales Produksi", Email: "prod@example.com", Role: "sales"}
	db.Create(&sales)

	customer := postgresRepo.CustomerModel{ID: uuid.NewString(), Name: "PT Produksi", CreatedBy: sales.ID, SalesID: &sales.ID}
	db.Create(&customer)

	batchPO := postgresRepo.BatchPOModel{
		ID:          uuid.NewString(),
		Name:        "PO Edisi Juli",
		Status:      string(domain.BatchPOStatusActive),
		Quota:       100,
		TargetMonth: 7,
		TargetYear:  2026,
		StartDate:   time.Now().Add(-24 * time.Hour),
		EndDate:     time.Now().Add(24 * time.Hour),
	}
	db.Create(&batchPO)

	order := postgresRepo.OrderModel{
		ID:            uuid.NewString(),
		OrderNumber:   "ORD-PROD-001",
		BatchPoID:     &batchPO.ID,
		CustomerID:    customer.ID,
		SalesID:       sales.ID,
		TotalAmount:   450000,
		OrderStatus:   string(domain.OrderStatusProduction),
		PaymentStatus: string(domain.PaymentStatusUnpaid),
	}
	db.Create(&order)

	orderItem := postgresRepo.OrderItemModel{ID: uuid.NewString(), OrderID: order.ID, ProductID: product.ID, Qty: 3, Price: 150000}
	db.Create(&orderItem)

	expenseCategory := postgresRepo.ExpenseCategoryModel{ID: uuid.NewString(), Name: "Bahan Baku", Type: "HPP"}
	db.Create(&expenseCategory)

	expense := postgresRepo.ExpenseModel{
		ID:                uuid.NewString(),
		ExpenseCategoryID: expenseCategory.ID,
		BatchPoID:         &batchPO.ID,
		Title:             "Beli Kain Tactical",
		Amount:            150000,
		ExpenseDate:       time.Now(),
		CreatedByID:       sales.ID,
	}
	db.Create(&expense)

	t.Run("Success Get Production Report", func(t *testing.T) {
		req := tests.AuthenticatedRequest("GET", "/api/reports/production?month=7&year=2026", bytes.NewBuffer(nil), "test-user", "test-user@sikon.com", domain.RoleOwner)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Data dto.ProductionReportResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		report := response.Data
		assert.Equal(t, 7, report.TargetMonth)
		assert.Equal(t, 2026, report.TargetYear)
		assert.Equal(t, 100, report.TotalQuota)
		assert.Equal(t, int64(3), report.TotalQtyOrdered)
		assert.Equal(t, int64(97), report.RemainingQuota)
		assert.Equal(t, float64(450000), report.TotalRevenue)

		assert.Equal(t, float64(150000), report.TotalHPP)
		assert.Equal(t, float64(300000), report.GrossProfit) // Revenue (450k) - HPP (150k) = 300k

		assert.Len(t, report.ActiveBatchPOs, 1)
		assert.Equal(t, "PO Edisi Juli", report.ActiveBatchPOs[0].Name)
	})
}

// ============================================================================
// 3. TEST PO SUMMARY ENDPOINT (Termasuk HPP, Gross Profit, & Piutang Customer)
// ============================================================================
func TestPOSummaryReportEndpoint(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := postgresRepo.CategoryModel{ID: uuid.NewString(), Name: "Celana"}
	db.Create(&category)

	product := postgresRepo.ProductModel{ID: uuid.NewString(), CategoryID: category.ID, Name: "Celana Tactical", BasePrice: 150000}
	db.Create(&product)

	sales := postgresRepo.UserModel{ID: uuid.NewString(), Name: "Budi", Email: "budi@example.com", Role: "sales"}
	db.Create(&sales)

	customer := postgresRepo.CustomerModel{ID: uuid.NewString(), Name: "PT Mundur", CreatedBy: sales.ID, SalesID: &sales.ID}
	db.Create(&customer)

	batchPO := postgresRepo.BatchPOModel{
		ID:        uuid.NewString(),
		Name:      "PO-SUMMARY",
		Status:    string(domain.OrderStatusCompleted),
		Quota:     50,
		StartDate: time.Now().Add(-48 * time.Hour),
		EndDate:   time.Now().Add(-24 * time.Hour),
	}
	db.Create(&batchPO)

	order := postgresRepo.OrderModel{
		ID:            uuid.NewString(),
		OrderNumber:   "ORD-TEST-002",
		BatchPoID:     &batchPO.ID,
		CustomerID:    customer.ID,
		SalesID:       sales.ID,
		TotalAmount:   450000,
		OrderStatus:   string(domain.OrderStatusCompleted),
		PaymentStatus: string(domain.PaymentStatusUnpaid),
	}
	db.Create(&order)

	orderItem := postgresRepo.OrderItemModel{ID: uuid.NewString(), OrderID: order.ID, ProductID: product.ID, Qty: 3, Price: 150000}
	db.Create(&orderItem)

	// Seed HPP untuk PO ini
	expCat := postgresRepo.ExpenseCategoryModel{ID: uuid.NewString(), Name: "HPP Kain", Type: "HPP"}
	db.Create(&expCat)
	expense := postgresRepo.ExpenseModel{
		ID:                uuid.NewString(),
		ExpenseCategoryID: expCat.ID,
		BatchPoID:         &batchPO.ID,
		Title:             "Kain Celana",
		Amount:            100000,
		ExpenseDate:       time.Now(),
		CreatedByID:       sales.ID,
	}
	db.Create(&expense)

	req := tests.AuthenticatedRequest("GET", "/api/reports/po/"+batchPO.ID+"/summary", bytes.NewBuffer(nil), "test-user", "test-user@sikon.com", domain.RoleOwner)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response struct {
		Data dto.POSummaryResponse `json:"data"`
	}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

	assert.Equal(t, batchPO.ID, response.Data.POID)
	assert.Equal(t, "PO-SUMMARY", response.Data.POName)
	assert.Equal(t, int64(3), response.Data.TotalQtyOrdered)
	assert.Equal(t, float64(450000), response.Data.TotalRevenue)
	assert.Equal(t, float64(100000), response.Data.TotalHPP)
	assert.Equal(t, float64(350000), response.Data.GrossProfit) // 450k - 100k

	// Validasi Piutang Customer spesifik di PO ini
	assert.Len(t, response.Data.CustomerReceivables, 1)
	assert.Equal(t, "PT Mundur", response.Data.CustomerReceivables[0].CustomerName)
	assert.Equal(t, float64(450000), response.Data.CustomerReceivables[0].OutstandingAmount)

	assert.Len(t, response.Data.ProductSummary, 1)
}

// ============================================================================
// 4. TEST RECEIVABLES REPORT ENDPOINT
// ============================================================================
func TestGetReceivablesReportEndpoint(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := postgresRepo.CategoryModel{ID: uuid.NewString(), Name: "Bawahan"}
	db.Create(&category)

	product := postgresRepo.ProductModel{ID: uuid.NewString(), CategoryID: category.ID, Name: "Celana", BasePrice: 150000}
	db.Create(&product)

	sales := postgresRepo.UserModel{ID: uuid.NewString(), Name: "Steven", Email: "steven@example.com", Role: "sales"}
	db.Create(&sales)

	customer := postgresRepo.CustomerModel{ID: uuid.NewString(), Name: "PT Sejahtera", CreatedBy: sales.ID, SalesID: &sales.ID}
	db.Create(&customer)

	batchPO := postgresRepo.BatchPOModel{
		ID:        uuid.NewString(),
		Name:      "PO PIUTANG",
		Status:    string(domain.BatchPOStatusActive),
		Quota:     100,
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now().Add(72 * time.Hour),
	}
	db.Create(&batchPO)

	order := postgresRepo.OrderModel{
		ID:            uuid.NewString(),
		OrderNumber:   "ORD-TEST-REC-001",
		BatchPoID:     &batchPO.ID,
		CustomerID:    customer.ID,
		SalesID:       sales.ID,
		TotalAmount:   450000,
		OrderStatus:   string(domain.OrderStatusProduction),
		PaymentStatus: string(domain.PaymentStatusUnpaid),
	}
	db.Create(&order)

	orderItem := postgresRepo.OrderItemModel{ID: uuid.NewString(), OrderID: order.ID, ProductID: product.ID, Qty: 3, Price: 150000}
	db.Create(&orderItem)

	req := tests.AuthenticatedRequest("GET", "/api/reports/receivables", bytes.NewBuffer(nil), "test-user", "test-user@sikon.com", domain.RoleOwner)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response struct {
		Data []dto.ReceivableDetailResponse `json:"data"`
	}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

	assert.NotEmpty(t, response.Data)
	assert.Len(t, response.Data, 1)

	firstData := response.Data[0]
	assert.Equal(t, "ORD-TEST-REC-001", firstData.OrderNumber)
	assert.Equal(t, "PT Sejahtera", firstData.CustomerName)
	assert.Equal(t, float64(450000), firstData.TotalAmount)
	assert.Equal(t, float64(0), firstData.TotalPaid)
	assert.Equal(t, float64(450000), firstData.OutstandingAmount)
}

// ============================================================================
// 5. TEST DAILY REPORT ENDPOINT
// ============================================================================
func TestDailyReportEndpoint(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := postgresRepo.CategoryModel{ID: uuid.NewString(), Name: "Kemeja"}
	db.Create(&category)

	product := postgresRepo.ProductModel{ID: uuid.NewString(), CategoryID: category.ID, Name: "Kemeja Dinas", BasePrice: 120000}
	db.Create(&product)

	sales := postgresRepo.UserModel{ID: uuid.NewString(), Name: "John Doe", Email: "john@example.com", Role: "sales"}
	db.Create(&sales)

	customer := postgresRepo.CustomerModel{ID: uuid.NewString(), Name: "PT Harian", CreatedBy: sales.ID, SalesID: &sales.ID}
	db.Create(&customer)

	batchPO := postgresRepo.BatchPOModel{
		ID:        uuid.NewString(),
		Name:      "PO 2 JULI 2026",
		Status:    string(domain.BatchPOStatusActive),
		Quota:     400,
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now().Add(72 * time.Hour),
	}
	db.Create(&batchPO)

	todayStr := time.Now().Format("2006-01-02")
	todayTime := time.Now()

	order := postgresRepo.OrderModel{
		ID:            uuid.NewString(),
		OrderNumber:   "ORD-DAILY-001",
		BatchPoID:     &batchPO.ID,
		CustomerID:    customer.ID,
		SalesID:       sales.ID,
		TotalAmount:   240000,
		OrderStatus:   string(domain.OrderStatusProduction),
		PaymentStatus: string(domain.PaymentStatusUnpaid),
		CreatedAt:     todayTime,
		UpdatedAt:     todayTime,
	}
	db.Create(&order)
	db.Exec("UPDATE orders SET approved_at = ? WHERE id = ?", todayTime.Format("2006-01-02 15:00:00"), order.ID)

	orderItem := postgresRepo.OrderItemModel{ID: uuid.NewString(), OrderID: order.ID, ProductID: product.ID, Qty: 2, Price: 120000}
	db.Create(&orderItem)

	req := tests.AuthenticatedRequest("GET", "/api/reports/daily?date="+todayStr, bytes.NewBuffer(nil), "test-user", "test-user@sikon.com", domain.RoleOwner)
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response struct {
		Data dto.DailyReportResponse `json:"data"`
	}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

	assert.Equal(t, todayStr, response.Data.ReportDate)
	assert.Equal(t, batchPO.ID, response.Data.POInfo.POID)
	assert.Equal(t, "PO 2 JULI 2026", response.Data.POInfo.POName)
	assert.Equal(t, int64(2), response.Data.OrderSummary.QtyToday)
	assert.Equal(t, int64(2), response.Data.OrderSummary.QtyTotalPO)
	assert.Equal(t, float64(240000), response.Data.FinancialSummary.TotalRevenue)
	assert.NotEmpty(t, response.Data.SalesDetails)
	assert.Equal(t, "John Doe", response.Data.SalesDetails[0].SalesName)
	assert.Equal(t, int64(2), response.Data.SalesDetails[0].TotalQty)
}
