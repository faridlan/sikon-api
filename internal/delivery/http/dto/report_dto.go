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
	TotalQtyReceived    int64   `json:"total_qty_received"`
	RemainingQuota      int64   `json:"remaining_quota"`
}

type PastDueReceivableResponse struct {
	SalesName       string  `json:"sales_name"`
	ProductCategory string  `json:"product_category"`
	CustomerName    string  `json:"customer_name"`
	UnpaidBalance   float64 `json:"unpaid_balance"`
}

type DailyReportResponse struct {
	ReportDate       string                   `json:"report_date"`
	POInfo           POInfoResponse           `json:"po_info"`
	OrderSummary     OrderSummaryResponse     `json:"order_summary"`
	FinancialSummary FinancialSummaryResponse `json:"financial_summary"`
	SalesDetails     []SalesDetailResponse    `json:"sales_details"`
}

type POInfoResponse struct {
	POID           string `json:"po_id"`
	POName         string `json:"po_name"`
	Quota          int    `json:"quota"`
	RemainingQuota int64  `json:"remaining_quota"`
}

type DailyTrendResponse struct {
	Date string `json:"date"`
	Qty  int64  `json:"qty"`
}

type OrderSummaryResponse struct {
	QtyToday   int64                `json:"qty_today"`
	QtyTotalPO int64                `json:"qty_total_po"`
	TrendData  []DailyTrendResponse `json:"trend_data"`
}

type FinancialSummaryResponse struct {
	TotalRevenue          float64 `json:"total_revenue"`
	TotalPaid             float64 `json:"total_paid"`
	ActivePOOutstanding   float64 `json:"active_po_outstanding"`
	PreviousPOOutstanding float64 `json:"previous_po_outstanding"`
	TotalOutstanding      float64 `json:"total_outstanding"`
}

type SalesDetailResponse struct {
	SalesName  string           `json:"sales_name"`
	Categories map[string]int64 `json:"categories"`
	TotalQty   int64            `json:"total_qty"`
}

type POSummaryResponse struct {
	POID             string                     `json:"po_id"`
	POName           string                     `json:"po_name"`
	StartDate        string                     `json:"start_date"`
	EndDate          string                     `json:"end_date"`
	Status           string                     `json:"status"`
	TotalQuota       int                        `json:"total_quota"`
	TotalQtyOrdered  int64                      `json:"total_qty_ordered"`
	TotalRevenue     float64                    `json:"total_revenue"`
	TotalPaid        float64                    `json:"total_paid"`
	TotalOutstanding float64                    `json:"total_outstanding"`
	ProductSummary   []POProductSummaryResponse `json:"product_summary"`
	SalesSummary     []POSalesSummaryResponse   `json:"sales_summary"`
}

type POProductSummaryResponse struct {
	CategoryName string `json:"category_name"`
	TotalQty     int64  `json:"total_qty"`
}

type POSalesSummaryResponse struct {
	SalesName    string  `json:"sales_name"`
	TotalQty     int64   `json:"total_qty"`
	TotalRevenue float64 `json:"total_revenue"`
}

type ReceivableDetailResponse struct {
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
			TotalQtyReceived:    item.TotalQtyReceived,
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

func ToDailyReportResponse(src *domain.DailyReport) DailyReportResponse {
	if src == nil {
		return DailyReportResponse{}
	}

	trendData := make([]DailyTrendResponse, 0, len(src.OrderSummary.TrendData))
	for _, t := range src.OrderSummary.TrendData {
		trendData = append(trendData, DailyTrendResponse{
			Date: t.Date,
			Qty:  t.Qty,
		})
	}

	salesDetails := make([]SalesDetailResponse, 0, len(src.SalesDetails))
	for _, sd := range src.SalesDetails {
		salesDetails = append(salesDetails, SalesDetailResponse{
			SalesName:  sd.SalesName,
			Categories: sd.Categories,
			TotalQty:   sd.TotalQty,
		})
	}

	return DailyReportResponse{
		ReportDate: src.ReportDate,
		POInfo: POInfoResponse{
			POID:           src.POInfo.POID,
			POName:         src.POInfo.POName,
			Quota:          src.POInfo.Quota,
			RemainingQuota: src.POInfo.RemainingQuota,
		},
		OrderSummary: OrderSummaryResponse{
			QtyToday:   src.OrderSummary.QtyToday,
			QtyTotalPO: src.OrderSummary.QtyTotalPO,
			TrendData:  trendData,
		},
		FinancialSummary: FinancialSummaryResponse{
			TotalRevenue:          src.FinancialSummary.TotalRevenue,
			TotalPaid:             src.FinancialSummary.TotalPaid,
			ActivePOOutstanding:   src.FinancialSummary.ActivePOOutstanding,
			PreviousPOOutstanding: src.FinancialSummary.PreviousPOOutstanding,
			TotalOutstanding:      src.FinancialSummary.TotalOutstanding,
		},
		SalesDetails: salesDetails,
	}
}

func ToPOSummaryResponse(src *domain.POSummaryReport) POSummaryResponse {
	if src == nil {
		return POSummaryResponse{}
	}

	prodSummary := make([]POProductSummaryResponse, 0, len(src.ProductSummary))
	for _, p := range src.ProductSummary {
		prodSummary = append(prodSummary, POProductSummaryResponse{
			CategoryName: p.CategoryName,
			TotalQty:     p.TotalQty,
		})
	}

	salesSummary := make([]POSalesSummaryResponse, 0, len(src.SalesSummary))
	for _, s := range src.SalesSummary {
		salesSummary = append(salesSummary, POSalesSummaryResponse{
			SalesName:    s.SalesName,
			TotalQty:     s.TotalQty,
			TotalRevenue: s.TotalRevenue,
		})
	}

	return POSummaryResponse{
		POID:             src.POID,
		POName:           src.POName,
		StartDate:        src.StartDate.Format("2006-01-02"),
		EndDate:          src.EndDate.Format("2006-01-02"),
		Status:           string(src.Status),
		TotalQuota:       src.TotalQuota,
		TotalQtyOrdered:  src.TotalQtyOrdered,
		TotalRevenue:     src.TotalRevenue,
		TotalPaid:        src.TotalPaid,
		TotalOutstanding: src.TotalOutstanding,
		ProductSummary:   prodSummary,
		SalesSummary:     salesSummary,
	}
}

func ToReceivableDetailListResponse(data []domain.ReceivableDetail) []ReceivableDetailResponse {
	var responses []ReceivableDetailResponse
	for _, v := range data {
		responses = append(responses, ReceivableDetailResponse{
			OrderID:           v.OrderID,
			OrderNumber:       v.OrderNumber,
			POName:            v.POName,
			POStatus:          v.POStatus,
			CustomerName:      v.CustomerName,
			SalesName:         v.SalesName,
			TotalAmount:       v.TotalAmount,
			TotalPaid:         v.TotalPaid,
			OutstandingAmount: v.OutstandingAmount,
		})
	}

	if responses == nil {
		return []ReceivableDetailResponse{}
	}

	return responses
}
