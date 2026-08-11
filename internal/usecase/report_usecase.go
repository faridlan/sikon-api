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

	if endDate.Before(startDate) {
		return nil, domain.NewError(domain.ErrBadParamInput, "Tanggal akhir tidak boleh lebih kecil dari tanggal awal")
	}

	// 1. Ambil data dasar laporan akuntansi (Omset, Cash In, Piutang)
	report, err := u.reportRepo.GetAccountingReportData(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// 2. Ambil breakdown pengeluaran (HPP vs OPEX)
	totalHPP, totalOPEX, err := u.reportRepo.GetExpenseBreakdownByDateRange(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// 3. Kalkulasi Akuntansi Presisi SIKOn
	report.Summary.TotalHPP = totalHPP
	report.Summary.GrossProfit = report.Summary.TotalOmset - totalHPP
	report.Summary.TotalOPEX = totalOPEX
	report.Summary.NetProfit = report.Summary.GrossProfit - totalOPEX
	report.Summary.NetCashflow = report.Summary.TotalCashIn - (totalHPP + totalOPEX)

	return report, nil
}

// ============================================================================
// 2. JALUR PRODUKSI (Berdasarkan Edisi PO)
// ============================================================================
func (u *reportUsecase) GetProductionReport(c context.Context, targetMonth, targetYear int) (*domain.ProductionReport, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if targetMonth < 1 || targetMonth > 12 {
		return nil, domain.NewError(domain.ErrBadParamInput, "Bulan tidak valid (1-12)")
	}
	if targetYear < 2000 {
		return nil, domain.NewError(domain.ErrBadParamInput, "Tahun tidak valid")
	}

	report, err := u.reportRepo.GetProductionReportData(ctx, targetMonth, targetYear)
	if err != nil {
		return nil, err
	}

	var poIDs []string
	for _, po := range report.ActiveBatchPOs {
		poIDs = append(poIDs, po.ID)
	}

	totalHPP, err := u.reportRepo.GetTotalExpenseByBatchPOs(ctx, poIDs)
	if err != nil {
		return nil, err
	}

	report.TotalHPP = totalHPP
	report.GrossProfit = report.TotalRevenue - totalHPP

	return report, nil
}

// ============================================================================
// 3. JALUR SPESIFIK (Detail PO & Piutang)
// ============================================================================
func (u *reportUsecase) GetPOSummaryReport(c context.Context, poID string) (*domain.POSummaryReport, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if poID == "" {
		return nil, errors.New("PO ID tidak boleh kosong")
	}

	summary, err := u.reportRepo.GetPOSummaryData(ctx, poID)
	if err != nil {
		return nil, err
	}

	totalHPP, err := u.reportRepo.GetTotalExpenseByBatchPOs(ctx, []string{poID})
	if err != nil {
		return nil, err
	}

	summary.TotalHPP = totalHPP
	summary.GrossProfit = summary.TotalRevenue - totalHPP

	return summary, nil
}

func (u *reportUsecase) GetReceivablesDetailReport(c context.Context, filter domain.ReceivablesFilter) ([]domain.ReceivableDetail, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	data, err := u.reportRepo.GetReceivablesDetailData(ctx, filter)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// ============================================================================
// 4. JALUR HARIAN (DAILY REPORT)
// ============================================================================
func (u *reportUsecase) GetDailyReport(c context.Context, date time.Time) (*domain.DailyReport, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if date.IsZero() {
		date = time.Now()
	}

	report, err := u.reportRepo.GetDailyReportData(ctx, date)
	if err != nil {
		return nil, err
	}

	return report, nil
}
