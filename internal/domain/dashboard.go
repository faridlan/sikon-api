package domain

import "context"

// DashboardSummary merepresentasikan data agregasi untuk laporan halaman depan
type DashboardSummary struct {
	TotalRevenue         float64 // Total nilai pesanan (Grand Total dari order valid/approved: production, ready, completed)
	TotalPaymentReceived float64 // Total uang riil yang sudah masuk ke kas (Dari tabel payments status verified)
	TotalReceivable      float64 // Total piutang (Sisa tagihan yang belum dibayar customer)

	TotalActiveOrders    int64 // Jumlah pesanan yang sedang berjalan (Status: production & ready)
	TotalCompletedOrders int64 // Jumlah pesanan yang sudah selesai dikerjakan (Status: completed)
	TotalCanceledOrders  int64 // Jumlah pesanan yang dibatalkan (Status: canceled)
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

// 🚨 STRUCT KHUSUS BATCH PO DENGAN INFORMATION KUOTA TERPAKAINYA
type ActiveBatchPOOverview struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	Quota           int    `json:"quota"`
	TotalQtyOrdered int64  `json:"total_qty_ordered"` // Total pcs baju yang dipesan
	RemainingQuota  int64  `json:"remaining_quota"`   // Sisa kuota pcs baju yang tersedia
}

type DashboardOverview struct {
	Summary        DashboardSummary        `json:"summary"`
	ActiveBatchPO  *ActiveBatchPOOverview  `json:"active_batch_po"` // 👈 Menggunakan struct overview khusus
	ActionRequired ActionRequiredOverview  `json:"action_required"`
	ChartTrends    []SalesReportItem       `json:"chart_trends"`
	RecentOrders   []RecentOrderOverview   `json:"recent_orders"`
	RecentPayments []RecentPaymentOverview `json:"recent_payments"`
}

type DashboardFilter struct {
	StartDate string // Format: YYYY-MM-DD
	EndDate   string // Format: YYYY-MM-DD
	SalesID   string
}

type DashboardRepository interface {
	GetSummary(ctx context.Context, filter DashboardFilter) (*DashboardSummary, error)
	GetSalesReport(ctx context.Context, filter DashboardFilter) ([]SalesReportItem, error)
	GetReceivablesReport(ctx context.Context, salesID string) ([]ReceivableReportItem, error)
	GetOverview(ctx context.Context, salesID string) (*DashboardOverview, error)
}

type DashboardUsecase interface {
	GetSummary(ctx context.Context, filter DashboardFilter) (*DashboardSummary, error)
	GetSalesReport(ctx context.Context, filter DashboardFilter) ([]SalesReportItem, error)
	GetReceivablesReport(ctx context.Context, salesID string) ([]ReceivableReportItem, error)
	GetOverview(ctx context.Context, salesID string) (*DashboardOverview, error)
}
