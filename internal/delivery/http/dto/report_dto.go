package dto

import "github.com/faridlan/sikon-api/internal/domain"

type ReportResponse struct {
	DailySnapshot               DailySnapshotResponse       `json:"daily_snapshot"`
	ActivePOs                   []ActivePOResponse          `json:"active_pos"`
	TotalOutstandingReceivables float64                     `json:"total_outstanding_receivables"`
	ActivePOReceivables         float64                     `json:"active_po_receivables"`
	PastDueReceivables          []PastDueReceivableResponse `json:"past_due_receivables"`
}

type DailySnapshotResponse struct {
	TotalRevenueToday     float64                         `json:"total_revenue_today"`
	TotalQtyToday         int64                           `json:"total_qty_today"`
	SalesPerformanceToday []SalesPerformanceTodayResponse `json:"sales_performance_today"`
}

type SalesPerformanceTodayResponse struct {
	SalesName       string `json:"sales_name"`
	ProductCategory string `json:"product_category"`
	TotalQty        int64  `json:"total_qty"`
}

type ActivePOResponse struct {
	BatchPOName         string  `json:"batch_po_name"`
	BatchPOID           string  `json:"batch_po_id"`
	TotalRevenueEntered float64 `json:"total_revenue_entered"`
	RemainingQuota      int64   `json:"remaining_quota"`
}

type PastDueReceivableResponse struct {
	SalesName       string  `json:"sales_name"`
	ProductCategory string  `json:"product_category"`
	CustomerName    string  `json:"customer_name"`
	UnpaidBalance   float64 `json:"unpaid_balance"`
}

func ToReportResponse(src *domain.ReportResponse) ReportResponse {
	if src == nil {
		return ReportResponse{}
	}

	resp := ReportResponse{
		DailySnapshot: DailySnapshotResponse{
			TotalRevenueToday:     src.DailySnapshot.TotalRevenueToday,
			TotalQtyToday:         src.DailySnapshot.TotalQtyToday,
			SalesPerformanceToday: make([]SalesPerformanceTodayResponse, 0, len(src.DailySnapshot.SalesPerformanceToday)),
		},
		ActivePOs:                   make([]ActivePOResponse, 0, len(src.ActivePOs)),
		TotalOutstandingReceivables: src.TotalOutstandingReceivables,
		ActivePOReceivables:         src.ActivePOReceivables,
		PastDueReceivables:          make([]PastDueReceivableResponse, 0, len(src.PastDueReceivables)),
	}

	for _, item := range src.DailySnapshot.SalesPerformanceToday {
		resp.DailySnapshot.SalesPerformanceToday = append(resp.DailySnapshot.SalesPerformanceToday, SalesPerformanceTodayResponse{
			SalesName:       item.SalesName,
			ProductCategory: item.ProductCategory,
			TotalQty:        item.TotalQty,
		})
	}

	for _, item := range src.ActivePOs {
		resp.ActivePOs = append(resp.ActivePOs, ActivePOResponse{
			BatchPOName:         item.BatchPOName,
			BatchPOID:           item.BatchPOID,
			TotalRevenueEntered: item.TotalRevenueEntered,
			RemainingQuota:      item.RemainingQuota,
		})
	}

	for _, item := range src.PastDueReceivables {
		resp.PastDueReceivables = append(resp.PastDueReceivables, PastDueReceivableResponse{
			SalesName:       item.SalesName,
			ProductCategory: item.ProductCategory,
			CustomerName:    item.CustomerName,
			UnpaidBalance:   item.UnpaidBalance,
		})
	}

	return resp
}
