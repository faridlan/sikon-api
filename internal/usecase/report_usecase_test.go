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
	err               error
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

// ============================================================================
// TEST: GetAccountingReport
// ============================================================================
func TestReportUsecase_GetAccountingReport(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		startDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC)

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

		repo := stubReportRepository{accountingReport: mockData}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetAccountingReport(context.Background(), startDate, endDate)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, float64(10000000), result.Summary.TotalOmset)
		assert.Equal(t, float64(8000000), result.Summary.TotalCashIn)
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

	t.Run("Error - Repository Fails", func(t *testing.T) {
		startDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC)

		repo := stubReportRepository{err: errors.New("db error")}
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
	t.Run("Success", func(t *testing.T) {
		mockData := &domain.ProductionReport{
			TargetMonth:     7,
			TargetYear:      2026,
			TotalQuota:      1000,
			TotalQtyOrdered: 800,
			RemainingQuota:  200,
		}

		repo := stubReportRepository{productionReport: mockData}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetProductionReport(context.Background(), 7, 2026)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1000, result.TotalQuota)
		assert.Equal(t, int64(800), result.TotalQtyOrdered)
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

	t.Run("Error - Repository Fails", func(t *testing.T) {
		repo := stubReportRepository{err: errors.New("db error")}
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
