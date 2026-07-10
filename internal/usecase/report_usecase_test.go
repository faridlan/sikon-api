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
	err              error
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
