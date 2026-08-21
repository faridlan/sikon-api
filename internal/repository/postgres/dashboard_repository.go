package postgres

import (
	"context"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type dashboardRepository struct {
	db *gorm.DB
}

func NewDashboardRepository(db *gorm.DB) domain.DashboardRepository {
	return &dashboardRepository{db: db}
}

func (r *dashboardRepository) GetSummary(ctx context.Context, filter domain.DashboardFilter) (*domain.DashboardSummary, error) {
	summary := &domain.DashboardSummary{}

	// 1. Setup Query Base (Orders & Payments)
	orderQuery := r.db.WithContext(ctx).Model(&OrderModel{})
	paymentQuery := r.db.WithContext(ctx).Model(&PaymentModel{}).
		Joins("LEFT JOIN orders ON orders.id = payments.order_id").
		Where("payments.status = ?", domain.PaymentVerificationVerified)

	if filter.SalesID != "" {
		orderQuery = orderQuery.Where("sales_id = ?", filter.SalesID)
		paymentQuery = paymentQuery.Where("orders.sales_id = ?", filter.SalesID)
	}

	// Filter Tanggal untuk Revenue & Payment Received dalam periode tersebut
	if filter.StartDate != "" && filter.EndDate != "" {
		if _, errStart := time.Parse("2006-01-02", filter.StartDate); errStart == nil {
			if _, errEnd := time.Parse("2006-01-02", filter.EndDate); errEnd == nil {
				orderQuery = orderQuery.Where("DATE(orders.created_at) BETWEEN ? AND ?", filter.StartDate, filter.EndDate)
				paymentQuery = paymentQuery.Where("DATE(payments.payment_date) BETWEEN ? AND ?", filter.StartDate, filter.EndDate)
			}
		}
	}

	// 2. Eksekusi Kalkulasi Order Stats
	type OrderStats struct {
		TotalRevenue    float64
		ActiveOrders    int64
		CompletedOrders int64
		CanceledOrders  int64
	}
	var stats OrderStats

	err := orderQuery.Select(`
		COALESCE(SUM(CASE WHEN order_status IN ('production', 'ready', 'completed') THEN total_amount ELSE 0 END), 0) as total_revenue,
		COALESCE(SUM(CASE WHEN order_status IN ('production', 'ready') THEN 1 ELSE 0 END), 0) as active_orders,
		COALESCE(SUM(CASE WHEN order_status = 'completed' THEN 1 ELSE 0 END), 0) as completed_orders,
		COALESCE(SUM(CASE WHEN order_status = 'canceled' THEN 1 ELSE 0 END), 0) as canceled_orders
	`).Scan(&stats).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	summary.TotalRevenue = stats.TotalRevenue
	summary.TotalActiveOrders = stats.ActiveOrders
	summary.TotalCompletedOrders = stats.CompletedOrders
	summary.TotalCanceledOrders = stats.CanceledOrders

	// 3. Eksekusi Kalkulasi Total Uang Masuk (Verified Payments)
	err = paymentQuery.Select("COALESCE(SUM(payments.amount), 0)").Scan(&summary.TotalPaymentReceived).Error
	if err != nil {
		return nil, TranslateError(err)
	}

	// 4. 🚨 PERBAIKAN KALKULASI PIUTANG (RECEIVABLE) SINKRON DENGAN LAPORAN PIUTANG
	// Menggunakan query sub-kalkulasi presisi dari order aktif ber-status Unpaid/Partial
	receivableQuery := r.db.WithContext(ctx).Table("orders o").
		Select(`
			COALESCE(SUM(o.total_amount - COALESCE(p.total_paid, 0)), 0) as total_receivable
		`).
		Joins(`LEFT JOIN (
			SELECT order_id, SUM(amount) as total_paid 
			FROM payments 
			WHERE status = 'verified' AND deleted_at IS NULL 
			GROUP BY order_id
		) p ON p.order_id = o.id`).
		Where("o.deleted_at IS NULL").
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusReady).
		Where("o.payment_status IN (?, ?)", domain.PaymentStatusUnpaid, domain.PaymentStatusPartial)

	if filter.SalesID != "" {
		receivableQuery = receivableQuery.Where("o.sales_id = ?", filter.SalesID)
	}

	var totalReceivable float64
	err = receivableQuery.Scan(&totalReceivable).Error
	if err != nil {
		return nil, TranslateError(err)
	}

	summary.TotalReceivable = totalReceivable
	if summary.TotalReceivable < 0 {
		summary.TotalReceivable = 0
	}

	return summary, nil
}

func (r *dashboardRepository) GetSalesReport(ctx context.Context, filter domain.DashboardFilter) ([]domain.SalesReportItem, error) {
	var report []domain.SalesReportItem

	query := r.db.WithContext(ctx).Model(&OrderModel{})

	if filter.SalesID != "" {
		query = query.Where("sales_id = ?", filter.SalesID)
	}

	if filter.StartDate != "" && filter.EndDate != "" {
		start, errStart := time.Parse("2006-01-02", filter.StartDate)
		end, errEnd := time.Parse("2006-01-02", filter.EndDate)

		if errStart == nil && errEnd == nil {
			end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			query = query.Where("created_at BETWEEN ? AND ?", start, end)
		}
	}

	err := query.Select(`
		TO_CHAR(created_at, 'YYYY-MM-DD') as date,
		COALESCE(SUM(CASE WHEN order_status IN ('production', 'ready', 'completed') THEN total_amount ELSE 0 END), 0) as total_revenue,
		COUNT(id) as total_orders,
		COALESCE(SUM(CASE WHEN order_status = 'completed' THEN 1 ELSE 0 END), 0) as completed_orders,
		COALESCE(SUM(CASE WHEN order_status = 'canceled' THEN 1 ELSE 0 END), 0) as canceled_orders
	`).
		Group("TO_CHAR(created_at, 'YYYY-MM-DD')").
		Order("date ASC").
		Scan(&report).Error

	if err != nil {
		return nil, err
	}

	if report == nil {
		report = []domain.SalesReportItem{}
	}

	return report, nil
}

func (r *dashboardRepository) GetReceivablesReport(ctx context.Context, salesID string) ([]domain.ReceivableReportItem, error) {
	var report []domain.ReceivableReportItem

	query := r.db.WithContext(ctx).Table("orders o").
		Select(`
			o.id as order_id,
			o.order_number,
			TO_CHAR(o.created_at, 'YYYY-MM-DD') as order_date,
			c.name as customer_name,
			u.name as sales_name,
			o.order_status,
			o.total_amount,
			COALESCE(SUM(CASE WHEN p.status = 'verified' THEN p.amount ELSE 0 END), 0) as total_paid,
			(o.total_amount - COALESCE(SUM(CASE WHEN p.status = 'verified' THEN p.amount ELSE 0 END), 0)) as remaining_bill
		`).
		Joins("LEFT JOIN customers c ON c.id = o.customer_id AND c.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = o.sales_id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN payments p ON p.order_id = o.id AND p.deleted_at IS NULL").
		Where("o.deleted_at IS NULL").
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusReady).
		Where("o.payment_status IN (?, ?)", domain.PaymentStatusUnpaid, domain.PaymentStatusPartial)

	if salesID != "" {
		query = query.Where("o.sales_id = ?", salesID)
	}

	err := query.
		Group("o.id, o.order_number, o.created_at, c.name, u.name, o.order_status, o.total_amount").
		Order("o.created_at ASC").
		Scan(&report).Error

	if err != nil {
		return nil, err
	}

	if report == nil {
		report = []domain.ReceivableReportItem{}
	}

	return report, nil
}

func (r *dashboardRepository) GetOverview(ctx context.Context, salesID string) (*domain.DashboardOverview, error) {
	overview := &domain.DashboardOverview{
		RecentOrders:   []domain.RecentOrderOverview{},
		RecentPayments: []domain.RecentPaymentOverview{},
		ChartTrends:    []domain.SalesReportItem{},
	}

	// 1. Ambil Summary
	summary, err := r.GetSummary(ctx, domain.DashboardFilter{SalesID: salesID})
	if err != nil {
		return nil, err
	}
	overview.Summary = *summary

	// 2. Ambil Chart Trends (14 Hari Terakhir)
	now := time.Now()
	startDate14Days := now.AddDate(0, 0, -14).Format("2006-01-02")
	endDateToday := now.Format("2006-01-02")

	chartTrends, err := r.GetSalesReport(ctx, domain.DashboardFilter{
		StartDate: startDate14Days,
		EndDate:   endDateToday,
		SalesID:   salesID,
	})
	if err == nil {
		overview.ChartTrends = chartTrends
	}

	// 3. Ambil Active Batch PO + Hitung Kuota Terpakai (Total Qty Ordered) & Sisa Kuota
	var activePOModel BatchPOModel
	errActivePO := r.db.WithContext(ctx).
		Where("status = ? AND ? BETWEEN start_date AND end_date AND deleted_at IS NULL", domain.BatchPOStatusActive, now).
		First(&activePOModel).Error

	if errActivePO == nil {
		var totalQtyOrdered int64
		r.db.WithContext(ctx).Table("order_items oi").
			Joins("JOIN orders o ON o.id = oi.order_id").
			Where("o.batch_po_id = ? AND o.order_status IN (?, ?, ?)", activePOModel.ID, domain.OrderStatusProduction, domain.OrderStatusReady, domain.OrderStatusCompleted).
			Where("o.deleted_at IS NULL AND oi.deleted_at IS NULL").
			Select("COALESCE(SUM(oi.qty), 0)").
			Scan(&totalQtyOrdered)

		remainingQuota := int64(activePOModel.Quota) - totalQtyOrdered
		if remainingQuota < 0 {
			remainingQuota = 0
		}

		overview.ActiveBatchPO = &domain.ActiveBatchPOOverview{
			ID:              activePOModel.ID,
			Name:            activePOModel.Name,
			Status:          activePOModel.Status,
			Quota:           activePOModel.Quota,
			TotalQtyOrdered: totalQtyOrdered,
			RemainingQuota:  remainingQuota,
		}
	}

	// 4. Hitung Badges "Action Required"
	orderQuery := r.db.WithContext(ctx).Model(&OrderModel{})
	paymentQuery := r.db.WithContext(ctx).Model(&PaymentModel{})

	if salesID != "" {
		orderQuery = orderQuery.Where("sales_id = ?", salesID)
		paymentQuery = paymentQuery.Joins("LEFT JOIN orders ON orders.id = payments.order_id").Where("orders.sales_id = ?", salesID)
	}

	paymentQuery.Where("payments.status = ?", domain.PaymentVerificationPending).Count(&overview.ActionRequired.UnverifiedPaymentsCount)

	orderQuery.Session(&gorm.Session{}).Where("order_status = ?", domain.OrderStatusReady).Count(&overview.ActionRequired.ReadyOrdersCount)
	orderQuery.Session(&gorm.Session{}).Where("order_status = ?", domain.OrderStatusPending).Count(&overview.ActionRequired.PendingOrdersCount)

	// 5. Ambil Recent Orders (Limit 6 Data)
	// 🚨 REVISI LOGIKA: HANYA tampilkan order yang sudah masuk produksi, siap kirim, atau selesai
	var recentOrders []struct {
		ID           string
		OrderNumber  string
		CustomerName string
		OrderStatus  string
		TotalAmount  float64
	}

	recentOrderQ := r.db.WithContext(ctx).Table("orders o").
		Select("o.id, o.order_number, c.name as customer_name, o.order_status, o.total_amount").
		Joins("LEFT JOIN customers c ON c.id = o.customer_id AND c.deleted_at IS NULL").
		Where("o.deleted_at IS NULL").
		Where("o.order_status IN (?, ?, ?)", domain.OrderStatusProduction, domain.OrderStatusReady, domain.OrderStatusCompleted) // 👈 HANYA ORDER VALID

	if salesID != "" {
		recentOrderQ = recentOrderQ.Where("o.sales_id = ?", salesID)
	}

	err = recentOrderQ.Order("COALESCE(o.approved_at, o.created_at) DESC").Limit(6).Scan(&recentOrders).Error
	if err == nil {
		for _, ro := range recentOrders {
			overview.RecentOrders = append(overview.RecentOrders, domain.RecentOrderOverview{
				ID:           ro.ID,
				OrderNumber:  ro.OrderNumber,
				CustomerName: ro.CustomerName,
				OrderStatus:  ro.OrderStatus,
				TotalAmount:  ro.TotalAmount,
			})
		}
	}

	// 6. Ambil Recent Payments (Limit 6 Data)
	var recentPayments []struct {
		ID          string
		PaymentDate time.Time
		PaymentType string
		Status      string
		Amount      float64
	}

	recentPayQ := r.db.WithContext(ctx).Table("payments p").
		Select("p.id, p.payment_date, p.payment_type, p.status, p.amount").
		Where("p.deleted_at IS NULL")

	if salesID != "" {
		recentPayQ = recentPayQ.Joins("LEFT JOIN orders o ON o.id = p.order_id").Where("o.sales_id = ?", salesID)
	}

	err = recentPayQ.Order("p.payment_date DESC").Limit(6).Scan(&recentPayments).Error
	if err == nil {
		for _, rp := range recentPayments {
			overview.RecentPayments = append(overview.RecentPayments, domain.RecentPaymentOverview{
				ID:          rp.ID,
				PaymentDate: rp.PaymentDate.Format(time.RFC3339),
				PaymentType: rp.PaymentType,
				Status:      rp.Status,
				Amount:      rp.Amount,
			})
		}
	}

	return overview, nil
}
