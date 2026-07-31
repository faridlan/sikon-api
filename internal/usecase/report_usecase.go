package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type reportUsecase struct {
	reportRepo     domain.ReportRepository
	contextTimeout time.Duration
}

func NewReportUsecase(rr domain.ReportRepository, timeout time.Duration) domain.ReportUsecase {
	return &reportUsecase{reportRepo: rr, contextTimeout: timeout}
}

// ============================================================================
// 1. JALUR AKUNTANSI (Rentang Kalender)
// ============================================================================
func (u *reportUsecase) GetAccountingReport(c context.Context, startDate, endDate time.Time) (*domain.AccountingReport, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Validasi dasar: End Date tidak boleh mendahului Start Date
	if endDate.Before(startDate) {
		return nil, domain.NewError(domain.ErrBadParamInput, "Tanggal akhir tidak boleh lebih kecil dari tanggal awal")
	}

	// 1. Ambil data dasar laporan akuntansi
	report, err := u.reportRepo.GetAccountingReportData(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// 2. Ambil total pengeluaran di rentang tanggal yang sama
	totalExpense, err := u.reportRepo.GetTotalExpenseByDateRange(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// 3. Kalkulasi Net Profit & Net Cashflow
	report.Summary.TotalExpense = totalExpense
	report.Summary.NetProfit = report.Summary.TotalOmset - totalExpense
	report.Summary.NetCashflow = report.Summary.TotalCashIn - totalExpense

	return report, nil
}

// ============================================================================
// 2. JALUR PRODUKSI (Berdasarkan Edisi PO)
// ============================================================================
func (u *reportUsecase) GetProductionReport(c context.Context, targetMonth, targetYear int) (*domain.ProductionReport, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Validasi input
	if targetMonth < 1 || targetMonth > 12 {
		return nil, domain.NewError(domain.ErrBadParamInput, "Bulan tidak valid (1-12)")
	}
	if targetYear < 2000 {
		return nil, domain.NewError(domain.ErrBadParamInput, "Tahun tidak valid")
	}

	// 1. Ambil data dasar laporan produksi
	report, err := u.reportRepo.GetProductionReportData(ctx, targetMonth, targetYear)
	if err != nil {
		return nil, err
	}

	// 2. Kumpulkan ID PO aktif untuk mencari pengeluaran spesifik
	var poIDs []string
	for _, po := range report.ActiveBatchPOs {
		poIDs = append(poIDs, po.ID)
	}

	// 3. Ambil total pengeluaran (HPP) khusus untuk PO-PO tersebut
	totalHPP, err := u.reportRepo.GetTotalExpenseByBatchPOs(ctx, poIDs)
	if err != nil {
		return nil, err
	}

	// 4. Kalkulasi Profit Produksi (Hanya dari pesanan yang masuk edisi ini)
	report.TotalHPP = totalHPP
	report.NetProfit = report.TotalRevenue - totalHPP

	return report, nil
}

// ============================================================================
// 3. JALUR SPESIFIK (Detail PO & Piutang) -- DI-UPDATE
// ============================================================================
func (u *reportUsecase) GetPOSummaryReport(c context.Context, poID string) (*domain.POSummaryReport, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if poID == "" {
		return nil, errors.New("PO ID tidak boleh kosong")
	}

	// 1. Ambil data summary (termasuk list customer receivables)
	summary, err := u.reportRepo.GetPOSummaryData(ctx, poID)
	if err != nil {
		return nil, err
	}

	// 2. Ambil pengeluaran (HPP) HANYA untuk PO ini
	totalHPP, err := u.reportRepo.GetTotalExpenseByBatchPOs(ctx, []string{poID})
	if err != nil {
		return nil, err
	}

	// 3. Set kalkulasi finansial spesifik PO ini
	summary.TotalHPP = totalHPP
	summary.NetProfit = summary.TotalRevenue - totalHPP

	return summary, nil
}

func (u *reportUsecase) GetReceivablesDetailReport(c context.Context) ([]domain.ReceivableDetail, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	data, err := u.reportRepo.GetReceivablesDetailData(ctx)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// ============================================================================
// 4. JALUR HARIAN (DAILY REPORT) - BARU
// ============================================================================
func (u *reportUsecase) GetDailyReport(c context.Context, date time.Time) (*domain.DailyReport, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Jika tanggal tidak diset secara eksplisit, gunakan waktu saat ini (hari ini)
	if date.IsZero() {
		date = time.Now()
	}

	report, err := u.reportRepo.GetDailyReportData(ctx, date)
	if err != nil {
		return nil, err
	}

	return report, nil
}
