package domain

import (
	"context"
	"time"
)

// ReportResponse represents the aggregated report data for a specific date.
type ReportResponse struct {
	DailySnapshot               DailySnapshot       `json:"daily_snapshot"`
	ActivePOs                   []ActivePO          `json:"active_pos"`
	TotalOutstandingReceivables float64             `json:"total_outstanding_receivables"`
	ActivePOReceivables         float64             `json:"active_po_receivables"`
	PastDueReceivables          []PastDueReceivable `json:"past_due_receivables"`
}

// DailySnapshot represents the daily revenue and quantity snapshot.
type DailySnapshot struct {
	TotalRevenueToday     float64                 `json:"total_revenue_today"`
	TotalQtyToday         int64                   `json:"total_qty_today"`
	SalesPerformanceToday []SalesPerformanceToday `json:"sales_performance_today"`
}

// SalesPerformanceToday represents the sales performance for a specific sales person and product category.
type SalesPerformanceToday struct {
	SalesName       string `json:"sales_name"`
	ProductCategory string `json:"product_category"`
	TotalQty        int64  `json:"total_qty"`
}

// ActivePO represents the active purchase order details along with revenue and quantity information.
type ActivePO struct {
	BatchPOName         string  `json:"batch_po_name"`
	BatchPOID           string  `json:"batch_po_id"`
	TotalRevenueEntered float64 `json:"total_revenue_entered"`
	TotalQtyReceived    int64   `json:"total_qty_received"`
	RemainingQuota      int64   `json:"remaining_quota"`
}

// PastDueReceivable represents the details of past due receivables for a specific sales person, product category, and customer.
type PastDueReceivable struct {
	SalesName       string  `json:"sales_name"`
	ProductCategory string  `json:"product_category"`
	CustomerName    string  `json:"customer_name"`
	UnpaidBalance   float64 `json:"unpaid_balance"`
}

// ReportRepository defines the interface for accessing report-related data from the data source.
type ReportRepository interface {
	GetDailyRevenueAndQty(ctx context.Context, startOfDay, endOfDay time.Time) (float64, int64, error)
	GetSalesPerformanceByActivePO(ctx context.Context) ([]SalesPerformanceToday, error)
	GetActivePOStats(ctx context.Context) ([]ActivePO, error)
	GetReceivablesStats(ctx context.Context) (float64, float64, error)
	GetPastDueReceivables(ctx context.Context) ([]PastDueReceivable, error)
}

// ReportUsecase defines the interface for the report use case, which provides methods to generate reports based on the data retrieved from the repository.
type ReportUsecase interface {
	GetDailyReport(ctx context.Context, date string) (*ReportResponse, error)
}
