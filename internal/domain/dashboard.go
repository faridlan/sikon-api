package domain

import "context"

// DashboardSummary merepresentasikan data agregasi untuk laporan halaman depan
type DashboardSummary struct {
	TotalRevenue         float64 // Total nilai pesanan (Grand Total dari order yang tidak batal)
	TotalPaymentReceived float64 // Total uang riil yang sudah masuk ke kas (Dari tabel payments)
	TotalReceivable      float64 // Total piutang (Sisa tagihan yang belum dibayar customer)

	TotalActiveOrders    int64 // Jumlah pesanan yang sedang berjalan (Status: pending & production)
	TotalCompletedOrders int64 // Jumlah pesanan yang sudah selesai dikerjakan
	TotalCanceledOrders  int64 // Jumlah pesanan yang dibatalkan
}

type SalesReportItem struct {
	Date            string  `json:"date"` // Format: YYYY-MM-DD
	TotalRevenue    float64 `json:"total_revenue"`
	TotalOrders     int64   `json:"total_orders"`
	CompletedOrders int64   `json:"completed_orders"`
	CanceledOrders  int64   `json:"canceled_orders"`
}

// ReceivableReportItem merepresentasikan baris data pesanan yang belum lunas (Piutang)
type ReceivableReportItem struct {
	OrderID       string  `json:"order_id"`
	OrderNumber   string  `json:"order_number"`
	OrderDate     string  `json:"order_date"`
	CustomerName  string  `json:"customer_name"`
	SalesName     string  `json:"sales_name"`
	OrderStatus   string  `json:"order_status"`
	TotalAmount   float64 `json:"total_amount"`
	TotalPaid     float64 `json:"total_paid"`
	RemainingBill float64 `json:"remaining_bill"`
}

// ========================================================================
// 🚨 TAMBAHAN STRUCT DTO / ENTITY DEDICATED UNTUK SINGLE ENDPOINT OVERVIEW
// ========================================================================

type ActionRequiredOverview struct {
	UnverifiedPaymentsCount int64 `json:"unverified_payments_count"`
	ReadyOrdersCount        int64 `json:"ready_orders_count"`
	PendingOrdersCount      int64 `json:"pending_orders_count"`
}

type RecentOrderOverview struct {
	ID           string  `json:"id"`
	OrderNumber  string  `json:"order_number"`
	CustomerName string  `json:"customer_name"`
	OrderStatus  string  `json:"order_status"`
	TotalAmount  float64 `json:"total_amount"`
}

type RecentPaymentOverview struct {
	ID          string  `json:"id"`
	PaymentDate string  `json:"payment_date"`
	PaymentType string  `json:"payment_type"`
	Status      string  `json:"status"`
	Amount      float64 `json:"amount"`
}

type DashboardOverview struct {
	Summary        DashboardSummary        `json:"summary"`
	ActiveBatchPO  *BatchPO                `json:"active_batch_po"`
	ActionRequired ActionRequiredOverview  `json:"action_required"`
	ChartTrends    []SalesReportItem       `json:"chart_trends"`
	RecentOrders   []RecentOrderOverview   `json:"recent_orders"`
	RecentPayments []RecentPaymentOverview `json:"recent_payments"`
}

// DashboardFilter untuk menyaring laporan berdasarkan rentang waktu (harian, bulanan, tahunan)
type DashboardFilter struct {
	StartDate string // Format: YYYY-MM-DD
	EndDate   string // Format: YYYY-MM-DD
	SalesID   string
}

// DashboardRepository adalah kontrak untuk layer database (GORM)
type DashboardRepository interface {
	GetSummary(ctx context.Context, filter DashboardFilter) (*DashboardSummary, error)
	GetSalesReport(ctx context.Context, filter DashboardFilter) ([]SalesReportItem, error)
	GetReceivablesReport(ctx context.Context, salesID string) ([]ReceivableReportItem, error)
	GetOverview(ctx context.Context, salesID string) (*DashboardOverview, error) // 👈 1. TAMBAHKAN METHOD INI
}

// DashboardUsecase adalah kontrak untuk layer logika bisnis
type DashboardUsecase interface {
	GetSummary(ctx context.Context, filter DashboardFilter) (*DashboardSummary, error)
	GetSalesReport(ctx context.Context, filter DashboardFilter) ([]SalesReportItem, error)
	GetReceivablesReport(ctx context.Context, salesID string) ([]ReceivableReportItem, error)
	GetOverview(ctx context.Context, salesID string) (*DashboardOverview, error) // 👈 2. TAMBAHKAN METHOD INI
}
