package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/faridlan/sikon-api/internal/domain"
	"github.com/faridlan/sikon-api/internal/usecase"
)

// stubReportRepository implements domain.ReportRepository for testing.
type stubReportRepository struct {
	accountingReport  *domain.AccountingReport
	productionReport  *domain.ProductionReport
	poSummary         *domain.POSummaryReport
	receivablesDetail []domain.ReceivableDetail

	// Nilai kembalian khusus untuk Expense
	totalExpense float64
	totalHPP     float64

	// Menyimpan error untuk simulasi gagal
	err            error
	expenseErr     error
	expenseByPOErr error
}

func (s stubReportRepository) GetAccountingReportData(_ context.Context, _, _ time.Time) (*domain.AccountingReport, error) {
	return s.accountingReport, s.err
}

func (s stubReportRepository) GetProductionReportData(_ context.Context, _, _ int) (*domain.ProductionReport, error) {
	return s.productionReport, s.err
}

func (s stubReportRepository) GetPOSummaryData(_ context.Context, _ string) (*domain.POSummaryReport, error) {
	return s.poSummary, s.err
}

func (s stubReportRepository) GetReceivablesDetailData(_ context.Context) ([]domain.ReceivableDetail, error) {
	return s.receivablesDetail, s.err
}

// IMPLEMENTASI KONTRAK BARU (FINANCIAL EXPENSES)
func (s stubReportRepository) GetTotalExpenseByDateRange(_ context.Context, _, _ time.Time) (float64, error) {
	return s.totalExpense, s.expenseErr
}

func (s stubReportRepository) GetTotalExpenseByBatchPOs(_ context.Context, _ []string) (float64, error) {
	return s.totalHPP, s.expenseByPOErr
}

// ============================================================================
// TEST: GetAccountingReport
// ============================================================================
func TestReportUsecase_GetAccountingReport(t *testing.T) {
	t.Run("Success - Menghitung Net Profit dengan Benar", func(t *testing.T) {
		startDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC)

		// Omset 10jt, Uang Masuk 8jt
		mockData := &domain.AccountingReport{
			StartDate:  startDate,
			EndDate:    endDate,
			PeriodName: "01 Jul 2026 - 31 Jul 2026",
			Summary: domain.AccountingSummary{
				TotalOmset:      10000000,
				TotalCashIn:     8000000,
				TotalReceivable: 2000000,
			},
		}

		// Pengeluaran 3jt
		repo := stubReportRepository{
			accountingReport: mockData,
			totalExpense:     3000000, // Simulasi HPP + OPEX
		}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetAccountingReport(context.Background(), startDate, endDate)

		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Verifikasi Kalkulasi Matematika
		assert.Equal(t, float64(10000000), result.Summary.TotalOmset)
		assert.Equal(t, float64(3000000), result.Summary.TotalExpense)
		// Net Profit = Omset (10jt) - Expense (3jt) = 7jt
		assert.Equal(t, float64(7000000), result.Summary.NetProfit)
		// Net Cashflow = Cash In (8jt) - Expense (3jt) = 5jt
		assert.Equal(t, float64(5000000), result.Summary.NetCashflow)
	})

	t.Run("Error - EndDate before StartDate", func(t *testing.T) {
		startDate := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC) // Invalid

		repo := stubReportRepository{}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetAccountingReport(context.Background(), startDate, endDate)

		assert.Error(t, err)
		assert.Nil(t, result)

		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)
	})

	t.Run("Error - Repository GetAccountingReportData Fails", func(t *testing.T) {
		startDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC)

		repo := stubReportRepository{err: errors.New("db error")}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetAccountingReport(context.Background(), startDate, endDate)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Error - Repository GetTotalExpense Fails", func(t *testing.T) {
		startDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC)

		repo := stubReportRepository{
			accountingReport: &domain.AccountingReport{},
			expenseErr:       errors.New("failed get expense"),
		}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetAccountingReport(context.Background(), startDate, endDate)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// ============================================================================
// TEST: GetProductionReport
// ============================================================================
func TestReportUsecase_GetProductionReport(t *testing.T) {
	t.Run("Success - Menghitung Profit Produksi", func(t *testing.T) {
		mockData := &domain.ProductionReport{
			TargetMonth:  7,
			TargetYear:   2026,
			TotalRevenue: 50000000, // Revenue Edisi Ini
			ActiveBatchPOs: []domain.ProductionBatchPO{
				{ID: "po-1", Name: "Batch A"},
				{ID: "po-2", Name: "Batch B"},
			},
		}

		repo := stubReportRepository{
			productionReport: mockData,
			totalHPP:         35000000, // HPP dari PO-1 dan PO-2
		}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetProductionReport(context.Background(), 7, 2026)

		assert.NoError(t, err)
		assert.NotNil(t, result)

		assert.Equal(t, float64(50000000), result.TotalRevenue)
		assert.Equal(t, float64(35000000), result.TotalHPP)
		// Profit = 50jt - 35jt = 15jt
		assert.Equal(t, float64(15000000), result.NetProfit)
	})

	t.Run("Error - Invalid Month", func(t *testing.T) {
		repo := stubReportRepository{}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetProductionReport(context.Background(), 13, 2026)

		assert.Error(t, err)
		assert.Nil(t, result)

		var appErr *domain.AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.ErrBadParamInput, appErr.ErrType)
	})

	t.Run("Error - Repository GetProductionReport Fails", func(t *testing.T) {
		repo := stubReportRepository{err: errors.New("db error")}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetProductionReport(context.Background(), 7, 2026)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Error - Repository GetTotalExpenseByBatchPOs Fails", func(t *testing.T) {
		repo := stubReportRepository{
			productionReport: &domain.ProductionReport{},
			expenseByPOErr:   errors.New("db err on expenses"),
		}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetProductionReport(context.Background(), 7, 2026)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// ============================================================================
// TEST: GetPOSummaryReport (Tidak banyak berubah)
// ============================================================================
func TestReportUsecase_GetPOSummaryReport(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockSummary := &domain.POSummaryReport{
			POID:         "po-123",
			TotalRevenue: 15000000,
		}

		repo := stubReportRepository{poSummary: mockSummary}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetPOSummaryReport(context.Background(), "po-123")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "po-123", result.POID)
	})

	t.Run("Error - Empty ID", func(t *testing.T) {
		repo := stubReportRepository{}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetPOSummaryReport(context.Background(), "")

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Error - Repository Fails", func(t *testing.T) {
		repo := stubReportRepository{err: errors.New("po not found")}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetPOSummaryReport(context.Background(), "po-123")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// ============================================================================
// TEST: GetReceivablesDetailReport (Tidak banyak berubah)
// ============================================================================
func TestReportUsecase_GetReceivablesDetailReport(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		mockData := []domain.ReceivableDetail{
			{OrderNumber: "ORD-001", OutstandingAmount: 3000000},
		}

		repo := stubReportRepository{receivablesDetail: mockData}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetReceivablesDetailReport(context.Background())

		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "ORD-001", result[0].OrderNumber)
	})

	t.Run("Error - Repository Fails", func(t *testing.T) {
		repo := stubReportRepository{err: errors.New("db timeout")}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetReceivablesDetailReport(context.Background())

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
