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

	// -------------------------------------------------------------------------
	// 4. REKAP PERFORMA SALES (DENGAN DETAIL KATEGORI)
	// -------------------------------------------------------------------------
	salesMap := make(map[string]*domain.POSalesSummary)

	// a. Ambil Total Revenue per Sales
	type salesRev struct {
		SalesName string
		TotalRev  float64
	}
	var revs []salesRev
	r.db.WithContext(ctx).Table("orders o").
		Select("u.name as sales_name, SUM(o.total_amount) as total_rev").
		Joins("JOIN users u ON u.id = o.sales_id").
		Where("o.batch_po_id IN (?) AND o.deleted_at IS NULL", poIDs).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("u.name").
		Scan(&revs)

	for _, rv := range revs {
		salesMap[rv.SalesName] = &domain.POSalesSummary{
			SalesName:    rv.SalesName,
			TotalRevenue: rv.TotalRev,
			Categories:   make(map[string]int64),
		}
	}

	// b. Ambil Total Qty dan Detail Kategori per Sales
	type salesCat struct {
		SalesName string
		CatName   string
		Qty       int64
	}
	var cats []salesCat
	r.db.WithContext(ctx).Table("order_items oi").
		Select("u.name as sales_name, c.name as cat_name, SUM(oi.qty) as qty").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Joins("JOIN users u ON u.id = o.sales_id").
		Joins("JOIN products p ON p.id = oi.product_id").
		Joins("JOIN categories c ON c.id = p.category_id").
		Where("o.batch_po_id IN (?) AND o.deleted_at IS NULL AND oi.deleted_at IS NULL", poIDs).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("u.name, c.name").
		Scan(&cats)

	for _, ct := range cats {
		if _, ok := salesMap[ct.SalesName]; ok {
			salesMap[ct.SalesName].Categories[ct.CatName] += ct.Qty
			salesMap[ct.SalesName].TotalQty += ct.Qty
		}
	}

	for _, v := range salesMap {
		report.SalesSummary = append(report.SalesSummary, *v)
	}
	// -------------------------------------------------------------------------
	// 5. TREND DATA BERDASARKAN RENTANG BATCH PO (BUKAN KALENDER)
	// -------------------------------------------------------------------------
	// Cari tanggal Start Date terawal dan End Date terakhir dari kumpulan PO edisi ini
	var minDate, maxDate time.Time
	if len(batchPOs) > 0 {
		minDate = batchPOs[0].StartDate
		maxDate = batchPOs[0].EndDate
		for _, bp := range batchPOs {
			if bp.StartDate.Before(minDate) {
				minDate = bp.StartDate
			}
			if bp.EndDate.After(maxDate) {
				maxDate = bp.EndDate
			}
		}
	} else {
		// Fallback (meskipun seharusnya tidak akan tereksekusi karena sudah ada return jika PO kosong)
		minDate = time.Date(targetYear, time.Month(targetMonth), 1, 0, 0, 0, 0, time.UTC)
		maxDate = minDate.AddDate(0, 1, -1)
	}

	type trendRaw struct {
		Date string
		Qty  int64
	}
	var tRaws []trendRaw
	r.db.WithContext(ctx).Table("order_items oi").
		Select("TO_CHAR(o.approved_at, 'YYYY-MM-DD') as date, SUM(oi.qty) as qty").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Where("o.batch_po_id IN (?) AND o.deleted_at IS NULL AND oi.deleted_at IS NULL", poIDs).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("TO_CHAR(o.approved_at, 'YYYY-MM-DD')").
		Scan(&tRaws)

	trendMap := make(map[string]int64)
	for _, tr := range tRaws {
		trendMap[tr.Date] = tr.Qty
	}

	// Looping dinamis dari tanggal paling awal di PO pertama, s/d tanggal terakhir di PO terakhir
	for d := minDate; !d.After(maxDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		report.TrendData = append(report.TrendData, domain.DailyTrend{
			Date: dateStr,
			Qty:  trendMap[dateStr],
		})
	}

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

	// 3. Rekap Produk & Sisa Kuota
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

	remQuota := int64(po.Quota) - summary.TotalQtyOrdered
	if remQuota < 0 {
		remQuota = 0
	}
	summary.RemainingQuota = remQuota

	// 4. Rekap Performa Sales (Dengan Detail Kategori via Go Mapping)
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
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL AND oi.deleted_at IS NULL", poID).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("u.id, u.name, c.name").
		Scan(&rawData)

	// Ambil Total Revenue per Sales
	type salesRev struct {
		SalesName string
		TotalRev  float64
	}
	var revs []salesRev
	r.db.WithContext(ctx).Table("orders o").
		Select("u.name as sales_name, SUM(o.total_amount) as total_rev").
		Joins("JOIN users u ON u.id = o.sales_id").
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL", poID).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("u.name").
		Scan(&revs)

	salesMap := make(map[string]*domain.POSalesSummary)
	for _, rv := range revs {
		salesMap[rv.SalesName] = &domain.POSalesSummary{
			SalesName:    rv.SalesName,
			TotalRevenue: rv.TotalRev,
			Categories:   make(map[string]int64),
		}
	}

	for _, row := range rawData {
		if _, exists := salesMap[row.SalesName]; exists {
			salesMap[row.SalesName].Categories[row.CatName] += row.Qty
			salesMap[row.SalesName].TotalQty += row.Qty
		}
	}

	for _, v := range salesMap {
		summary.SalesSummary = append(summary.SalesSummary, *v)
	}

	// 5. Trend Data Harian Selama Rentang PO Berjalan (StartDate s/d EndDate)
	type trendRaw struct {
		Date string
		Qty  int64
	}
	var tRaws []trendRaw
	r.db.WithContext(ctx).Table("order_items oi").
		Select("TO_CHAR(o.approved_at, 'YYYY-MM-DD') as date, SUM(oi.qty) as qty").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL AND oi.deleted_at IS NULL", poID).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("TO_CHAR(o.approved_at, 'YYYY-MM-DD')").
		Scan(&tRaws)

	trendMap := make(map[string]int64)
	for _, tr := range tRaws {
		trendMap[tr.Date] = tr.Qty
	}

	for d := po.StartDate; !d.After(po.EndDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		summary.TrendData = append(summary.TrendData, domain.DailyTrend{
			Date: dateStr,
			Qty:  trendMap[dateStr],
		})
	}

	// 6. Daftar Piutang per Customer spesifik di PO ini
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
		Having("SUM(o.total_amount - COALESCE(p.total_paid, 0)) > 0").
		Order("outstanding_amount DESC").
		Scan(&summary.CustomerReceivables)

	// 7. Kalkulasi Expense (HPP) & Net Profit khusus PO ini
	var totalHPP float64
	r.db.WithContext(ctx).Table("expenses").
		Where("batch_po_id = ? AND deleted_at IS NULL", poID).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalHPP)

	summary.TotalHPP = totalHPP
	summary.NetProfit = summary.TotalRevenue - totalHPP

	return summary, nil
}

func (r *reportRepository) GetReceivablesDetailData(ctx context.Context, filter domain.ReceivablesFilter) ([]domain.ReceivableDetail, error) {
	var details []domain.ReceivableDetail

	query := r.db.WithContext(ctx).Table("orders o").
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
		Where("(o.total_amount - COALESCE(p.total_paid, 0)) > 0")

	// 1. Filter berdasarkan Batch PO ID (Jika diisi)
	if filter.BatchPoID != "" {
		query = query.Where("o.batch_po_id = ?", filter.BatchPoID)
	}

	// 2. Filter berdasarkan Order Status (Opsional jika ingin dispesifikan)
	if filter.OrderStatus != "" {
		query = query.Where("o.order_status = ?", filter.OrderStatus)
	}

	// 3. Pengurutan Data (Sorting Dinamis)
	switch filter.SortBy {
	case "amount_asc":
		query = query.Order("outstanding_amount ASC")
	case "amount_desc":
		query = query.Order("outstanding_amount DESC")
	case "date_asc":
		query = query.Order("o.created_at ASC")
	case "date_desc":
		query = query.Order("o.created_at DESC")
	default:
		// Default behavior lama (Tagihan terbesar ke terkecil)
		query = query.Order("outstanding_amount DESC")
	}

	err := query.Scan(&details).Error
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
// 5. JALUR HARIAN (DAILY TACTICAL REPORT)
// ============================================================================
func (r *reportRepository) GetDailyReportData(ctx context.Context, targetDate time.Time) (*domain.DailyReport, error) {
	dateStr := targetDate.Format("2006-01-02")
	report := &domain.DailyReport{
		ReportDate: dateStr,
	}

	// 1. Cek PO Berdasarkan Tanggal (Time Travel Logic)
	var po domain.BatchPO
	err := r.db.WithContext(ctx).Table("batch_pos").
		Where("DATE(start_date) <= DATE(?) AND DATE(end_date) >= DATE(?) AND deleted_at IS NULL", targetDate, targetDate).
		Order("end_date ASC"). // PERBAIKAN: Prioritaskan PO yang akan segera DITUTUP jika ada tanggal yang beririsan
		First(&po).Error

	// Skenario B (Fallback): Jika tidak ada PO di hari itu, ambil PO TERAKHIR sebelum tanggal tersebut
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			errFallback := r.db.WithContext(ctx).Table("batch_pos").
				Where("DATE(end_date) < DATE(?) AND deleted_at IS NULL", targetDate).
				Order("end_date DESC"). // Ambil yang end_date-nya paling dekat dengan hari ini
				First(&po).Error

			// Jika ternyata memang belum pernah ada PO sama sekali di database
			if errFallback != nil {
				if errFallback == gorm.ErrRecordNotFound {
					return report, nil
				}
				return nil, TranslateError(errFallback)
			}
		} else {
			return nil, TranslateError(err) // Error database beneran
		}
	}

	// 2. Set PO Info & Kalkulasi Qty PO (Diisolasi HANYA sampai targetDate)
	var qtyTotalPO int64
	r.db.WithContext(ctx).Table("order_items oi").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL AND oi.deleted_at IS NULL", po.ID).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Where("DATE(o.approved_at) <= DATE(?)", targetDate). // Cegah pesanan masa depan memotong kuota masa lalu
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

	// 4. Financial Summary (Time Travel Snapshot)
	var totalRevenueToday, totalPaidToday float64
	// Omset masuk HARI INI
	r.db.WithContext(ctx).Table("orders").
		Where("DATE(approved_at) = DATE(?) AND deleted_at IS NULL", targetDate).
		Select("COALESCE(SUM(total_amount), 0)").Scan(&totalRevenueToday)
	// Cash masuk HARI INI
	r.db.WithContext(ctx).Table("payments").
		Where("DATE(payment_date) = DATE(?) AND deleted_at IS NULL", targetDate).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalPaidToday)

	// Piutang PO Aktif (Order & Payment diisolasi s/d targetDate)
	var activeOut float64
	r.db.WithContext(ctx).Table("orders o").
		Select("COALESCE(SUM(o.total_amount - (SELECT COALESCE(SUM(amount), 0) FROM payments p WHERE p.order_id = o.id AND p.deleted_at IS NULL AND DATE(p.payment_date) <= DATE(?))), 0)", targetDate).
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL AND o.order_status IN (?, ?)", po.ID, domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Where("DATE(o.approved_at) <= DATE(?)", targetDate).
		Scan(&activeOut)

	// Piutang PO Sebelumnya (Order & Payment diisolasi s/d targetDate)
	var prevOut float64
	r.db.WithContext(ctx).Table("orders o").
		Select("COALESCE(SUM(o.total_amount - (SELECT COALESCE(SUM(amount), 0) FROM payments p WHERE p.order_id = o.id AND p.deleted_at IS NULL AND DATE(p.payment_date) <= DATE(?))), 0)", targetDate).
		Where("o.batch_po_id != ? AND o.deleted_at IS NULL AND o.order_status IN (?, ?)", po.ID, domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Where("DATE(o.approved_at) <= DATE(?)", targetDate).
		Scan(&prevOut)

	report.FinancialSummary = domain.DailyFinancialSummary{
		TotalRevenue:          totalRevenueToday,
		TotalPaid:             totalPaidToday,
		ActivePOOutstanding:   activeOut,
		PreviousPOOutstanding: prevOut,
		TotalOutstanding:      activeOut + prevOut,
	}

	// -------------------------------------------------------------------------
	// 5. TREND DATA HARIAN (Dari Awal PO Dimulai s/d Tanggal Request)
	// -------------------------------------------------------------------------
	type trendRaw struct {
		Date string
		Qty  int64
	}
	var tRaws []trendRaw
	r.db.WithContext(ctx).Table("order_items oi").
		Select("TO_CHAR(o.approved_at, 'YYYY-MM-DD') as date, SUM(oi.qty) as qty").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL AND oi.deleted_at IS NULL", po.ID).
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Where("DATE(o.approved_at) >= DATE(?) AND DATE(o.approved_at) <= DATE(?)", po.StartDate, targetDate).
		Group("TO_CHAR(o.approved_at, 'YYYY-MM-DD')").
		Scan(&tRaws)

	trendMap := make(map[string]int64)
	for _, tr := range tRaws {
		trendMap[tr.Date] = tr.Qty
	}

	trendData := make([]domain.DailyTrend, 0)
	// Looping presisi: Mulai dari hari pertama PO dibuka, sampai hari request (Time Travel)
	for d := po.StartDate; !d.After(targetDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		trendData = append(trendData, domain.DailyTrend{
			Date: dateStr,
			Qty:  trendMap[dateStr],
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

	// -------------------------------------------------------------------------
	// 7. SALES DETAILS KESELURUHAN PO (TIME TRAVEL s/d targetDate)
	// -------------------------------------------------------------------------
	var rawDataPO []salesRaw
	r.db.WithContext(ctx).Table("order_items oi").
		Select("u.id as sales_id, u.name as sales_name, c.name as cat_name, SUM(oi.qty) as qty").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Joins("JOIN users u ON u.id = o.sales_id").
		Joins("JOIN products p ON p.id = oi.product_id").
		Joins("JOIN categories c ON c.id = p.category_id").
		Where("o.batch_po_id = ? AND o.deleted_at IS NULL AND oi.deleted_at IS NULL", po.ID).
		Where("DATE(o.approved_at) <= DATE(?)", targetDate). // Penting: Hanya ambil pencapaian sampai tanggal request!
		Where("o.order_status IN (?, ?)", domain.OrderStatusProduction, domain.OrderStatusCompleted).
		Group("u.id, u.name, c.name").
		Scan(&rawDataPO)

	salesMapPO := make(map[string]*domain.DailySalesDetail)
	for _, row := range rawDataPO {
		if _, exists := salesMapPO[row.SalesID]; !exists {
			salesMapPO[row.SalesID] = &domain.DailySalesDetail{
				SalesName:  row.SalesName,
				Categories: make(map[string]int64),
				TotalQty:   0,
			}
		}
		salesMapPO[row.SalesID].Categories[row.CatName] += row.Qty
		salesMapPO[row.SalesID].TotalQty += row.Qty
	}

	for _, v := range salesMapPO {
		report.POSalesDetails = append(report.POSalesDetails, *v)
	}

	return report, nil
}
