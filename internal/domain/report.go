package domain

import (
	"context"
	"time"
)

// ============================================================================
// 1. JALUR AKUNTANSI (FINANCIAL / CALENDAR BASED)
// ============================================================================

type AccountingReport struct {
	StartDate         time.Time                    `json:"start_date"`
	EndDate           time.Time                    `json:"end_date"`
	PeriodName        string                       `json:"period_name"` // Contoh: "01 Juli 2026 - 31 Juli 2026"
	Summary           AccountingSummary            `json:"summary"`
	DailyTrends       []AccountingDailyTrend       `json:"daily_trends"`
	SalesPerformances []AccountingSalesPerformance `json:"sales_performances"`
}

type AccountingSummary struct {
	TotalOmset      float64 `json:"total_omset"`      // Gross Sales / Revenue
	TotalCashIn     float64 `json:"total_cash_in"`    // Real Uang Masuk dari Payment Verified
	TotalReceivable float64 `json:"total_receivable"` // Total Piutang Aktif
	TotalOrderCount int     `json:"total_order_count"`
	TotalItemQty    int     `json:"total_item_qty"`

	// METRIK FINANSIAL LENGKAP (HPP vs OPEX)
	TotalHPP    float64 `json:"total_hpp"`    // Direct Costs (Belanja Kain, Maklon Jahit, Potong, Bordir)
	GrossProfit float64 `json:"gross_profit"` // TotalOmset - TotalHPP
	TotalOPEX   float64 `json:"total_opex"`   // Overhead/Operasional (Gaji Admin, Listrik, Plastik, Benang)
	NetProfit   float64 `json:"net_profit"`   // GrossProfit - TotalOPEX
	NetCashflow float64 `json:"net_cashflow"` // TotalCashIn - (TotalHPP + TotalOPEX)
}

type AccountingDailyTrend struct {
	Date        string  `json:"date"`
	OmsetAmount float64 `json:"omset_amount"`
	CashIn      float64 `json:"cash_in"`
}

type AccountingSalesPerformance struct {
	SalesID     string  `json:"sales_id"`
	SalesName   string  `json:"sales_name"`
	TotalOmset  float64 `json:"total_omset"`
	TotalOrders int     `json:"total_orders"`
}

// ============================================================================
// 2. JALUR PRODUKSI (PO EDITION BASED)
// ============================================================================

type ProductionReport struct {
	TargetMonth      int     `json:"target_month"`
	TargetYear       int     `json:"target_year"`
	PeriodName       string  `json:"period_name"` // Contoh: "Edisi Juli 2026"
	TotalQuota       int     `json:"total_quota"`
	TotalQtyOrdered  int64   `json:"total_qty_ordered"`
	RemainingQuota   int64   `json:"remaining_quota"`
	TotalRevenue     float64 `json:"total_revenue"`
	TotalPaid        float64 `json:"total_paid"`
	TotalOutstanding float64 `json:"total_outstanding"`

	TotalHPP    float64 `json:"total_hpp"`
	GrossProfit float64 `json:"gross_profit"` // TotalRevenue - TotalHPP

	ActiveBatchPOs []ProductionBatchPO `json:"active_batch_pos"`
	SalesSummary   []POSalesSummary    `json:"sales_summary"`
	ProductSummary []POProductSummary  `json:"product_summary"`
	TrendData      []DailyTrend        `json:"trend_data"`
}

type ProductionBatchPO struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	Status BatchPOStatus `json:"status"`
	Quota  int           `json:"quota"`
}

// ============================================================================
// 3. JALUR SPESIFIK (PO SUMMARY & RECEIVABLE DETAIL)
// ============================================================================

type POSummaryReport struct {
	POID             string        `json:"po_id"`
	POName           string        `json:"po_name"`
	StartDate        time.Time     `json:"start_date"`
	EndDate          time.Time     `json:"end_date"`
	Status           BatchPOStatus `json:"status"`
	TotalQuota       int           `json:"total_quota"`
	RemainingQuota   int64         `json:"remaining_quota"`
	TotalQtyOrdered  int64         `json:"total_qty_ordered"`
	TotalRevenue     float64       `json:"total_revenue"`
	TotalPaid        float64       `json:"total_paid"`
	TotalOutstanding float64       `json:"total_outstanding"`

	TotalHPP    float64 `json:"total_hpp"`
	GrossProfit float64 `json:"gross_profit"` // TotalRevenue - TotalHPP

	CustomerReceivables []POCustomerReceivable `json:"customer_receivables"`
	ProductSummary      []POProductSummary     `json:"product_summary"`
	SalesSummary        []POSalesSummary       `json:"sales_summary"`
	TrendData           []DailyTrend           `json:"trend_data"`
}

type POCustomerReceivable struct {
	CustomerName      string  `json:"customer_name"`
	TotalAmount       float64 `json:"total_amount"`
	TotalPaid         float64 `json:"total_paid"`
	OutstandingAmount float64 `json:"outstanding_amount"`
}

type POProductSummary struct {
	CategoryName string `json:"category_name"`
	TotalQty     int64  `json:"total_qty"`
}

type POSalesSummary struct {
	SalesName    string           `json:"sales_name"`
	TotalQty     int64            `json:"total_qty"`
	TotalRevenue float64          `json:"total_revenue"`
	Categories   map[string]int64 `json:"categories"`
}

type ReceivableDetail struct {
	OrderID           string  `json:"order_id"`
	OrderNumber       string  `json:"order_number"`
	POName            string  `json:"po_name"`
	POStatus          string  `json:"po_status"`
	CustomerName      string  `json:"customer_name"`
	SalesName         string  `json:"sales_name"`
	TotalAmount       float64 `json:"total_amount"`
	TotalPaid         float64 `json:"total_paid"`
	OutstandingAmount float64 `json:"outstanding_amount"`
}

// ============================================================================
// 4. JALUR HARIAN (DAILY TACTICAL REPORT)
// ============================================================================

type DailyReport struct {
	ReportDate       string                `json:"report_date"` // YYYY-MM-DD
	POInfo           DailyPOInfo           `json:"po_info"`
	OrderSummary     DailyOrderSummary     `json:"order_summary"`
	FinancialSummary DailyFinancialSummary `json:"financial_summary"`
	SalesDetails     []DailySalesDetail    `json:"sales_details"`
	POSalesDetails   []DailySalesDetail    `json:"po_sales_details"`
}

type DailyPOInfo struct {
	POID           string `json:"po_id"`
	POName         string `json:"po_name"`
	Quota          int    `json:"quota"`
	RemainingQuota int64  `json:"remaining_quota"`
}

type DailyOrderSummary struct {
	QtyToday   int64        `json:"qty_today"`
	QtyTotalPO int64        `json:"qty_total_po"`
	TrendData  []DailyTrend `json:"trend_data"`
}

type DailyTrend struct {
	Date string `json:"date"`
	Qty  int64  `json:"qty"`
}

type DailyFinancialSummary struct {
	TotalRevenue          float64 `json:"total_revenue"`
	TotalPaid             float64 `json:"total_paid"`
	ActivePOOutstanding   float64 `json:"active_po_outstanding"`   // Piutang khusus PO yang sedang jalan
	PreviousPOOutstanding float64 `json:"previous_po_outstanding"` // Piutang dari PO-PO sebelumnya yang belum lunas
	TotalOutstanding      float64 `json:"total_outstanding"`       // Total semua piutang (Active + Previous)
}

type DailySalesDetail struct {
	SalesName  string           `json:"sales_name"`
	Categories map[string]int64 `json:"categories"`
	TotalQty   int64            `json:"total_qty"`
}

type ReceivablesFilter struct {
	BatchPoID   string `json:"batch_po_id"`  // Filter spesifik berdasarkan PO tertentu
	OrderStatus string `json:"order_status"` // Filter berdasarkan status order
	SortBy      string `json:"sort_by"`      // Opsi sorting: "amount_desc", "amount_asc", "date_asc", "date_desc"
}

// ============================================================================
// 5. INTERFACE KONTRAK (REPOSITORIES & USECASES)
// ============================================================================

type ReportRepository interface {
	// Laporan Utama
	GetAccountingReportData(ctx context.Context, startDate, endDate time.Time) (*AccountingReport, error)
	GetProductionReportData(ctx context.Context, targetMonth, targetYear int) (*ProductionReport, error)

	// Laporan Spesifik
	GetPOSummaryData(ctx context.Context, poID string) (*POSummaryReport, error)
	GetReceivablesDetailData(ctx context.Context, filter ReceivablesFilter) ([]ReceivableDetail, error)
	GetDailyReportData(ctx context.Context, date time.Time) (*DailyReport, error)

	// Helper terpisah untuk mengambil total Pengeluaran berdasarkan Tipe (HPP vs OPEX)
	GetExpenseBreakdownByDateRange(ctx context.Context, startDate, endDate time.Time) (totalHPP float64, totalOPEX float64, err error)
	GetTotalExpenseByBatchPOs(ctx context.Context, poIDs []string) (float64, error)
}

type ReportUsecase interface {
	GetAccountingReport(ctx context.Context, startDate, endDate time.Time) (*AccountingReport, error)
	GetProductionReport(ctx context.Context, targetMonth, targetYear int) (*ProductionReport, error)
	GetPOSummaryReport(ctx context.Context, poID string) (*POSummaryReport, error)
	GetReceivablesDetailReport(ctx context.Context, filter ReceivablesFilter) ([]ReceivableDetail, error)
	GetDailyReport(ctx context.Context, date time.Time) (*DailyReport, error)
}
