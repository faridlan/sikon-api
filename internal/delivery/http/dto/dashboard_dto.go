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

type SalesReportItemResponse struct {
	Date            string  `json:"date"`
	TotalRevenue    float64 `json:"total_revenue"`
	TotalOrders     int64   `json:"total_orders"`
	CompletedOrders int64   `json:"completed_orders"`
	CanceledOrders  int64   `json:"canceled_orders"`
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

func ToSalesReportItemResponseList(data []domain.SalesReportItem) []SalesReportItemResponse {
	var result []SalesReportItemResponse

	for _, item := range data {
		result = append(result, SalesReportItemResponse{
			Date:            item.Date,
			TotalRevenue:    item.TotalRevenue,
			TotalOrders:     item.TotalOrders,
			CompletedOrders: item.CompletedOrders,
			CanceledOrders:  item.CanceledOrders,
		})
	}

	// Pastikan mengembalikan array kosong [] jika tidak ada data, BUKAN null
	if result == nil {
		result = []SalesReportItemResponse{}
	}

	return result
}
