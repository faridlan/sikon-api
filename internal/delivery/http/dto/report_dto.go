package dto

import (
	"github.com/faridlan/sikon-api/internal/domain"
)

// ============================================================================
// 1. DTO JALUR AKUNTANSI (FINANCIAL / CALENDAR BASED)
// ============================================================================

type AccountingReportResponse struct {
	StartDate         string                               `json:"start_date"`
	EndDate           string                               `json:"end_date"`
	PeriodName        string                               `json:"period_name"`
	Summary           AccountingSummaryResponse            `json:"summary"`
	DailyTrends       []AccountingDailyTrendResponse       `json:"daily_trends"`
	SalesPerformances []AccountingSalesPerformanceResponse `json:"sales_performances"`
}

type AccountingSummaryResponse struct {
	TotalOmset      float64 `json:"total_omset"`
	TotalCashIn     float64 `json:"total_cash_in"`
	TotalReceivable float64 `json:"total_receivable"`
	TotalOrderCount int     `json:"total_order_count"`
	TotalItemQty    int     `json:"total_item_qty"`

	// METRIK FINANSIAL LENGKAP (HPP vs OPEX)
	TotalHPP    float64 `json:"total_hpp"`
	GrossProfit float64 `json:"gross_profit"`
	TotalOPEX   float64 `json:"total_opex"`
	NetProfit   float64 `json:"net_profit"`
	NetCashflow float64 `json:"net_cashflow"`
}

type AccountingDailyTrendResponse struct {
	Date        string  `json:"date"`
	OmsetAmount float64 `json:"omset_amount"`
	CashIn      float64 `json:"cash_in"`
}

type AccountingSalesPerformanceResponse struct {
	SalesID     string  `json:"sales_id"`
	SalesName   string  `json:"sales_name"`
	TotalOmset  float64 `json:"total_omset"`
	TotalOrders int     `json:"total_orders"`
}

func ToAccountingReportResponse(src *domain.AccountingReport) *AccountingReportResponse {
	if src == nil {
		return nil
	}

	resp := &AccountingReportResponse{
		StartDate:  src.StartDate.Format("2006-01-02"),
		EndDate:    src.EndDate.Format("2006-01-02"),
		PeriodName: src.PeriodName,
		Summary: AccountingSummaryResponse{
			TotalOmset:      src.Summary.TotalOmset,
			TotalCashIn:     src.Summary.TotalCashIn,
			TotalReceivable: src.Summary.TotalReceivable,
			TotalOrderCount: src.Summary.TotalOrderCount,
			TotalItemQty:    src.Summary.TotalItemQty,
			TotalHPP:        src.Summary.TotalHPP,
			GrossProfit:     src.Summary.GrossProfit,
			TotalOPEX:       src.Summary.TotalOPEX,
			NetProfit:       src.Summary.NetProfit,
			NetCashflow:     src.Summary.NetCashflow,
		},
		DailyTrends:       make([]AccountingDailyTrendResponse, 0),
		SalesPerformances: make([]AccountingSalesPerformanceResponse, 0),
	}

	for _, v := range src.DailyTrends {
		resp.DailyTrends = append(resp.DailyTrends, AccountingDailyTrendResponse(v))
	}

	for _, v := range src.SalesPerformances {
		resp.SalesPerformances = append(resp.SalesPerformances, AccountingSalesPerformanceResponse(v))
	}

	return resp
}

// ============================================================================
// 2. DTO JALUR PRODUKSI (PO EDITION BASED)
// ============================================================================

type ProductionReportResponse struct {
	TargetMonth      int     `json:"target_month"`
	TargetYear       int     `json:"target_year"`
	PeriodName       string  `json:"period_name"`
	TotalQuota       int     `json:"total_quota"`
	TotalQtyOrdered  int64   `json:"total_qty_ordered"`
	RemainingQuota   int64   `json:"remaining_quota"`
	TotalRevenue     float64 `json:"total_revenue"`
	TotalPaid        float64 `json:"total_paid"`
	TotalOutstanding float64 `json:"total_outstanding"`

	TotalHPP    float64 `json:"total_hpp"`
	GrossProfit float64 `json:"gross_profit"`

	ActiveBatchPOs []ProductionBatchPOResponse `json:"active_batch_pos"`
	ProductSummary []POProductSummaryResponse  `json:"product_summary"`
	SalesSummary   []POSalesSummaryResponse    `json:"sales_summary"`
	TrendData      []DailyTrendResponse        `json:"trend_data"`
}

type ProductionBatchPOResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Quota  int    `json:"quota"`
}

func ToProductionReportResponse(src *domain.ProductionReport) *ProductionReportResponse {
	if src == nil {
		return nil
	}

	resp := &ProductionReportResponse{
		TargetMonth:      src.TargetMonth,
		TargetYear:       src.TargetYear,
		PeriodName:       src.PeriodName,
		TotalQuota:       src.TotalQuota,
		TotalQtyOrdered:  src.TotalQtyOrdered,
		RemainingQuota:   src.RemainingQuota,
		TotalRevenue:     src.TotalRevenue,
		TotalPaid:        src.TotalPaid,
		TotalOutstanding: src.TotalOutstanding,
		TotalHPP:         src.TotalHPP,
		GrossProfit:      src.GrossProfit,

		ActiveBatchPOs: make([]ProductionBatchPOResponse, 0),
		ProductSummary: make([]POProductSummaryResponse, 0),
		SalesSummary:   make([]POSalesSummaryResponse, 0),
	}

	for _, bp := range src.ActiveBatchPOs {
		resp.ActiveBatchPOs = append(resp.ActiveBatchPOs, ProductionBatchPOResponse{
			ID:     bp.ID,
			Name:   bp.Name,
			Status: string(bp.Status),
			Quota:  bp.Quota,
		})
	}

	for _, p := range src.ProductSummary {
		resp.ProductSummary = append(resp.ProductSummary, POProductSummaryResponse{
			CategoryName: p.CategoryName,
			TotalQty:     p.TotalQty,
		})
	}

	for _, s := range src.SalesSummary {
		resp.SalesSummary = append(resp.SalesSummary, POSalesSummaryResponse{
			SalesName:    s.SalesName,
			TotalQty:     s.TotalQty,
			TotalRevenue: s.TotalRevenue,
			Categories:   s.Categories,
		})
	}

	resp.TrendData = make([]DailyTrendResponse, 0)
	for _, t := range src.TrendData {
		resp.TrendData = append(resp.TrendData, DailyTrendResponse{
			Date: t.Date,
			Qty:  t.Qty,
		})
	}

	return resp
}

// ============================================================================
// 3. DTO SPESIFIK (PO Summary & Receivable Detail)
// ============================================================================

type POSummaryResponse struct {
	POID             string  `json:"po_id"`
	POName           string  `json:"po_name"`
	StartDate        string  `json:"start_date"`
	EndDate          string  `json:"end_date"`
	Status           string  `json:"status"`
	TotalQuota       int     `json:"total_quota"`
	RemainingQuota   int64   `json:"remaining_quota"`
	TotalQtyOrdered  int64   `json:"total_qty_ordered"`
	TotalRevenue     float64 `json:"total_revenue"`
	TotalPaid        float64 `json:"total_paid"`
	TotalOutstanding float64 `json:"total_outstanding"`

	// FINANSIAL
	TotalHPP    float64 `json:"total_hpp"`
	GrossProfit float64 `json:"gross_profit"`

	CustomerReceivables []POCustomerReceivableResponse `json:"customer_receivables"`
	ProductSummary      []POProductSummaryResponse     `json:"product_summary"`
	SalesSummary        []POSalesSummaryResponse       `json:"sales_summary"`
	TrendData           []DailyTrendResponse           `json:"trend_data"`
}

type POCustomerReceivableResponse struct {
	CustomerName      string  `json:"customer_name"`
	TotalAmount       float64 `json:"total_amount"`
	TotalPaid         float64 `json:"total_paid"`
	OutstandingAmount float64 `json:"outstanding_amount"`
}

type POProductSummaryResponse struct {
	CategoryName string `json:"category_name"`
	TotalQty     int64  `json:"total_qty"`
}

type POSalesSummaryResponse struct {
	SalesName    string           `json:"sales_name"`
	TotalQty     int64            `json:"total_qty"`
	TotalRevenue float64          `json:"total_revenue"`
	Categories   map[string]int64 `json:"categories"`
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
			Categories:   s.Categories,
		})
	}

	trendData := make([]DailyTrendResponse, 0, len(src.TrendData))
	for _, t := range src.TrendData {
		trendData = append(trendData, DailyTrendResponse{
			Date: t.Date,
			Qty:  t.Qty,
		})
	}
	customerReceivables := make([]POCustomerReceivableResponse, 0, len(src.CustomerReceivables))
	for _, c := range src.CustomerReceivables {
		customerReceivables = append(customerReceivables, POCustomerReceivableResponse{
			CustomerName:      c.CustomerName,
			TotalAmount:       c.TotalAmount,
			TotalPaid:         c.TotalPaid,
			OutstandingAmount: c.OutstandingAmount,
		})
	}

	return POSummaryResponse{
		POID:                src.POID,
		POName:              src.POName,
		StartDate:           src.StartDate.Format("2006-01-02"),
		EndDate:             src.EndDate.Format("2006-01-02"),
		Status:              string(src.Status),
		TotalQuota:          src.TotalQuota,
		RemainingQuota:      src.RemainingQuota,
		TotalQtyOrdered:     src.TotalQtyOrdered,
		TotalRevenue:        src.TotalRevenue,
		TotalPaid:           src.TotalPaid,
		TotalOutstanding:    src.TotalOutstanding,
		TotalHPP:            src.TotalHPP,
		GrossProfit:         src.GrossProfit,
		CustomerReceivables: customerReceivables,
		ProductSummary:      prodSummary,
		SalesSummary:        salesSummary,
		TrendData:           trendData,
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

// ============================================================================
// 4. DTO JALUR HARIAN (DAILY REPORT)
// ============================================================================

type DailyReportResponse struct {
	ReportDate       string                        `json:"report_date"`
	POInfo           DailyPOInfoResponse           `json:"po_info"`
	OrderSummary     DailyOrderSummaryResponse     `json:"order_summary"`
	FinancialSummary DailyFinancialSummaryResponse `json:"financial_summary"`
	SalesDetails     []DailySalesDetailResponse    `json:"sales_details"`
	POSalesDetails   []DailySalesDetailResponse    `json:"po_sales_details"`
}

type DailyPOInfoResponse struct {
	POID           string `json:"po_id"`
	POName         string `json:"po_name"`
	Quota          int    `json:"quota"`
	RemainingQuota int64  `json:"remaining_quota"`
}

type DailyOrderSummaryResponse struct {
	QtyToday   int64                `json:"qty_today"`
	QtyTotalPO int64                `json:"qty_total_po"`
	TrendData  []DailyTrendResponse `json:"trend_data"`
}

type DailyTrendResponse struct {
	Date string `json:"date"`
	Qty  int64  `json:"qty"`
}

type DailyFinancialSummaryResponse struct {
	TotalRevenue          float64 `json:"total_revenue"`
	TotalPaid             float64 `json:"total_paid"`
	ActivePOOutstanding   float64 `json:"active_po_outstanding"`
	PreviousPOOutstanding float64 `json:"previous_po_outstanding"`
	TotalOutstanding      float64 `json:"total_outstanding"`
}

type DailySalesDetailResponse struct {
	SalesName  string           `json:"sales_name"`
	Categories map[string]int64 `json:"categories"`
	TotalQty   int64            `json:"total_qty"`
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

	salesDetails := make([]DailySalesDetailResponse, 0, len(src.SalesDetails))
	for _, s := range src.SalesDetails {
		salesDetails = append(salesDetails, DailySalesDetailResponse{
			SalesName:  s.SalesName,
			Categories: s.Categories,
			TotalQty:   s.TotalQty,
		})
	}

	poSalesDetails := make([]DailySalesDetailResponse, 0, len(src.POSalesDetails))
	for _, s := range src.POSalesDetails {
		poSalesDetails = append(poSalesDetails, DailySalesDetailResponse{
			SalesName:  s.SalesName,
			Categories: s.Categories,
			TotalQty:   s.TotalQty,
		})
	}

	return DailyReportResponse{
		ReportDate: src.ReportDate,
		POInfo: DailyPOInfoResponse{
			POID:           src.POInfo.POID,
			POName:         src.POInfo.POName,
			Quota:          src.POInfo.Quota,
			RemainingQuota: src.POInfo.RemainingQuota,
		},
		OrderSummary: DailyOrderSummaryResponse{
			QtyToday:   src.OrderSummary.QtyToday,
			QtyTotalPO: src.OrderSummary.QtyTotalPO,
			TrendData:  trendData,
		},
		FinancialSummary: DailyFinancialSummaryResponse{
			TotalRevenue:          src.FinancialSummary.TotalRevenue,
			TotalPaid:             src.FinancialSummary.TotalPaid,
			ActivePOOutstanding:   src.FinancialSummary.ActivePOOutstanding,
			PreviousPOOutstanding: src.FinancialSummary.PreviousPOOutstanding,
			TotalOutstanding:      src.FinancialSummary.TotalOutstanding,
		},
		SalesDetails:   salesDetails,
		POSalesDetails: poSalesDetails,
	}
}
