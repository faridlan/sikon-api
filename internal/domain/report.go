package domain

import (
	"context"
	"time"
)

// ============================================================================
// 1. JALUR AKUNTANSI (FINANCIAL / CALENDAR BASED)
// Menggantikan fungsi lama Laporan Harian & Bulanan yang redundan
// ============================================================================

type AccountingReport struct {
	StartDate         time.Time
	EndDate           time.Time
	PeriodName        string // Contoh: "01 Juli 2026 - 31 Juli 2026"
	Summary           AccountingSummary
	DailyTrends       []AccountingDailyTrend
	SalesPerformances []AccountingSalesPerformance
}

type AccountingSummary struct {
	TotalOmset      float64
	TotalCashIn     float64
	TotalReceivable float64
	TotalOrderCount int
	TotalItemQty    int

	// TAMBAHAN METRIK FINANSIAL PENGELUARAN
	TotalExpense float64
	NetProfit    float64
	NetCashflow  float64
}

type AccountingDailyTrend struct {
	Date        string
	OmsetAmount float64
	CashIn      float64
}

type AccountingSalesPerformance struct {
	SalesID     string
	SalesName   string
	TotalOmset  float64
	TotalOrders int
}

// ============================================================================
// 2. JALUR PRODUKSI (PO EDITION BASED)
// Fokus pada performa produksi berdasarkan Target Month & Target Year PO
// ============================================================================

type ProductionReport struct {
	TargetMonth      int
	TargetYear       int
	PeriodName       string // Contoh: "Edisi Juli 2026"
	TotalQuota       int
	TotalQtyOrdered  int64
	RemainingQuota   int64
	TotalRevenue     float64
	TotalPaid        float64
	TotalOutstanding float64

	// TAMBAHAN METRIK FINANSIAL PENGELUARAN
	TotalHPP  float64
	NetProfit float64

	ActiveBatchPOs []ProductionBatchPO // Daftar PO apa saja yang masuk di edisi ini
	SalesSummary   []POSalesSummary
	ProductSummary []POProductSummary
}

type ProductionBatchPO struct {
	ID     string
	Name   string
	Status BatchPOStatus
	Quota  int
}

// ============================================================================
// 3. JALUR SPESIFIK (TETAP DIPERTAHANKAN KARENA DIBUTUHKAN)
// ============================================================================

type POSummaryReport struct {
	POID             string
	POName           string
	StartDate        time.Time
	EndDate          time.Time
	Status           BatchPOStatus
	TotalQuota       int
	TotalQtyOrdered  int64
	TotalRevenue     float64
	TotalPaid        float64
	TotalOutstanding float64
	ProductSummary   []POProductSummary
	SalesSummary     []POSalesSummary
}

type POProductSummary struct {
	CategoryName string
	TotalQty     int64
}

type POSalesSummary struct {
	SalesName    string
	TotalQty     int64
	TotalRevenue float64
}

type ReceivableDetail struct {
	OrderID           string
	OrderNumber       string
	POName            string
	POStatus          string
	CustomerName      string
	SalesName         string
	TotalAmount       float64
	TotalPaid         float64
	OutstandingAmount float64
}

// ============================================================================
// 4. INTERFACE KONTRAK (REPOSITORIES & USECASES)
// ============================================================================

type ReportRepository interface {
	// Laporan Utama
	GetAccountingReportData(ctx context.Context, startDate, endDate time.Time) (*AccountingReport, error)
	GetProductionReportData(ctx context.Context, targetMonth, targetYear int) (*ProductionReport, error)

	// Laporan Spesifik
	GetPOSummaryData(ctx context.Context, poID string) (*POSummaryReport, error)
	GetReceivablesDetailData(ctx context.Context) ([]ReceivableDetail, error)

	// TAMBAHAN: Helper untuk mengambil total Pengeluaran (Expenses)
	GetTotalExpenseByDateRange(ctx context.Context, startDate, endDate time.Time) (float64, error)
	GetTotalExpenseByBatchPOs(ctx context.Context, poIDs []string) (float64, error)
}

type ReportUsecase interface {
	GetAccountingReport(ctx context.Context, startDate, endDate time.Time) (*AccountingReport, error)
	GetProductionReport(ctx context.Context, targetMonth, targetYear int) (*ProductionReport, error)
	GetPOSummaryReport(ctx context.Context, poID string) (*POSummaryReport, error)
	GetReceivablesDetailReport(ctx context.Context) ([]ReceivableDetail, error)
}
