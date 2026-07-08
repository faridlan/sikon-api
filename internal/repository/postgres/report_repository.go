package postgres

import (
	"context"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type reportRepository struct {
	db *gorm.DB
}

func NewReportRepository(db *gorm.DB) domain.ReportRepository {
	return &reportRepository{db: db}
}

func (r *reportRepository) GetDailyReport(ctx context.Context, date string) (*domain.ReportResponse, error) {
	selectedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		selectedDate = time.Now()
	}

	startOfDay := time.Date(selectedDate.Year(), selectedDate.Month(), selectedDate.Day(), 0, 0, 0, 0, selectedDate.Location())
	endOfDay := startOfDay.Add(24*time.Hour - time.Second)

	response := &domain.ReportResponse{}

	var snapshot struct {
		TotalRevenueToday float64
		TotalQtyToday     int64
	}

	err = r.db.WithContext(ctx).Table("orders o").
		Select(`
			COALESCE(SUM(o.total_amount), 0) as total_revenue_today,
			COALESCE(SUM(oi.qty), 0) as total_qty_today
		`).
		Joins("LEFT JOIN order_items oi ON oi.order_id = o.id AND oi.deleted_at IS NULL").
		Where("o.deleted_at IS NULL").
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Where("o.updated_at BETWEEN ? AND ?", startOfDay, endOfDay).
		Scan(&snapshot).Error
	if err != nil {
		return nil, TranslateError(err)
	}

	response.DailySnapshot.TotalRevenueToday = snapshot.TotalRevenueToday
	response.DailySnapshot.TotalQtyToday = snapshot.TotalQtyToday

	var salesPerformance []domain.SalesPerformanceToday
	err = r.db.WithContext(ctx).Table("orders o").
		Select(`
			u.name as sales_name,
			c.name as product_category,
			COALESCE(SUM(oi.qty), 0) as total_qty
		`).
		Joins("LEFT JOIN order_items oi ON oi.order_id = o.id AND oi.deleted_at IS NULL").
		Joins("LEFT JOIN products p ON p.id = oi.product_id AND p.deleted_at IS NULL").
		Joins("LEFT JOIN categories c ON c.id = p.category_id AND c.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = o.sales_id AND u.deleted_at IS NULL").
		Where("o.deleted_at IS NULL").
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Where("o.updated_at BETWEEN ? AND ?", startOfDay, endOfDay).
		Group("u.name, c.name").
		Order("total_qty DESC").
		Scan(&salesPerformance).Error
	if err != nil {
		return nil, TranslateError(err)
	}
	response.DailySnapshot.SalesPerformanceToday = salesPerformance

	var activePOs []domain.ActivePO
	err = r.db.WithContext(ctx).Table("batch_pos bp").
		Select(`
			bp.id as batch_po_id,
			bp.name as batch_po_name,
			COALESCE(SUM(o.total_amount), 0) as total_revenue_entered,
			(bp.quota - COALESCE(COUNT(o.id), 0)) as remaining_quota
		`).
		Joins("LEFT JOIN orders o ON o.batch_po_id = bp.id AND o.deleted_at IS NULL").
		Where("bp.deleted_at IS NULL").
		Where("bp.status = ?", domain.BatchPOStatusActive).
		Group("bp.id, bp.name, bp.quota").
		Scan(&activePOs).Error
	if err != nil {
		return nil, TranslateError(err)
	}
	response.ActivePOs = activePOs

	var outstanding struct {
		TotalOutstanding float64
		ActivePO         float64
	}
	err = r.db.WithContext(ctx).Table("orders o").
		Select(`
			COALESCE(SUM(CASE WHEN o.payment_status IN (?, ?) THEN o.total_amount - COALESCE(p.total_paid, 0) ELSE 0 END), 0) as total_outstanding,
			COALESCE(SUM(CASE WHEN o.payment_status IN (?, ?) AND bp.status = ? THEN o.total_amount - COALESCE(p.total_paid, 0) ELSE 0 END), 0) as active_po
		`, domain.PaymentStatusUnpaid, domain.PaymentStatusPartial, domain.PaymentStatusUnpaid, domain.PaymentStatusPartial, domain.BatchPOStatusActive).
		Joins("LEFT JOIN (SELECT order_id, COALESCE(SUM(amount), 0) as total_paid FROM payments WHERE deleted_at IS NULL GROUP BY order_id) p ON p.order_id = o.id").
		Joins("LEFT JOIN batch_pos bp ON bp.id = o.batch_po_id AND bp.deleted_at IS NULL").
		Where("o.deleted_at IS NULL").
		Scan(&outstanding).Error
	if err != nil {
		return nil, TranslateError(err)
	}
	response.TotalOutstandingReceivables = outstanding.TotalOutstanding
	response.ActivePOReceivables = outstanding.ActivePO

	var pastDue []domain.PastDueReceivable
	err = r.db.WithContext(ctx).Table("orders o").
		Select(`
			u.name as sales_name,
			c.name as product_category,
			cu.name as customer_name,
			(o.total_amount - COALESCE(p.total_paid, 0)) as unpaid_balance
		`).
		Joins("LEFT JOIN (SELECT order_id, COALESCE(SUM(amount), 0) as total_paid FROM payments WHERE deleted_at IS NULL GROUP BY order_id) p ON p.order_id = o.id").
		Joins("LEFT JOIN customers cu ON cu.id = o.customer_id AND cu.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = o.sales_id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN order_items oi ON oi.order_id = o.id AND oi.deleted_at IS NULL").
		Joins("LEFT JOIN products pr ON pr.id = oi.product_id AND pr.deleted_at IS NULL").
		Joins("LEFT JOIN categories c ON c.id = pr.category_id AND c.deleted_at IS NULL").
		Where("o.deleted_at IS NULL").
		Where("o.order_status = ?", domain.OrderStatusCompleted).
		Where("o.payment_status IN (?, ?)", domain.PaymentStatusUnpaid, domain.PaymentStatusPartial).
		Where("(o.total_amount - COALESCE(p.total_paid, 0)) > 0").
		Group("u.name, c.name, cu.name, o.total_amount, p.total_paid").
		Order("unpaid_balance DESC").
		Scan(&pastDue).Error
	if err != nil {
		return nil, TranslateError(err)
	}
	response.PastDueReceivables = pastDue

	if response.ActivePOs == nil {
		response.ActivePOs = []domain.ActivePO{}
	}
	if response.DailySnapshot.SalesPerformanceToday == nil {
		response.DailySnapshot.SalesPerformanceToday = []domain.SalesPerformanceToday{}
	}
	if response.PastDueReceivables == nil {
		response.PastDueReceivables = []domain.PastDueReceivable{}
	}

	return response, nil
}
