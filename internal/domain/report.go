package domain

import (
	"context"
	"time"
)

type ReportResponse struct {
	DailySnapshot               DailySnapshot       `json:"daily_snapshot"`
	ActivePOs                   []ActivePO          `json:"active_pos"`
	TotalOutstandingReceivables float64             `json:"total_outstanding_receivables"`
	ActivePOReceivables         float64             `json:"active_po_receivables"`
	PastDueReceivables          []PastDueReceivable `json:"past_due_receivables"`
}

type DailySnapshot struct {
	TotalRevenueToday     float64                 `json:"total_revenue_today"`
	TotalQtyToday         int64                   `json:"total_qty_today"`
	SalesPerformanceToday []SalesPerformanceToday `json:"sales_performance_today"`
}

type SalesPerformanceToday struct {
	SalesName       string `json:"sales_name"`
	ProductCategory string `json:"product_category"`
	TotalQty        int64  `json:"total_qty"`
}

type ActivePO struct {
	BatchPOName         string  `json:"batch_po_name"`
	BatchPOID           string  `json:"batch_po_id"`
	TotalRevenueEntered float64 `json:"total_revenue_entered"`
	TotalQtyReceived    int64   `json:"total_qty_received"`
	RemainingQuota      int64   `json:"remaining_quota"`
}

type PastDueReceivable struct {
	SalesName       string  `json:"sales_name"`
	ProductCategory string  `json:"product_category"`
	CustomerName    string  `json:"customer_name"`
	UnpaidBalance   float64 `json:"unpaid_balance"`
}

type ReportRepository interface {
	GetDailyRevenueAndQty(ctx context.Context, startOfDay, endOfDay time.Time) (float64, int64, error)
	GetSalesPerformanceByActivePO(ctx context.Context) ([]SalesPerformanceToday, error)
	GetActivePOStats(ctx context.Context) ([]ActivePO, error)
	GetReceivablesStats(ctx context.Context) (float64, float64, error)
	GetPastDueReceivables(ctx context.Context) ([]PastDueReceivable, error)
}

type ReportUsecase interface {
	GetDailyReport(ctx context.Context, date string) (*ReportResponse, error)
}
