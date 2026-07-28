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

	return u.reportRepo.GetAccountingReportData(ctx, startDate, endDate)
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

	return u.reportRepo.GetProductionReportData(ctx, targetMonth, targetYear)
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
