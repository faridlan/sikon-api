package dto

import "github.com/faridlan/sikon-api/internal/domain"

type DashboardSummaryResponse struct {
	TotalRevenue         float64 `json:"total_revenue"`
	TotalPaymentReceived float64 `json:"total_payment_received"`
	TotalReceivable      float64 `json:"total_receivable"`

	TotalActiveOrders    int64 `json:"total_active_orders"`
	TotalCompletedOrders int64 `json:"total_completed_orders"`
	TotalCanceledOrders  int64 `json:"total_canceled_orders"`
}

func ToDashboardSummaryResponse(d *domain.DashboardSummary) DashboardSummaryResponse {
	return DashboardSummaryResponse{
		TotalRevenue:         d.TotalRevenue,
		TotalPaymentReceived: d.TotalPaymentReceived,
		TotalReceivable:      d.TotalReceivable,
		TotalActiveOrders:    d.TotalActiveOrders,
		TotalCompletedOrders: d.TotalCompletedOrders,
		TotalCanceledOrders:  d.TotalCanceledOrders,
	}
}
