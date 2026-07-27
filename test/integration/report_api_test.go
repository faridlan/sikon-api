package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
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

func TestReportDailyEndpoint(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	category := postgresRepo.CategoryModel{
		ID:   uuid.NewString(),
		Name: "Atasan",
	}
	assert.NoError(t, db.Create(&category).Error)

	product := postgresRepo.ProductModel{
		ID:         uuid.NewString(),
		CategoryID: category.ID,
		Name:       "Kemeja",
		BasePrice:  100000,
	}
	assert.NoError(t, db.Create(&product).Error)

	sales := postgresRepo.UserModel{
		ID:       uuid.NewString(),
		Name:     "Rina",
		Email:    "rina@example.com",
		Password: "secret",
		Role:     "sales",
	}
	assert.NoError(t, db.Create(&sales).Error)

	customer := postgresRepo.CustomerModel{
		ID:        uuid.NewString(),
		Name:      "PT Maju",
		Phone:     "0812",
		Address:   "Bandung",
		CreatedBy: sales.ID,
		SalesID:   &sales.ID,
	}
	assert.NoError(t, db.Create(&customer).Error)

	batchPO := postgresRepo.BatchPOModel{
		ID:        uuid.NewString(),
		Name:      "PO-001",
		Status:    string(domain.BatchPOStatusActive),
		Quota:     10,
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now().Add(24 * time.Hour),
	}
	assert.NoError(t, db.Create(&batchPO).Error)

	order := postgresRepo.OrderModel{
		ID:            uuid.NewString(),
		OrderNumber:   "ORD-TEST-001",
		BatchPoID:     &batchPO.ID,
		CustomerID:    customer.ID,
		SalesID:       sales.ID,
		TotalAmount:   300000,
		Subtotal:      300000,
		OrderStatus:   string(domain.OrderStatusProduction),
		PaymentStatus: string(domain.PaymentStatusUnpaid),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	assert.NoError(t, db.Create(&order).Error)

	orderItem := postgresRepo.OrderItemModel{
		ID:        uuid.NewString(),
		OrderID:   order.ID,
		ProductID: product.ID,
		Qty:       3,
		Price:     100000,
	}
	assert.NoError(t, db.Create(&orderItem).Error)

	req := httptest.NewRequest("GET", "/api/reports/daily?date="+time.Now().Format("2006-01-02"), bytes.NewBuffer(nil))
	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var response struct {
		Data struct {
			DailySnapshot struct {
				TotalRevenueToday float64 `json:"total_revenue_today"`
				TotalQtyToday     int64   `json:"total_qty_today"`
			} `json:"daily_snapshot"`
			ActivePOs []struct {
				BatchPOName string `json:"batch_po_name"`
			} `json:"active_pos"`
			TotalOutstandingReceivables float64 `json:"total_outstanding_receivables"`
			ActivePOReceivables         float64 `json:"active_po_receivables"`
		} `json:"data"`
	}

	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&response))
	assert.Equal(t, float64(300000), response.Data.DailySnapshot.TotalRevenueToday)
	assert.Equal(t, int64(3), response.Data.DailySnapshot.TotalQtyToday)
	assert.NotEmpty(t, response.Data.ActivePOs)
	assert.GreaterOrEqual(t, response.Data.TotalOutstandingReceivables, float64(0))
}

func TestGenerateDailyReportEndpoint(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// --- 1. Seeding Data ---
	category := postgresRepo.CategoryModel{
		ID:   uuid.NewString(),
		Name: "Atasan",
	}
	assert.NoError(t, db.Create(&category).Error)

	product := postgresRepo.ProductModel{
		ID:         uuid.NewString(),
		CategoryID: category.ID,
		Name:       "Kemeja",
		BasePrice:  100000,
	}
	assert.NoError(t, db.Create(&product).Error)

	sales := postgresRepo.UserModel{
		ID:       uuid.NewString(),
		Name:     "Rina",
		Email:    "rina@example.com",
		Password: "secret",
		Role:     "sales",
	}
	assert.NoError(t, db.Create(&sales).Error)

	customer := postgresRepo.CustomerModel{
		ID:        uuid.NewString(),
		Name:      "PT Maju",
		Phone:     "0812",
		Address:   "Bandung",
		CreatedBy: sales.ID,
		SalesID:   &sales.ID,
	}
	assert.NoError(t, db.Create(&customer).Error)

	batchPO := postgresRepo.BatchPOModel{
		ID:        uuid.NewString(),
		Name:      "PO-001",
		Status:    string(domain.BatchPOStatusActive),
		Quota:     10,
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now().Add(24 * time.Hour),
	}
	assert.NoError(t, db.Create(&batchPO).Error)

	order := postgresRepo.OrderModel{
		ID:            uuid.NewString(),
		OrderNumber:   "ORD-TEST-001",
		BatchPoID:     &batchPO.ID,
		CustomerID:    customer.ID,
		SalesID:       sales.ID,
		TotalAmount:   300000,
		Subtotal:      300000,
		OrderStatus:   string(domain.OrderStatusProduction),
		PaymentStatus: string(domain.PaymentStatusUnpaid),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	assert.NoError(t, db.Create(&order).Error)

	orderItem := postgresRepo.OrderItemModel{
		ID:        uuid.NewString(),
		OrderID:   order.ID,
		ProductID: product.ID,
		Qty:       3,
		Price:     100000,
	}
	assert.NoError(t, db.Create(&orderItem).Error)

	// --- 2. Hit Endpoint API ---
	url := "/api/reports/daily/generate?date=" + time.Now().Format("2006-01-02")
	req := httptest.NewRequest("GET", url, bytes.NewBuffer(nil))

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// --- 3. Decode & Assert Response ---
	var response struct {
		Data struct {
			ReportDate string `json:"report_date"`
			POInfo     struct {
				POID           string `json:"po_id"`
				POName         string `json:"po_name"`
				RemainingQuota int64  `json:"remaining_quota"`
			} `json:"po_info"`
			OrderSummary struct {
				QtyToday   int64 `json:"qty_today"`
				QtyTotalPO int64 `json:"qty_total_po"`
				TrendData  []struct {
					Date string `json:"date"`
					Qty  int64  `json:"qty"`
				} `json:"trend_data"`
			} `json:"order_summary"`
			FinancialSummary struct {
				TotalRevenue        float64 `json:"total_revenue"`
				ActivePOOutstanding float64 `json:"active_po_outstanding"`
			} `json:"financial_summary"`
			SalesDetails []struct {
				SalesName  string           `json:"sales_name"`
				TotalQty   int64            `json:"total_qty"`
				Categories map[string]int64 `json:"categories"`
			} `json:"sales_details"`
		} `json:"data"`
	}

	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

	assert.Equal(t, time.Now().Format("2006-01-02"), response.Data.ReportDate)
	assert.Equal(t, "PO-001", response.Data.POInfo.POName)
	assert.Equal(t, int64(3), response.Data.OrderSummary.QtyToday)
	assert.Equal(t, int64(3), response.Data.OrderSummary.QtyTotalPO)

	// Menyesuaikan dengan field financial_summary terbaru
	assert.Equal(t, float64(300000), response.Data.FinancialSummary.TotalRevenue)
	assert.Equal(t, float64(300000), response.Data.FinancialSummary.ActivePOOutstanding)

	// Memastikan trend_data ter-generate dengan benar
	assert.NotEmpty(t, response.Data.OrderSummary.TrendData)

	assert.Len(t, response.Data.SalesDetails, 1)
	assert.Equal(t, "Rina", response.Data.SalesDetails[0].SalesName)
	assert.Equal(t, int64(3), response.Data.SalesDetails[0].Categories["Atasan"])
}

func TestPOSummaryReportEndpoint(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// --- 1. Seeding Data ---
	// Gunakan setup yang identik dengan tes sebelumnya
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
		Name:      "PO-002",
		Status:    string(domain.OrderStatusCompleted), // Test status completed
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

	orderItem := postgresRepo.OrderItemModel{
		ID:        uuid.NewString(),
		OrderID:   order.ID,
		ProductID: product.ID,
		Qty:       3,
		Price:     150000,
	}
	db.Create(&orderItem)

	// --- 2. Hit Endpoint API ---
	// Endpoint Close Order
	url := "/api/reports/po/" + batchPO.ID + "/summary"
	req := httptest.NewRequest("GET", url, bytes.NewBuffer(nil))

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// --- 3. Decode & Assert Response ---
	var response struct {
		Data struct {
			POID             string  `json:"po_id"`
			POName           string  `json:"po_name"`
			TotalQtyOrdered  int64   `json:"total_qty_ordered"`
			TotalRevenue     float64 `json:"total_revenue"`
			TotalOutstanding float64 `json:"total_outstanding"`
			ProductSummary   []struct {
				CategoryName string `json:"category_name"`
				TotalQty     int64  `json:"total_qty"`
			} `json:"product_summary"`
			SalesSummary []struct {
				SalesName    string  `json:"sales_name"`
				TotalQty     int64   `json:"total_qty"`
				TotalRevenue float64 `json:"total_revenue"`
			} `json:"sales_summary"`
		} `json:"data"`
	}

	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

	assert.Equal(t, batchPO.ID, response.Data.POID)
	assert.Equal(t, "PO-002", response.Data.POName)
	assert.Equal(t, int64(3), response.Data.TotalQtyOrdered)
	assert.Equal(t, float64(450000), response.Data.TotalRevenue)
	assert.Equal(t, float64(450000), response.Data.TotalOutstanding) // Belum ada payment

	assert.Len(t, response.Data.ProductSummary, 1)
	assert.Equal(t, "Celana", response.Data.ProductSummary[0].CategoryName)
	assert.Equal(t, int64(3), response.Data.ProductSummary[0].TotalQty)

	assert.Len(t, response.Data.SalesSummary, 1)
	assert.Equal(t, "Budi", response.Data.SalesSummary[0].SalesName)
	assert.Equal(t, int64(3), response.Data.SalesSummary[0].TotalQty)
	assert.Equal(t, float64(450000), response.Data.SalesSummary[0].TotalRevenue)
}

func TestGetReceivablesReportEndpoint(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// --- 1. Seeding Data ---
	category := postgresRepo.CategoryModel{ID: uuid.NewString(), Name: "Bawahan"}
	assert.NoError(t, db.Create(&category).Error)

	product := postgresRepo.ProductModel{ID: uuid.NewString(), CategoryID: category.ID, Name: "Celana", BasePrice: 150000}
	assert.NoError(t, db.Create(&product).Error)

	sales := postgresRepo.UserModel{ID: uuid.NewString(), Name: "Steven", Email: "steven@example.com", Role: "sales"}
	assert.NoError(t, db.Create(&sales).Error)

	customer := postgresRepo.CustomerModel{ID: uuid.NewString(), Name: "PT Sejahtera", CreatedBy: sales.ID, SalesID: &sales.ID}
	assert.NoError(t, db.Create(&customer).Error)

	batchPO := postgresRepo.BatchPOModel{
		ID:        uuid.NewString(),
		Name:      "PO AGUSTUS 2026",
		Status:    string(domain.BatchPOStatusActive),
		Quota:     100,
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now().Add(72 * time.Hour),
	}
	assert.NoError(t, db.Create(&batchPO).Error)

	// Skenario: Order status production, payment_status unpaid (Sisa piutang utuh)
	order := postgresRepo.OrderModel{
		ID:            uuid.NewString(),
		OrderNumber:   "ORD-TEST-REC-001",
		BatchPoID:     &batchPO.ID,
		CustomerID:    customer.ID,
		SalesID:       sales.ID,
		TotalAmount:   450000,
		Subtotal:      450000,
		OrderStatus:   string(domain.OrderStatusProduction),
		PaymentStatus: string(domain.PaymentStatusUnpaid),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	assert.NoError(t, db.Create(&order).Error)

	orderItem := postgresRepo.OrderItemModel{
		ID:        uuid.NewString(),
		OrderID:   order.ID,
		ProductID: product.ID,
		Qty:       3,
		Price:     150000,
	}
	assert.NoError(t, db.Create(&orderItem).Error)

	// --- 2. Hit Endpoint API ---
	req := httptest.NewRequest("GET", "/api/reports/receivables", bytes.NewBuffer(nil))
	// req.Header.Set("Authorization", "Bearer "+token) // Aktifkan jika pakai JWT

	resp, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// --- 3. Decode & Assert Response ---
	var response struct {
		Data []struct {
			OrderID           string  `json:"order_id"`
			OrderNumber       string  `json:"order_number"`
			POName            string  `json:"po_name"`
			POStatus          string  `json:"po_status"`
			CustomerName      string  `json:"customer_name"`
			SalesName         string  `json:"sales_name"`
			TotalAmount       float64 `json:"total_amount"`
			TotalPaid         float64 `json:"total_paid"`
			OutstandingAmount float64 `json:"outstanding_amount"`
		} `json:"data"`
	}

	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&response))

	// Karena ada 1 order yang belum dibayar, array harus berisi minimal 1
	assert.NotEmpty(t, response.Data)
	assert.Len(t, response.Data, 1)

	// Validasi nilainya
	firstData := response.Data[0]
	assert.Equal(t, "ORD-TEST-REC-001", firstData.OrderNumber)
	assert.Equal(t, "PO AGUSTUS 2026", firstData.POName)
	assert.Equal(t, "PT Sejahtera", firstData.CustomerName)
	assert.Equal(t, "Steven", firstData.SalesName)
	assert.Equal(t, float64(450000), firstData.TotalAmount)
	assert.Equal(t, float64(0), firstData.TotalPaid) // Karena unpaid, total_paid = 0
	assert.Equal(t, float64(450000), firstData.OutstandingAmount)
}

// Tambahkan di bagian paling bawah file integration/report_api_test.go
func TestGetMonthlyReportEndpoint(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// --- 1. Seeding Data Master ---
	category := postgresRepo.CategoryModel{ID: uuid.NewString(), Name: "Jaket"}
	assert.NoError(t, db.Create(&category).Error)

	product := postgresRepo.ProductModel{ID: uuid.NewString(), CategoryID: category.ID, Name: "Jaket Outdoor", BasePrice: 100000}
	assert.NoError(t, db.Create(&product).Error)

	sales := postgresRepo.UserModel{ID: uuid.NewString(), Name: "Sales Juli", Email: "salesjuli@example.com", Role: "sales"}
	assert.NoError(t, db.Create(&sales).Error)

	customer := postgresRepo.CustomerModel{ID: uuid.NewString(), Name: "PT Bulan Ini", CreatedBy: sales.ID, SalesID: &sales.ID}
	assert.NoError(t, db.Create(&customer).Error)

	batchPO := postgresRepo.BatchPOModel{
		ID:        uuid.NewString(),
		Name:      "PO JULI 2026",
		Status:    string(domain.BatchPOStatusActive),
		Quota:     100,
		StartDate: time.Now().Add(-72 * time.Hour),
		EndDate:   time.Now().Add(72 * time.Hour),
	}
	assert.NoError(t, db.Create(&batchPO).Error)

	bank := postgresRepo.BankAccountModel{ID: uuid.NewString(), BankName: "BCA", AccountNumber: "123", AccountName: "Bos"}
	assert.NoError(t, db.Create(&bank).Error)

	// --- 2. Setup Data Transaksi (Modifikasi Tanggal Langsung di DB agar simulasi bulan JULI 2026) ---

	// Gunakan waktu statis Juli 2026
	juliTime := time.Date(2026, 7, 10, 10, 0, 0, 0, time.Local)

	// Order 1: Total 1.000.000, Approved di Juli 2026
	order1 := postgresRepo.OrderModel{
		ID:            uuid.NewString(),
		OrderNumber:   "ORD-JULI-001",
		BatchPoID:     &batchPO.ID,
		CustomerID:    customer.ID,
		SalesID:       sales.ID,
		TotalAmount:   1000000,
		OrderStatus:   string(domain.OrderStatusProduction),
		PaymentStatus: string(domain.PaymentStatusPartial),
		CreatedAt:     juliTime,
		UpdatedAt:     juliTime,
	}
	// Paksa ApprovedAt terisi agar masuk ke query Monthly Report
	db.Create(&order1)
	db.Exec("UPDATE orders SET approved_at = ? WHERE id = ?", "2026-07-10 10:00:00", order1.ID)

	// Order Item 1 (10 pcs)
	orderItem1 := postgresRepo.OrderItemModel{ID: uuid.NewString(), OrderID: order1.ID, ProductID: product.ID, Qty: 10, Price: 100000}
	db.Create(&orderItem1)

	// Pembayaran 1: DP 500.000 di Juli 2026
	payment1 := postgresRepo.PaymentModel{
		ID:              uuid.NewString(),
		OrderID:         order1.ID,
		BankAccountID:   bank.ID,
		Amount:          500000,
		PaymentDate:     juliTime, // 10 Juli 2026
		ReferenceNumber: "DP-001",
	}
	db.Create(&payment1)

	// --- 3. Hit Endpoint API & Assert ---
	t.Run("Success_Get_Monthly_Report_July_2026", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/reports/monthly?month=7&year=2026", bytes.NewBuffer(nil))
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response struct {
			Data dto.MonthlyReportResponse `json:"data"`
		}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		report := response.Data
		assert.Equal(t, 7, report.Month)
		assert.Equal(t, 2026, report.Year)

		// Validasi Kalkulasi Keuangan
		assert.Equal(t, float64(1000000), report.Summary.TotalOmset)
		assert.Equal(t, float64(500000), report.Summary.TotalCashIn)

		// Piutang = Omset (1jt) - Cash In (500rb) = 500.000
		assert.Equal(t, float64(500000), report.Summary.TotalReceivable)

		assert.Equal(t, 1, report.Summary.TotalOrderCount)
		assert.Equal(t, 10, report.Summary.TotalItemQty)

		// Validasi Leaderboard Sales
		assert.NotEmpty(t, report.SalesPerformances)
		assert.Equal(t, "Sales Juli", report.SalesPerformances[0].SalesName)

		// Validasi trend harian (tanggal 1-31) ter-generate tanpa bolong
		assert.Equal(t, 31, len(report.DailyTrends))
	})

	t.Run("Failed_Invalid_Month_Param", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/reports/monthly?month=13", bytes.NewBuffer(nil))
		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}
