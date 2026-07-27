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

func (r *reportRepository) GetDailyReportData(ctx context.Context, targetDate time.Time) (*domain.DailyReport, error) {
	report := &domain.DailyReport{
		ReportDate: targetDate.Format("2006-01-02"),
	}

	// 1. Cari PO Aktif
	var activePO struct {
		ID        string
		Name      string
		Quota     int
		StartDate time.Time
	}
	err := r.db.WithContext(ctx).Table("batch_pos").
		Select("id, name, quota, start_date").
		Where("? BETWEEN start_date AND end_date", targetDate).
		// Where("status = ?", domain.BatchPOStatusActive). // <- Sengaja di-comment untuk support time-travel
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		First(&activePO).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return report, nil
		}
		return nil, TranslateError(err)
	}

	report.POInfo.POID = activePO.ID
	report.POInfo.POName = activePO.Name
	report.POInfo.Quota = activePO.Quota

	// Set waktu awal dan akhir hari untuk kalkulasi QTY harian
	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	endOfDay := startOfDay.Add(24*time.Hour - time.Nanosecond)

	// 2. Kalkulasi Qty (Query terpisah agar tidak duplikasi dengan payment)
	var qtyStats struct {
		QtyToday   int64
		QtyTotalPO int64
	}
	r.db.WithContext(ctx).Table("orders o").
		Select(`
			COALESCE(SUM(CASE WHEN o.created_at BETWEEN ? AND ? THEN oi.qty ELSE 0 END), 0) as qty_today,
			COALESCE(SUM(oi.qty), 0) as qty_total_po
		`, startOfDay, endOfDay).
		Joins("JOIN order_items oi ON oi.order_id = o.id AND oi.deleted_at IS NULL").
		Where("o.batch_po_id = ?", activePO.ID).
		Where("o.deleted_at IS NULL").
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Scan(&qtyStats)

	report.OrderSummary.QtyToday = qtyStats.QtyToday
	report.OrderSummary.QtyTotalPO = qtyStats.QtyTotalPO
	report.POInfo.RemainingQuota = int64(activePO.Quota) - qtyStats.QtyTotalPO

	// =========================================================================
	// 2B. KALKULASI GRAFIK TREND HARIAN (TAMBAHKAN DI SINI)
	// =========================================================================
	var trendRows []struct {
		Date time.Time
		Qty  int64
	}
	r.db.WithContext(ctx).Table("orders o").
		Select("DATE(o.created_at) as date, COALESCE(SUM(oi.qty), 0) as qty").
		Joins("JOIN order_items oi ON oi.order_id = o.id AND oi.deleted_at IS NULL").
		Where("o.batch_po_id = ?", activePO.ID).
		Where("o.deleted_at IS NULL").
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Where("o.created_at <= ?", endOfDay). // Jangan hitung orderan "besok" jika ada time-travel
		Group("DATE(o.created_at)").
		Order("date ASC").
		Scan(&trendRows)

	// Buat Map untuk mempercepat pencarian data per tanggal
	qtyMap := make(map[string]int64)
	for _, row := range trendRows {
		qtyMap[row.Date.Format("2006-01-02")] = row.Qty
	}

	// Looping dari hari pertama PO sampai ke hari targetDate
	for d := activePO.StartDate; !d.After(targetDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		report.OrderSummary.TrendData = append(report.OrderSummary.TrendData, domain.DailyTrend{
			Date: dateStr,
			Qty:  qtyMap[dateStr], // Jika tidak ada order di tanggal tersebut, otomatis jadi 0
		})
	}
	// =========================================================================

	// 3. Kalkulasi Finansial PO Aktif (Bug 2 Fixed: Join dengan Payments)
	var finActive struct {
		TotalRevenue float64
		TotalPaid    float64
	}
	r.db.WithContext(ctx).Table("orders o").
		Select(`
			COALESCE(SUM(o.total_amount), 0) as total_revenue,
			COALESCE(SUM(p.total_paid), 0) as total_paid
		`).
		Joins("LEFT JOIN (SELECT order_id, SUM(amount) as total_paid FROM payments WHERE deleted_at IS NULL GROUP BY order_id) p ON p.order_id = o.id").
		Where("o.batch_po_id = ?", activePO.ID).
		Where("o.deleted_at IS NULL").
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Scan(&finActive)

	report.FinancialSummary.TotalRevenue = finActive.TotalRevenue
	report.FinancialSummary.TotalPaid = finActive.TotalPaid
	report.FinancialSummary.ActivePOOutstanding = finActive.TotalRevenue - finActive.TotalPaid

	// 4. Kalkulasi Sisa Piutang (Outstanding) dari PO Sebelumnya
	var prevOutstanding float64
	r.db.WithContext(ctx).Table("orders o").
		Select("COALESCE(SUM(o.total_amount - COALESCE(p.total_paid, 0)), 0)").
		Joins("JOIN batch_pos bp ON bp.id = o.batch_po_id").
		Joins("LEFT JOIN (SELECT order_id, SUM(amount) as total_paid FROM payments WHERE deleted_at IS NULL GROUP BY order_id) p ON p.order_id = o.id").
		Where("bp.end_date < ?", activePO.StartDate).
		Where("o.deleted_at IS NULL AND bp.deleted_at IS NULL").
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Where("o.payment_status IN (?, ?)", domain.PaymentStatusUnpaid, domain.PaymentStatusPartial). // Opsional untuk optimasi query
		Scan(&prevOutstanding)

	report.FinancialSummary.PreviousPOOutstanding = prevOutstanding
	report.FinancialSummary.TotalOutstanding = report.FinancialSummary.ActivePOOutstanding + prevOutstanding

	// 5. Pivot Sales Detail
	var salesRows []struct {
		SalesName    string
		CategoryName string
		Qty          int64
	}
	r.db.WithContext(ctx).Table("orders o").
		Select("u.name as sales_name, c.name as category_name, COALESCE(SUM(oi.qty), 0) as qty").
		Joins("JOIN users u ON u.id = o.sales_id").
		Joins("JOIN order_items oi ON oi.order_id = o.id AND oi.deleted_at IS NULL").
		Joins("JOIN products p ON p.id = oi.product_id").
		Joins("JOIN categories c ON c.id = p.category_id").
		Where("o.batch_po_id = ?", activePO.ID).
		Where("o.deleted_at IS NULL").
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("u.name, c.name").
		Scan(&salesRows)

	salesMap := make(map[string]*domain.SalesDetail)
	for _, row := range salesRows {
		if _, exists := salesMap[row.SalesName]; !exists {
			salesMap[row.SalesName] = &domain.SalesDetail{
				SalesName:  row.SalesName,
				Categories: make(map[string]int64),
			}
		}
		salesMap[row.SalesName].Categories[row.CategoryName] += row.Qty
		salesMap[row.SalesName].TotalQty += row.Qty
	}

	for _, sd := range salesMap {
		report.SalesDetails = append(report.SalesDetails, *sd)
	}

	return report, nil
}

func (r *reportRepository) GetPOSummaryData(ctx context.Context, poID string) (*domain.POSummaryReport, error) {
	summary := &domain.POSummaryReport{}

	// 1. Ambil Data Dasar PO
	var po domain.BatchPO
	if err := r.db.WithContext(ctx).Table("batch_pos").Where("id = ? AND deleted_at IS NULL", poID).First(&po).Error; err != nil {
		return nil, TranslateError(err)
	}
	summary.POID = po.ID
	summary.POName = po.Name
	summary.StartDate = po.StartDate
	summary.EndDate = po.EndDate
	summary.Status = po.Status
	summary.TotalQuota = po.Quota

	// 2. Kalkulasi Finansial (Revenue, Paid, Outstanding)
	var financial struct {
		TotalRevenue     float64
		TotalPaid        float64
		TotalOutstanding float64
	}
	r.db.WithContext(ctx).Table("orders o").
		Select(`
			COALESCE(SUM(o.total_amount), 0) as total_revenue,
			COALESCE(SUM(p.total_paid), 0) as total_paid,
			COALESCE(SUM(o.total_amount - COALESCE(p.total_paid, 0)), 0) as total_outstanding
		`).
		Joins("LEFT JOIN (SELECT order_id, SUM(amount) as total_paid FROM payments WHERE deleted_at IS NULL GROUP BY order_id) p ON p.order_id = o.id").
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL", poID).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Scan(&financial)

	summary.TotalRevenue = financial.TotalRevenue
	summary.TotalPaid = financial.TotalPaid
	summary.TotalOutstanding = financial.TotalOutstanding

	// 3. Rekap Produk untuk Produksi Konveksi
	r.db.WithContext(ctx).Table("order_items oi").
		Select("c.name as category_name, SUM(oi.qty) as total_qty").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Joins("JOIN products p ON p.id = oi.product_id").
		Joins("JOIN categories c ON c.id = p.category_id").
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL AND oi.deleted_at IS NULL", poID).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("c.name").
		Scan(&summary.ProductSummary)

	// Hitung total keseluruhan QTY order
	for _, p := range summary.ProductSummary {
		summary.TotalQtyOrdered += p.TotalQty
	}

	// 4. Rekap Performa Sales
	r.db.WithContext(ctx).Table("orders o").
		Select("u.name as sales_name, SUM(oi.qty) as total_qty, SUM(DISTINCT o.total_amount) as total_revenue").
		Joins("JOIN users u ON u.id = o.sales_id").
		Joins("LEFT JOIN order_items oi ON oi.order_id = o.id AND oi.deleted_at IS NULL").
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL", poID).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("u.name").
		Scan(&summary.SalesSummary)

	return summary, nil
}

func (r *reportRepository) GetReceivablesDetailData(ctx context.Context) ([]domain.ReceivableDetail, error) {
	var details []domain.ReceivableDetail

	err := r.db.WithContext(ctx).Table("orders o").
		Select(`
			o.id as order_id,
			o.order_number,
			bp.name as po_name,
			bp.status as po_status,
			c.name as customer_name,
			u.name as sales_name,
			o.total_amount,
			COALESCE(p.total_paid, 0) as total_paid,
			(o.total_amount - COALESCE(p.total_paid, 0)) as outstanding_amount
		`).
		Joins("LEFT JOIN batch_pos bp ON bp.id = o.batch_po_id AND bp.deleted_at IS NULL").
		Joins("LEFT JOIN customers c ON c.id = o.customer_id AND c.deleted_at IS NULL").
		Joins("LEFT JOIN users u ON u.id = o.sales_id AND u.deleted_at IS NULL").
		Joins("LEFT JOIN (SELECT order_id, SUM(amount) as total_paid FROM payments WHERE deleted_at IS NULL GROUP BY order_id) p ON p.order_id = o.id").
		Where("o.deleted_at IS NULL").
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Where("o.payment_status IN (?, ?)", domain.PaymentStatusUnpaid, domain.PaymentStatusPartial).
		Where("(o.total_amount - COALESCE(p.total_paid, 0)) > 0"). // Filter utama: hanya yang masih punya piutang
		Order("outstanding_amount DESC").                          // Tampilkan dari piutang terbesar
		Scan(&details).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	return details, nil
}

func (r *reportRepository) GetMonthlyReportData(ctx context.Context, month, year int) (*domain.MonthlyReport, error) {
	// Tentukan rentang waktu: Awal bulan ini sampai awal bulan depan
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, 0)

	var report domain.MonthlyReport
	report.Month = month
	report.Year = year
	report.PeriodName = startDate.Format("January 2006")

	// 1. Ambil Summary Orders (Berdasarkan approved_at)
	var totalOrders int64
	var totalOmset float64
	r.db.WithContext(ctx).Table("orders").
		Where("approved_at >= ? AND approved_at < ? AND deleted_at IS NULL", startDate, endDate).
		Count(&totalOrders)

	r.db.WithContext(ctx).Table("orders").
		Where("approved_at >= ? AND approved_at < ? AND deleted_at IS NULL", startDate, endDate).
		Select("COALESCE(SUM(total_amount), 0)").Scan(&totalOmset)

	// 2. Ambil Summary Total Qty Barang
	var totalQty int64
	r.db.WithContext(ctx).Table("order_items").
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.approved_at >= ? AND orders.approved_at < ? AND orders.deleted_at IS NULL AND order_items.deleted_at IS NULL", startDate, endDate).
		Select("COALESCE(SUM(order_items.qty), 0)").Scan(&totalQty)

	// 3. Ambil Summary Cash In (Berdasarkan payment_date)
	var totalCashIn float64
	r.db.WithContext(ctx).Table("payments").
		Where("payment_date >= ? AND payment_date < ? AND deleted_at IS NULL", startDate, endDate).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalCashIn)

	// 4. Hitung Piutang Baru (Sisa tagihan DARI order yang masuk bulan ini)
	var piutangBaru float64
	r.db.WithContext(ctx).Table("orders").
		Select("COALESCE(SUM(orders.total_amount - (SELECT COALESCE(SUM(amount), 0) FROM payments WHERE payments.order_id = orders.id AND payments.deleted_at IS NULL)), 0)").
		Where("approved_at >= ? AND approved_at < ? AND deleted_at IS NULL", startDate, endDate).
		Scan(&piutangBaru)

	report.Summary = domain.MonthlySummary{
		TotalOmset:      totalOmset,
		TotalCashIn:     totalCashIn,
		TotalReceivable: piutangBaru,
		TotalOrderCount: int(totalOrders),
		TotalItemQty:    int(totalQty),
	}

	// 5. Leaderboard Sales
	r.db.WithContext(ctx).Table("orders").
		Select("users.id as sales_id, users.name as sales_name, COUNT(orders.id) as total_orders, SUM(orders.total_amount) as total_omset").
		Joins("JOIN users ON users.id = orders.sales_id").
		Where("orders.approved_at >= ? AND orders.approved_at < ? AND orders.deleted_at IS NULL", startDate, endDate).
		Group("users.id, users.name").
		Order("total_omset DESC").
		Scan(&report.SalesPerformances)

	// 6. Menyusun Daily Trends (Grafik Harian)
	type dailyResult struct {
		Date   string
		Amount float64
	}
	var omsetResults []dailyResult
	var cashInResults []dailyResult

	r.db.WithContext(ctx).Table("orders").
		Select("TO_CHAR(approved_at, 'YYYY-MM-DD') as date, SUM(total_amount) as amount").
		Where("approved_at >= ? AND approved_at < ? AND deleted_at IS NULL", startDate, endDate).
		Group("TO_CHAR(approved_at, 'YYYY-MM-DD')").Scan(&omsetResults)

	r.db.WithContext(ctx).Table("payments").
		Select("TO_CHAR(payment_date, 'YYYY-MM-DD') as date, SUM(amount) as amount").
		Where("payment_date >= ? AND payment_date < ? AND deleted_at IS NULL", startDate, endDate).
		Group("TO_CHAR(payment_date, 'YYYY-MM-DD')").Scan(&cashInResults)

	// Petakan hasil ke kalender bulan penuh agar grafik FE tidak bolong
	daysInMonth := endDate.AddDate(0, 0, -1).Day()
	omsetMap := make(map[string]float64)
	for _, r := range omsetResults {
		omsetMap[r.Date] = r.Amount
	}
	cashInMap := make(map[string]float64)
	for _, r := range cashInResults {
		cashInMap[r.Date] = r.Amount
	}

	for i := 1; i <= daysInMonth; i++ {
		dateStr := time.Date(year, time.Month(month), i, 0, 0, 0, 0, time.Local).Format("2006-01-02")

		// <-- Pastikan struct di sini menggunakan MonthlyDailyTrend
		report.DailyTrends = append(report.DailyTrends, domain.MonthlyDailyTrend{
			Date:        dateStr,
			OmsetAmount: omsetMap[dateStr],
			CashIn:      cashInMap[dateStr],
		})
	}

	return &report, nil
}
