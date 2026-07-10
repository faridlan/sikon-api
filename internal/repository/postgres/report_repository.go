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

func calculateRemainingQuota(quota int, usedQty int64) int64 {
	if quota <= 0 {
		return 0
	}

	remaining := int64(quota) - usedQty
	if remaining < 0 {
		return 0
	}

	return remaining
}

func (r *reportRepository) GetDailyRevenueAndQty(ctx context.Context, startOfDay, endOfDay time.Time) (float64, int64, error) {
	var snapshot struct {
		TotalRevenueToday float64
		TotalQtyToday     int64
	}

	// total_revenue_today  : semua order pada tanggal tersebut (tanpa filter PO).
	// total_qty_today      : hanya order yang terikat pada PO yang ACTIVE.
	err := r.db.WithContext(ctx).Table("orders o").
		Select(`
			COALESCE(SUM(o.total_amount), 0) as total_revenue_today,
			COALESCE(SUM(CASE WHEN bp.status = ? THEN oi.qty ELSE 0 END), 0) as total_qty_today
		`, domain.BatchPOStatusActive).
		Joins("LEFT JOIN batch_pos bp ON bp.id = o.batch_po_id AND bp.deleted_at IS NULL").
		Joins("LEFT JOIN order_items oi ON oi.order_id = o.id AND oi.deleted_at IS NULL").
		Where("o.deleted_at IS NULL").
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Where("o.updated_at BETWEEN ? AND ?", startOfDay, endOfDay).
		Scan(&snapshot).Error
	if err != nil {
		return 0, 0, TranslateError(err)
	}

	return snapshot.TotalRevenueToday, snapshot.TotalQtyToday, nil
}

func (r *reportRepository) GetSalesPerformanceByActivePO(ctx context.Context) ([]domain.SalesPerformanceToday, error) {
	var salesPerformance []domain.SalesPerformanceToday
	err := r.db.WithContext(ctx).Table("orders o").
		Select(`
			u.name as sales_name,
			c.name as product_category,
			COALESCE(SUM(oi.qty), 0) as total_qty
		`).
		Joins("JOIN batch_pos bp ON bp.id = o.batch_po_id AND bp.deleted_at IS NULL AND bp.status = ?", domain.BatchPOStatusActive).
		Joins("LEFT JOIN order_items oi ON oi.order_id = o.id AND oi.deleted_at IS NULL").
		Joins("LEFT JOIN products p ON p.id = oi.product_id AND p.deleted_at IS NULL").
		Joins("LEFT JOIN categories c ON c.id = p.category_id AND c.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = o.sales_id AND u.deleted_at IS NULL").
		Where("o.deleted_at IS NULL").
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("u.name, c.name").
		Order("total_qty DESC").
		Scan(&salesPerformance).Error
	if err != nil {
		return nil, TranslateError(err)
	}

	if salesPerformance == nil {
		salesPerformance = []domain.SalesPerformanceToday{}
	}

	return salesPerformance, nil
}

func (r *reportRepository) GetActivePOStats(ctx context.Context) ([]domain.ActivePO, error) {
	var activePORows []struct {
		BatchPOID           string
		BatchPOName         string
		Quota               int
		TotalRevenueEntered float64
		QtyReceived         int64
	}

	err := r.db.WithContext(ctx).Table("batch_pos bp").
		Select(`
			bp.id as batch_po_id,
			bp.name as batch_po_name,
			bp.quota as quota,
			COALESCE(orders_revenue.total_revenue_entered, 0) as total_revenue_entered,
			COALESCE(order_item_qty.qty_received, 0) as qty_received
		`).
		Joins(`
			LEFT JOIN (
				SELECT batch_po_id, COALESCE(SUM(total_amount), 0) as total_revenue_entered
				FROM orders
				WHERE deleted_at IS NULL AND order_status IN (?, ?)
				GROUP BY batch_po_id
			) orders_revenue ON orders_revenue.batch_po_id = bp.id
		`, domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Joins(`
			LEFT JOIN (
				SELECT o.batch_po_id, COALESCE(SUM(oi.qty), 0) as qty_received
				FROM orders o
				LEFT JOIN order_items oi ON oi.order_id = o.id AND oi.deleted_at IS NULL
				WHERE o.deleted_at IS NULL AND o.order_status IN (?, ?)
				GROUP BY o.batch_po_id
			) order_item_qty ON order_item_qty.batch_po_id = bp.id
		`, domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Where("bp.deleted_at IS NULL").
		Where("bp.status = ?", domain.BatchPOStatusActive).
		Group("bp.id, bp.name, bp.quota, orders_revenue.total_revenue_entered, order_item_qty.qty_received").
		Scan(&activePORows).Error
	if err != nil {
		return nil, TranslateError(err)
	}

	activePOs := make([]domain.ActivePO, 0, len(activePORows))
	for _, row := range activePORows {
		activePOs = append(activePOs, domain.ActivePO{
			BatchPOID:           row.BatchPOID,
			BatchPOName:         row.BatchPOName,
			TotalRevenueEntered: row.TotalRevenueEntered,
			TotalQtyReceived:    row.QtyReceived,
			RemainingQuota:      calculateRemainingQuota(row.Quota, row.QtyReceived),
		})
	}

	return activePOs, nil
}

func (r *reportRepository) GetReceivablesStats(ctx context.Context) (float64, float64, error) {
	var outstanding struct {
		TotalOutstanding float64
		ActivePO         float64
	}

	err := r.db.WithContext(ctx).Table("orders o").
		Select(`
			COALESCE(SUM(CASE WHEN o.payment_status IN (?, ?) THEN (o.total_amount - COALESCE(p.total_paid, 0)) ELSE 0 END), 0) as total_outstanding,
			COALESCE(SUM(CASE WHEN o.payment_status IN (?, ?) AND bp.status = ? THEN (o.total_amount - COALESCE(p.total_paid, 0)) ELSE 0 END), 0) as active_po
		`, domain.PaymentStatusUnpaid, domain.PaymentStatusPartial, domain.PaymentStatusUnpaid, domain.PaymentStatusPartial, domain.BatchPOStatusActive).
		Joins("LEFT JOIN batch_pos bp ON bp.id = o.batch_po_id AND bp.deleted_at IS NULL").
		Joins("LEFT JOIN (SELECT order_id, COALESCE(SUM(amount), 0) as total_paid FROM payments WHERE deleted_at IS NULL GROUP BY order_id) p ON p.order_id = o.id").
		Where("o.deleted_at IS NULL").
		Where("o.order_status IN (?, ?, ?)", domain.OrderStatusProduction, domain.OrderStatusReady, domain.OrderStatusCompleted).
		Scan(&outstanding).Error
	if err != nil {
		return 0, 0, TranslateError(err)
	}

	return outstanding.TotalOutstanding, outstanding.ActivePO, nil
}

func (r *reportRepository) GetPastDueReceivables(ctx context.Context) ([]domain.PastDueReceivable, error) {
	var pastDue []domain.PastDueReceivable
	err := r.db.WithContext(ctx).Table("orders o").
		Select(`
			u.name as sales_name,
			c.name as product_category,
			cu.name as customer_name,
			(o.total_amount - COALESCE(p.total_paid, 0)) as unpaid_balance
		`).
		Joins("JOIN batch_pos bp ON bp.id = o.batch_po_id AND bp.deleted_at IS NULL AND bp.status != ?", domain.BatchPOStatusActive).
		Joins("LEFT JOIN (SELECT order_id, COALESCE(SUM(amount), 0) as total_paid FROM payments WHERE deleted_at IS NULL GROUP BY order_id) p ON p.order_id = o.id").
		Joins("LEFT JOIN customers cu ON cu.id = o.customer_id AND cu.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = o.sales_id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN order_items oi ON oi.order_id = o.id AND oi.deleted_at IS NULL").
		Joins("LEFT JOIN products pr ON pr.id = oi.product_id AND pr.deleted_at IS NULL").
		Joins("LEFT JOIN categories c ON c.id = pr.category_id AND c.deleted_at IS NULL").
		Where("o.deleted_at IS NULL").
		Where("o.order_status IN (?, ?, ?)", domain.OrderStatusProduction, domain.OrderStatusReady, domain.OrderStatusCompleted).
		Where("o.payment_status IN (?, ?)", domain.PaymentStatusUnpaid, domain.PaymentStatusPartial).
		Where("(o.total_amount - COALESCE(p.total_paid, 0)) > 0").
		Group("u.name, c.name, cu.name, o.total_amount, p.total_paid").
		Order("unpaid_balance DESC").
		Scan(&pastDue).Error
	if err != nil {
		return nil, TranslateError(err)
	}

	if pastDue == nil {
		pastDue = []domain.PastDueReceivable{}
	}

	return pastDue, nil
}
