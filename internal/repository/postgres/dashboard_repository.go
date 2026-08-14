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
		Where("payments.status = ?", domain.PaymentVerificationVerified) // 🚨 FIX FINANCIAL BUG: Hanya uang terverifikasi!

	// 🚨 FIX SCOPING: Tambahkan filter SalesID jika diisi (untuk role Sales)
	if filter.SalesID != "" {
		orderQuery = orderQuery.Where("sales_id = ?", filter.SalesID)
		paymentQuery = paymentQuery.Where("orders.sales_id = ?", filter.SalesID)
	}

	// Filter Tanggal
	if filter.StartDate != "" && filter.EndDate != "" {
		if _, errStart := time.Parse("2006-01-02", filter.StartDate); errStart == nil {
			if _, errEnd := time.Parse("2006-01-02", filter.EndDate); errEnd == nil {
				orderQuery = orderQuery.Where("DATE(orders.created_at) BETWEEN ? AND ?", filter.StartDate, filter.EndDate)
				paymentQuery = paymentQuery.Where("DATE(payments.payment_date) BETWEEN ? AND ?", filter.StartDate, filter.EndDate)
			}
		}
	}

	// 2. Eksekusi Kalkulasi Order
	type OrderStats struct {
		TotalRevenue    float64
		ActiveOrders    int64
		CompletedOrders int64
		CanceledOrders  int64
	}
	var stats OrderStats

	err := orderQuery.Select(`
		COALESCE(SUM(CASE WHEN order_status != 'canceled' THEN total_amount ELSE 0 END), 0) as total_revenue,
		COALESCE(SUM(CASE WHEN order_status IN ('pending', 'production', 'ready') THEN 1 ELSE 0 END), 0) as active_orders,
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

	// 3. Eksekusi Kalkulasi Total Uang Masuk
	err = paymentQuery.Select("COALESCE(SUM(payments.amount), 0)").Scan(&summary.TotalPaymentReceived).Error
	if err != nil {
		return nil, TranslateError(err)
	}

	// 4. Eksekusi Kalkulasi Piutang All-Time (Perhatikan scoping SalesID)
	var allTimeRevenue float64
	var allTimePayment float64

	allTimeOrderQ := r.db.WithContext(ctx).Model(&OrderModel{}).Where("order_status != 'canceled'")
	allTimePaymentQ := r.db.WithContext(ctx).Model(&PaymentModel{}).
		Joins("LEFT JOIN orders ON orders.id = payments.order_id").
		Where("payments.status = ?", domain.PaymentVerificationVerified) // 🚨 FIX FINANCIAL BUG

	if filter.SalesID != "" {
		allTimeOrderQ = allTimeOrderQ.Where("sales_id = ?", filter.SalesID)
		allTimePaymentQ = allTimePaymentQ.Where("orders.sales_id = ?", filter.SalesID)
	}

	allTimeOrderQ.Select("COALESCE(SUM(total_amount), 0)").Scan(&allTimeRevenue)
	allTimePaymentQ.Select("COALESCE(SUM(payments.amount), 0)").Scan(&allTimePayment)

	summary.TotalReceivable = allTimeRevenue - allTimePayment
	if summary.TotalReceivable < 0 {
		summary.TotalReceivable = 0
	}

	return summary, nil
}

func (r *dashboardRepository) GetSalesReport(ctx context.Context, filter domain.DashboardFilter) ([]domain.SalesReportItem, error) {
	var report []domain.SalesReportItem

	query := r.db.WithContext(ctx).Model(&OrderModel{})

	// 🚨 FIX SCOPING SALES
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
		COALESCE(SUM(CASE WHEN order_status != 'canceled' THEN total_amount ELSE 0 END), 0) as total_revenue,
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
		Where("o.order_status IN (?, ?, ?)", domain.OrderStatusPending, domain.OrderStatusProduction, domain.OrderStatusReady).
		Where("o.payment_status IN (?, ?)", domain.PaymentStatusUnpaid, domain.PaymentStatusPartial)

	// 🚨 FIX SCOPING: Tambahkan klausa ini jika dipanggil oleh Sales
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
