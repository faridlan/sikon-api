package integration_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/faridlan/sikon-api/internal/delivery/http/dto"
	"github.com/faridlan/sikon-api/internal/utils"
	tests "github.com/faridlan/sikon-api/test"
)

func TestGetDashboardSummary_Integration(t *testing.T) {
	// 1. Setup App & Bersihkan Database
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	// 2. Siapkan Data Pendukung (Kategori, Produk, User, Customer, Bank)
	cat := tests.SeedCategory(db, "Kemeja Taktikal")
	prod := tests.SeedProduct(db, cat.ID, "Kemeja W-Tac", 100000)
	sales := tests.SeedUser(db, "Sales Budi", "budi@sikon.com", "sales")
	cust := tests.SeedCustomer(db, "Bapak Polisi", "08123", "Mabes")
	bank := tests.SeedBankAccount(db, nil, "BCA", "12345", "PT Sikon")

	// 3. Siapkan Data Transaksional (Orders & Payments)

	// Order 1: Status Production (Aktif), Total 110.000, Sudah DP 50.000
	order1 := tests.SeedOrder(db, cust.ID, sales.ID, prod.ID, "production")
	tests.SeedPayment(db, order1.ID, bank.ID, 50000, "transfer")

	// Order 2: Status Pending (Aktif), Total 110.000, Belum Bayar
	tests.SeedOrder(db, cust.ID, sales.ID, prod.ID, "pending")

	// Order 3: Status Completed (Selesai), Total 110.000, Sudah Lunas (110.000)
	order3 := tests.SeedOrder(db, cust.ID, sales.ID, prod.ID, "completed")
	tests.SeedPayment(db, order3.ID, bank.ID, 110000, "cash")

	// Order 4: Status Canceled (Batal), Total 110.000 (Tapi tidak boleh dihitung ke Revenue)
	tests.SeedOrder(db, cust.ID, sales.ID, prod.ID, "canceled")

	/* EKSPEKTASI KALKULASI ALL-TIME:
	   - Total Revenue: Order 1 + Order 2 + Order 3 = 110.000 + 110.000 + 110.000 = 330.000 (Order Batal tidak dihitung)
	   - Total Payment: Payment Order 1 (50.000) + Payment Order 3 (110.000) = 160.000
	   - Total Receivable (Piutang): Revenue (330.000) - Payment (160.000) = 170.000
	   - Active Orders: 2 (Order 1 & Order 2)
	   - Completed Orders: 1 (Order 3)
	   - Canceled Orders: 1 (Order 4)
	*/

	t.Run("Success_Get_All_Time_Summary", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dashboard/summary", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.DashboardSummaryResponse]
		respBody, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(respBody, &response)
		assert.NoError(t, err)

		// Verifikasi angka kalkulasi finansial
		assert.Equal(t, float64(330000), response.Data.TotalRevenue, "Total Omzet salah")
		assert.Equal(t, float64(160000), response.Data.TotalPaymentReceived, "Total Pembayaran Masuk salah")
		assert.Equal(t, float64(170000), response.Data.TotalReceivable, "Total Piutang salah")

		// Verifikasi agregasi status order
		assert.Equal(t, int64(2), response.Data.TotalActiveOrders, "Jumlah order aktif salah")
		assert.Equal(t, int64(1), response.Data.TotalCompletedOrders, "Jumlah order selesai salah")
		assert.Equal(t, int64(1), response.Data.TotalCanceledOrders, "Jumlah order batal salah")
	})

	t.Run("Success_Get_Summary_With_Date_Filter", func(t *testing.T) {
		// Mengambil tanggal hari ini untuk mensimulasikan filter (karena seeder memakai time.Now)
		todayStr := time.Now().Format("2006-01-02")
		url := "/api/dashboard/summary?start_date=" + todayStr + "&end_date=" + todayStr

		req := httptest.NewRequest("GET", url, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.DashboardSummaryResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Karena semua data di-*seed* hari ini, angka revenue & order count harusnya sama dengan All-Time
		assert.Equal(t, float64(330000), response.Data.TotalRevenue)

		// NAMUN, Piutang (Receivable) tetap harus mengambil All-Time Data
		// (Sesuai aturan akuntansi di repository: Revenue AllTime - Payment AllTime)
		assert.Equal(t, float64(170000), response.Data.TotalReceivable)
	})

	t.Run("Success_Get_Summary_With_Empty_Data_Date_Filter", func(t *testing.T) {
		// Menguji filter dengan rentang tanggal di mana TIDAK ADA transaksi (Misal tahun 2000)
		url := "/api/dashboard/summary?start_date=2000-01-01&end_date=2000-12-31"

		req := httptest.NewRequest("GET", url, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[dto.DashboardSummaryResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Karena di tahun 2000 tidak ada transaksi, Revenue, Payment, dan Count harus 0
		assert.Equal(t, float64(0), response.Data.TotalRevenue)
		assert.Equal(t, float64(0), response.Data.TotalPaymentReceived)
		assert.Equal(t, int64(0), response.Data.TotalActiveOrders)

		// TAPI PERHATIKAN: Total Receivable (Piutang) HARUS TETAP 170.000
		// Karena piutang tidak terpengaruh filter tanggal!
		assert.Equal(t, float64(170000), response.Data.TotalReceivable)
	})
}

func TestGetSalesReport_Integration(t *testing.T) {
	app, db := tests.SetupTestApp()
	tests.ClearTables(db)

	cat := tests.SeedCategory(db, "Kemeja Taktikal")
	prod := tests.SeedProduct(db, cat.ID, "Kemeja W-Tac", 100000)
	sales := tests.SeedUser(db, "Sales Budi", "budi@sikon.com", "sales")
	cust := tests.SeedCustomer(db, "Bapak Polisi", "08123", "Mabes")

	// Karena fungsi SeedOrder menggunakan time.Now(), semua order ini akan tercatat
	// pada tanggal "hari ini" saat test dijalankan.
	todayStr := time.Now().Format("2006-01-02")

	// Order 1: Pending (Nilai 110.000)
	tests.SeedOrder(db, cust.ID, sales.ID, prod.ID, "pending")
	// Order 2: Completed (Nilai 110.000)
	tests.SeedOrder(db, cust.ID, sales.ID, prod.ID, "completed")
	// Order 3: Canceled (Nilai 110.000 - Tidak boleh masuk hitungan Revenue)
	tests.SeedOrder(db, cust.ID, sales.ID, prod.ID, "canceled")

	/* EKSPEKTASI UNTUK TANGGAL HARI INI:
	   - Total Baris Array: 1 (Karena semua order dibuat di hari yang sama)
	   - Date: todayStr
	   - Total Orders: 3
	   - Completed Orders: 1
	   - Canceled Orders: 1
	   - Total Revenue: Order 1 (110k) + Order 2 (110k) = 220.000
	*/

	t.Run("Success_Get_Report_No_Filter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dashboard/sales-report", nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[[]dto.SalesReportItemResponse]
		respBody, _ := io.ReadAll(resp.Body)
		err = json.Unmarshal(respBody, &response)
		assert.NoError(t, err)

		// Verifikasi jumlah baris data tanggal (Hanya 1 hari)
		assert.Len(t, response.Data, 1)

		reportToday := response.Data[0]
		assert.Equal(t, todayStr, reportToday.Date)
		assert.Equal(t, int64(3), reportToday.TotalOrders)
		assert.Equal(t, int64(1), reportToday.CompletedOrders)
		assert.Equal(t, int64(1), reportToday.CanceledOrders)
		assert.Equal(t, float64(220000), reportToday.TotalRevenue)
	})

	t.Run("Success_Get_Report_Empty_Data_On_Date_Filter", func(t *testing.T) {
		// Filter menggunakan tanggal tahun 2000 di mana tidak ada data yang di-seed
		url := "/api/dashboard/sales-report?start_date=2000-01-01&end_date=2000-01-31"

		req := httptest.NewRequest("GET", url, nil)
		resp, err := app.Test(req, -1)

		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response utils.SuccessResponse[[]dto.SalesReportItemResponse]
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)

		// Pastikan mengembalikan array kosong [], bukan null
		assert.NotNil(t, response.Data)
		assert.Len(t, response.Data, 0)
	})
}
