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

// ============================================================================
// 1. JALUR AKUNTANSI (Rentang Kalender Dinamis: Harian, Mingguan, Bulanan, Tahunan)
// ============================================================================
func (r *reportRepository) GetAccountingReportData(ctx context.Context, startDate, endDate time.Time) (*domain.AccountingReport, error) {
	report := &domain.AccountingReport{
		StartDate: startDate,
		EndDate:   endDate,
		// Menyesuaikan format jika di hari yang sama (Laporan Harian) atau beda hari
		PeriodName: startDate.Format("02 Jan 2006") + " - " + endDate.Format("02 Jan 2006"),
	}

	// 1. Summary Omset & Orders (Berdasarkan rentang approved_at)
	var totalOrders int64
	var totalOmset float64
	r.db.WithContext(ctx).Table("orders").
		Where("approved_at >= ? AND approved_at <= ? AND deleted_at IS NULL", startDate, endDate).
		Count(&totalOrders)

	r.db.WithContext(ctx).Table("orders").
		Where("approved_at >= ? AND approved_at <= ? AND deleted_at IS NULL", startDate, endDate).
		Select("COALESCE(SUM(total_amount), 0)").Scan(&totalOmset)

	// 2. Summary Qty Barang (Dari order yang deal di rentang tersebut)
	var totalQty int64
	r.db.WithContext(ctx).Table("order_items").
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.approved_at >= ? AND orders.approved_at <= ? AND orders.deleted_at IS NULL AND order_items.deleted_at IS NULL", startDate, endDate).
		Select("COALESCE(SUM(order_items.qty), 0)").Scan(&totalQty)

	// 3. Summary Cash In (Berdasarkan payment_date)
	var totalCashIn float64
	r.db.WithContext(ctx).Table("payments").
		Where("payment_date >= ? AND payment_date <= ? AND deleted_at IS NULL", startDate, endDate).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalCashIn)

	// 4. Hitung Piutang Baru yang Tercipta di Rentang Ini
	var piutangBaru float64
	r.db.WithContext(ctx).Table("orders").
		Select("COALESCE(SUM(orders.total_amount - (SELECT COALESCE(SUM(amount), 0) FROM payments WHERE payments.order_id = orders.id AND payments.deleted_at IS NULL)), 0)").
		Where("approved_at >= ? AND approved_at <= ? AND deleted_at IS NULL", startDate, endDate).
		Scan(&piutangBaru)

	report.Summary = domain.AccountingSummary{
		TotalOmset:      totalOmset,
		TotalCashIn:     totalCashIn,
		TotalReceivable: piutangBaru,
		TotalOrderCount: int(totalOrders),
		TotalItemQty:    int(totalQty),
	}

	// 5. Leaderboard Sales (Siapa penyumbang omset di periode ini)
	r.db.WithContext(ctx).Table("orders").
		Select("users.id as sales_id, users.name as sales_name, COUNT(orders.id) as total_orders, SUM(orders.total_amount) as total_omset").
		Joins("JOIN users ON users.id = orders.sales_id").
		Where("orders.approved_at >= ? AND orders.approved_at <= ? AND orders.deleted_at IS NULL", startDate, endDate).
		Group("users.id, users.name").
		Order("total_omset DESC").
		Scan(&report.SalesPerformances)

	// 6. Menyusun Grafik Trend (Per Hari)
	type dailyResult struct {
		Date   string
		Amount float64
	}
	var omsetResults, cashInResults []dailyResult

	r.db.WithContext(ctx).Table("orders").
		Select("TO_CHAR(approved_at, 'YYYY-MM-DD') as date, SUM(total_amount) as amount").
		Where("approved_at >= ? AND approved_at <= ? AND deleted_at IS NULL", startDate, endDate).
		Group("TO_CHAR(approved_at, 'YYYY-MM-DD')").Scan(&omsetResults)

	r.db.WithContext(ctx).Table("payments").
		Select("TO_CHAR(payment_date, 'YYYY-MM-DD') as date, SUM(amount) as amount").
		Where("payment_date >= ? AND payment_date <= ? AND deleted_at IS NULL", startDate, endDate).
		Group("TO_CHAR(payment_date, 'YYYY-MM-DD')").Scan(&cashInResults)

	omsetMap := make(map[string]float64)
	cashInMap := make(map[string]float64)
	for _, r := range omsetResults {
		omsetMap[r.Date] = r.Amount
	}
	for _, r := range cashInResults {
		cashInMap[r.Date] = r.Amount
	}

	// Loop mengisi array agar grafik tidak bolong
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		report.DailyTrends = append(report.DailyTrends, domain.AccountingDailyTrend{
			Date:        dateStr,
			OmsetAmount: omsetMap[dateStr],
			CashIn:      cashInMap[dateStr],
		})
	}

	return report, nil
}

// ============================================================================
// 2. JALUR PRODUKSI (Kinerja Berdasarkan Edisi PO)
// ============================================================================
func (r *reportRepository) GetProductionReportData(ctx context.Context, targetMonth, targetYear int) (*domain.ProductionReport, error) {
	report := &domain.ProductionReport{
		TargetMonth: targetMonth,
		TargetYear:  targetYear,
	}

	// 1. Ambil Semua PO di Edisi Ini
	var batchPOs []domain.BatchPO
	r.db.WithContext(ctx).Table("batch_pos").
		Where("target_month = ? AND target_year = ? AND deleted_at IS NULL", targetMonth, targetYear).
		Find(&batchPOs)

	var totalQuota int
	var poIDs []string
	for _, bp := range batchPOs {
		totalQuota += bp.Quota
		poIDs = append(poIDs, bp.ID)
		report.ActiveBatchPOs = append(report.ActiveBatchPOs, domain.ProductionBatchPO{
			ID:     bp.ID,
			Name:   bp.Name,
			Status: bp.Status,
			Quota:  bp.Quota,
		})
	}
	report.TotalQuota = totalQuota

	if len(poIDs) == 0 {
		return report, nil
	}

	// 2. Hitung Finansial
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
		Where("o.batch_po_id IN (?) AND o.deleted_at IS NULL", poIDs).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Scan(&financial)

	report.TotalRevenue = financial.TotalRevenue
	report.TotalPaid = financial.TotalPaid
	report.TotalOutstanding = financial.TotalOutstanding

	// 3. Rekap Produk & Kuantitas
	r.db.WithContext(ctx).Table("order_items oi").
		Select("c.name as category_name, SUM(oi.qty) as total_qty").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Joins("JOIN products p ON p.id = oi.product_id").
		Joins("JOIN categories c ON c.id = p.category_id").
		Where("o.batch_po_id IN (?) AND o.deleted_at IS NULL AND oi.deleted_at IS NULL", poIDs).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("c.name").
		Scan(&report.ProductSummary)

	for _, p := range report.ProductSummary {
		report.TotalQtyOrdered += p.TotalQty
	}

	remaining := int64(report.TotalQuota) - report.TotalQtyOrdered
	if remaining < 0 {
		remaining = 0
	}
	report.RemainingQuota = remaining

	// 4. Rekap Performa Sales
	r.db.WithContext(ctx).Table("orders o").
		Select("u.name as sales_name, SUM(oi.qty) as total_qty, SUM(DISTINCT o.total_amount) as total_revenue").
		Joins("JOIN users u ON u.id = o.sales_id").
		Joins("LEFT JOIN order_items oi ON oi.order_id = o.id AND oi.deleted_at IS NULL").
		Where("o.batch_po_id IN (?) AND o.deleted_at IS NULL", poIDs).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("u.name").
		Order("total_revenue DESC").
		Scan(&report.SalesSummary)

	return report, nil
}

// ============================================================================
// 3. JALUR SPESIFIK (Summary per PO & Daftar Piutang) -- DI-UPDATE
// ============================================================================
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

	// 2. Kalkulasi Finansial
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

	// 3. Rekap Produk
	r.db.WithContext(ctx).Table("order_items oi").
		Select("c.name as category_name, SUM(oi.qty) as total_qty").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Joins("JOIN products p ON p.id = oi.product_id").
		Joins("JOIN categories c ON c.id = p.category_id").
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL AND oi.deleted_at IS NULL", poID).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("c.name").
		Scan(&summary.ProductSummary)

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
		Order("total_revenue DESC").
		Scan(&summary.SalesSummary)

	// 5. TAMBAHAN: Daftar Piutang per Customer spesifik di PO ini
	r.db.WithContext(ctx).Table("orders o").
		Select(`
			c.name as customer_name,
			SUM(o.total_amount) as total_amount,
			COALESCE(SUM(p.total_paid), 0) as total_paid,
			SUM(o.total_amount - COALESCE(p.total_paid, 0)) as outstanding_amount
		`).
		Joins("JOIN customers c ON c.id = o.customer_id AND c.deleted_at IS NULL").
		Joins("LEFT JOIN (SELECT order_id, SUM(amount) as total_paid FROM payments WHERE deleted_at IS NULL GROUP BY order_id) p ON p.order_id = o.id").
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL", poID).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("c.name").
		Having("SUM(o.total_amount - COALESCE(p.total_paid, 0)) > 0"). // Hanya yang masih ada hutang
		Order("outstanding_amount DESC").
		Scan(&summary.CustomerReceivables)

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
		Where("(o.total_amount - COALESCE(p.total_paid, 0)) > 0").
		Order("outstanding_amount DESC").
		Scan(&details).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	return details, nil
}

// ============================================================================
// 4. HELPER FINANSIAL (EXPENSES)
// ============================================================================

func (r *reportRepository) GetTotalExpenseByDateRange(ctx context.Context, startDate, endDate time.Time) (float64, error) {
	var total float64
	err := r.db.WithContext(ctx).Table("expenses").
		Where("expense_date >= ? AND expense_date <= ? AND deleted_at IS NULL", startDate, endDate).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error

	if err != nil {
		return 0, TranslateError(err)
	}
	return total, nil
}

func (r *reportRepository) GetTotalExpenseByBatchPOs(ctx context.Context, poIDs []string) (float64, error) {
	if len(poIDs) == 0 {
		return 0, nil
	}
	var total float64
	err := r.db.WithContext(ctx).Table("expenses").
		Where("batch_po_id IN ? AND deleted_at IS NULL", poIDs).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error

	if err != nil {
		return 0, TranslateError(err)
	}
	return total, nil
}

// ============================================================================
// 5. JALUR HARIAN (DAILY TACTICAL REPORT) - BARU
// ============================================================================
func (r *reportRepository) GetDailyReportData(ctx context.Context, targetDate time.Time) (*domain.DailyReport, error) {
	dateStr := targetDate.Format("2006-01-02")
	report := &domain.DailyReport{
		ReportDate: dateStr,
	}

	// 1. Cek PO Aktif (Asumsi: Hanya ada 1 PO aktif di sistem, kita ambil yang terbaru/pertama)
	var po domain.BatchPO
	err := r.db.WithContext(ctx).Table("batch_pos").
		Where("status = ? AND deleted_at IS NULL", domain.BatchPOStatusActive).
		Order("created_at DESC").
		First(&po).Error

	// Jika tidak ada PO aktif sama sekali, kembalikan report kosong
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return report, nil
		}
		return nil, TranslateError(err)
	}

	// 2. Set PO Info & Kalkulasi Qty PO
	var qtyTotalPO int64
	r.db.WithContext(ctx).Table("order_items oi").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL AND oi.deleted_at IS NULL", po.ID).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Select("COALESCE(SUM(oi.qty), 0)").Scan(&qtyTotalPO)

	remQuota := int64(po.Quota) - qtyTotalPO
	if remQuota < 0 {
		remQuota = 0
	}

	report.POInfo = domain.DailyPOInfo{
		POID:           po.ID,
		POName:         po.Name,
		Quota:          po.Quota,
		RemainingQuota: remQuota,
	}
	report.OrderSummary.QtyTotalPO = qtyTotalPO

	// 3. Kalkulasi Qty Hari Ini (Dari order yang deal hari ini di PO tsb)
	var qtyToday int64
	r.db.WithContext(ctx).Table("order_items oi").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL AND oi.deleted_at IS NULL", po.ID).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Where("DATE(o.approved_at) = DATE(?)", targetDate).
		Select("COALESCE(SUM(oi.qty), 0)").Scan(&qtyToday)

	report.OrderSummary.QtyToday = qtyToday

	// 4. Financial Summary (Cash in hari ini, dan outstanding)
	var totalRevenueToday, totalPaidToday float64
	// Omset masuk HARI INI
	r.db.WithContext(ctx).Table("orders").
		Where("DATE(approved_at) = DATE(?) AND deleted_at IS NULL", targetDate).
		Select("COALESCE(SUM(total_amount), 0)").Scan(&totalRevenueToday)
	// Cash masuk HARI INI
	r.db.WithContext(ctx).Table("payments").
		Where("DATE(payment_date) = DATE(?) AND deleted_at IS NULL", targetDate).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalPaidToday)

	// Piutang PO Aktif
	var activeOut float64
	r.db.WithContext(ctx).Table("orders o").
		Select("COALESCE(SUM(o.total_amount - (SELECT COALESCE(SUM(amount), 0) FROM payments WHERE payments.order_id = o.id AND payments.deleted_at IS NULL)), 0)").
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL AND o.order_status IN (?, ?)", po.ID, domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Scan(&activeOut)

	// Piutang PO Sebelumnya (Bukan PO yang saat ini aktif)
	var prevOut float64
	r.db.WithContext(ctx).Table("orders o").
		Select("COALESCE(SUM(o.total_amount - (SELECT COALESCE(SUM(amount), 0) FROM payments WHERE payments.order_id = o.id AND payments.deleted_at IS NULL)), 0)").
		Where("o.batch_po_id != ? AND o.deleted_at IS NULL AND o.order_status IN (?, ?)", po.ID, domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Scan(&prevOut)

	report.FinancialSummary = domain.DailyFinancialSummary{
		TotalRevenue:          totalRevenueToday,
		TotalPaid:             totalPaidToday,
		ActivePOOutstanding:   activeOut,
		PreviousPOOutstanding: prevOut,
		TotalOutstanding:      activeOut + prevOut,
	}

	// 5. Trend Data (7 Hari Terakhir untuk PO ini)
	trendData := make([]domain.DailyTrend, 0)
	for i := 6; i >= 0; i-- {
		d := targetDate.AddDate(0, 0, -i)
		dateStr := d.Format("2006-01-02")

		var qty int64
		r.db.WithContext(ctx).Table("order_items oi").
			Joins("JOIN orders o ON o.id = oi.order_id").
			Where("o.batch_po_id = ? AND o.deleted_at IS NULL AND oi.deleted_at IS NULL", po.ID).
			Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
			Where("DATE(o.approved_at) = DATE(?)", d).
			Select("COALESCE(SUM(oi.qty), 0)").Scan(&qty)

		trendData = append(trendData, domain.DailyTrend{
			Date: dateStr,
			Qty:  qty,
		})
	}
	report.OrderSummary.TrendData = trendData

	// 6. Sales Details Hari Ini (Agregasi multi-tabel via Go code karena JSON nested)
	type salesRaw struct {
		SalesID   string
		SalesName string
		CatName   string
		Qty       int64
	}
	var rawData []salesRaw
	r.db.WithContext(ctx).Table("order_items oi").
		Select("u.id as sales_id, u.name as sales_name, c.name as cat_name, SUM(oi.qty) as qty").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Joins("JOIN users u ON u.id = o.sales_id").
		Joins("JOIN products p ON p.id = oi.product_id").
		Joins("JOIN categories c ON c.id = p.category_id").
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL AND oi.deleted_at IS NULL", po.ID).
		Where("DATE(o.approved_at) = DATE(?)", targetDate).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("u.id, u.name, c.name").
		Scan(&rawData)

	salesMap := make(map[string]*domain.DailySalesDetail)
	for _, row := range rawData {
		if _, exists := salesMap[row.SalesID]; !exists {
			salesMap[row.SalesID] = &domain.DailySalesDetail{
				SalesName:  row.SalesName,
				Categories: make(map[string]int64),
				TotalQty:   0,
			}
		}
		salesMap[row.SalesID].Categories[row.CatName] += row.Qty
		salesMap[row.SalesID].TotalQty += row.Qty
	}

	for _, v := range salesMap {
		report.SalesDetails = append(report.SalesDetails, *v)
	}

	return report, nil
}
