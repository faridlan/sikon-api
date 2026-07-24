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
	revenue          float64
	qty              int64
	salesPerformance []domain.SalesPerformanceToday
	activePOs        []domain.ActivePO
	totalOutstanding float64
	activePO         float64
	pastDue          []domain.PastDueReceivable

	dailyReport *domain.DailyReport
	poSummary   *domain.POSummaryReport
	err         error
}

func (s stubReportRepository) GetDailyRevenueAndQty(_ context.Context, _, _ time.Time) (float64, int64, error) {
	return s.revenue, s.qty, s.err
}

func (s stubReportRepository) GetSalesPerformanceByActivePO(_ context.Context) ([]domain.SalesPerformanceToday, error) {
	return s.salesPerformance, s.err
}

func (s stubReportRepository) GetActivePOStats(_ context.Context) ([]domain.ActivePO, error) {
	return s.activePOs, s.err
}

func (s stubReportRepository) GetReceivablesStats(_ context.Context) (float64, float64, error) {
	return s.totalOutstanding, s.activePO, s.err
}

func (s stubReportRepository) GetPastDueReceivables(_ context.Context) ([]domain.PastDueReceivable, error) {
	return s.pastDue, s.err
}

func (s stubReportRepository) GetDailyReportData(ctx context.Context, targetDate time.Time) (*domain.DailyReport, error) {
	return s.dailyReport, s.err
}

func (s stubReportRepository) GetPOSummaryData(ctx context.Context, poID string) (*domain.POSummaryReport, error) {
	return s.poSummary, s.err
}

func TestReportUsecase_GetDailyReport(t *testing.T) {
	t.Run("returns assembled report when all repository calls succeed", func(t *testing.T) {
		repo := stubReportRepository{
			revenue: 9750000,
			qty:     26,
			salesPerformance: []domain.SalesPerformanceToday{
				{SalesName: "John Doe", ProductCategory: "Kemeja", TotalQty: 20},
			},
			activePOs: []domain.ActivePO{
				{
					BatchPOName:         "PO 1 JULI 2026",
					BatchPOID:           "2d673e63-9226-414b-817b-067751636527",
					TotalRevenueEntered: 9750000,
					TotalQtyReceived:    26,
					RemainingQuota:      274,
				},
			},
			totalOutstanding: 7000000,
			activePO:         7000000,
			pastDue:          []domain.PastDueReceivable{},
		}

		uc := usecase.NewReportUsecase(repo, 2*time.Second)
		result, err := uc.GetDailyReport(context.Background(), "2026-07-01")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, float64(9750000), result.DailySnapshot.TotalRevenueToday)
		assert.Equal(t, int64(26), result.DailySnapshot.TotalQtyToday)
		assert.Len(t, result.DailySnapshot.SalesPerformanceToday, 1)
		assert.Equal(t, "John Doe", result.DailySnapshot.SalesPerformanceToday[0].SalesName)
		assert.Len(t, result.ActivePOs, 1)
		assert.Equal(t, float64(7000000), result.TotalOutstandingReceivables)
		assert.Equal(t, float64(7000000), result.ActivePOReceivables)
		assert.Empty(t, result.PastDueReceivables)
	})

	t.Run("uses today when no date is provided", func(t *testing.T) {
		repo := stubReportRepository{
			salesPerformance: []domain.SalesPerformanceToday{},
			activePOs:        []domain.ActivePO{},
			pastDue:          []domain.PastDueReceivable{},
		}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetDailyReport(context.Background(), "")

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		repo := stubReportRepository{err: errors.New("db down")}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetDailyReport(context.Background(), "2026-07-08")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestReportUsecase_GenerateDailyReport(t *testing.T) {
	t.Run("returns assembled daily report when repository succeeds", func(t *testing.T) {
		mockReport := &domain.DailyReport{
			ReportDate: "2026-07-15",
			POInfo: domain.POInfo{
				POID:           "po-123",
				POName:         "PO 3 JULI 2026",
				Quota:          400,
				RemainingQuota: 360,
			},
			OrderSummary: domain.OrderSummary{
				QtyToday:   20,
				QtyTotalPO: 40,
				// Tambahkan mock data untuk TrendData di sini
				TrendData: []domain.DailyTrend{
					{Date: "2026-07-14", Qty: 20},
					{Date: "2026-07-15", Qty: 20},
				},
			},
			FinancialSummary: domain.FinancialSummary{
				TotalRevenue:          10000000, // Misal total omset 10 juta
				TotalPaid:             5000000,  // Sudah dibayar 5 juta
				ActivePOOutstanding:   5000000,  // Sisa 5 juta
				PreviousPOOutstanding: 15000000, // Sisa piutang PO lama 15 juta
				TotalOutstanding:      20000000, // Total piutang 20 juta
			},
		}

		repo := stubReportRepository{dailyReport: mockReport}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GenerateDailyReport(context.Background(), "2026-07-15")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "2026-07-15", result.ReportDate)
		assert.Equal(t, "PO 3 JULI 2026", result.POInfo.POName)
		assert.Equal(t, int64(20), result.OrderSummary.QtyToday)

		// Assertion untuk TrendData
		assert.Len(t, result.OrderSummary.TrendData, 2)
		assert.Equal(t, "2026-07-14", result.OrderSummary.TrendData[0].Date)
		assert.Equal(t, int64(20), result.OrderSummary.TrendData[0].Qty)

		// Assertion menyesuaikan skema FinancialSummary yang sudah di-update
		assert.Equal(t, float64(10000000), result.FinancialSummary.TotalRevenue)
		assert.Equal(t, float64(5000000), result.FinancialSummary.ActivePOOutstanding)
		assert.Equal(t, float64(20000000), result.FinancialSummary.TotalOutstanding)
	})

	t.Run("uses today when no date is provided", func(t *testing.T) {
		repo := stubReportRepository{
			dailyReport: &domain.DailyReport{ReportDate: time.Now().Format("2006-01-02")},
		}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GenerateDailyReport(context.Background(), "")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, time.Now().Format("2006-01-02"), result.ReportDate)
	})

	t.Run("returns error when date format is invalid", func(t *testing.T) {
		repo := stubReportRepository{}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GenerateDailyReport(context.Background(), "15-Juli-2026") // Format salah

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "format tanggal tidak valid, gunakan YYYY-MM-DD", err.Error())
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		repo := stubReportRepository{err: errors.New("db error")}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GenerateDailyReport(context.Background(), "2026-07-15")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
func TestReportUsecase_GetPOSummaryReport(t *testing.T) {
	t.Run("returns PO summary when repository succeeds", func(t *testing.T) {
		mockSummary := &domain.POSummaryReport{
			POID:            "po-123",
			POName:          "PO 3 JULI 2026",
			Status:          domain.BatchPOStatusActive,
			TotalQuota:      400,
			TotalQtyOrdered: 100,
			TotalRevenue:    15000000,
		}

		repo := stubReportRepository{poSummary: mockSummary}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetPOSummaryReport(context.Background(), "po-123")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "po-123", result.POID)
		assert.Equal(t, float64(15000000), result.TotalRevenue)
	})

	t.Run("returns error when poID is empty", func(t *testing.T) {
		repo := stubReportRepository{}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetPOSummaryReport(context.Background(), "")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "PO ID tidak boleh kosong", err.Error())
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		repo := stubReportRepository{err: errors.New("po not found")}
		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetPOSummaryReport(context.Background(), "po-invalid")

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
