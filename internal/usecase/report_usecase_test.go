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

type stubReportRepository struct {
	report *domain.ReportResponse
	err    error
}

func (s stubReportRepository) GetDailyReport(ctx context.Context, date string) (*domain.ReportResponse, error) {
	return s.report, s.err
}

func TestReportUsecase_GetDailyReport(t *testing.T) {
	t.Run("returns report when repository succeeds", func(t *testing.T) {
		repo := stubReportRepository{
			report: &domain.ReportResponse{
				DailySnapshot: domain.DailySnapshot{
					TotalRevenueToday: 150000,
					TotalQtyToday:     12,
					SalesPerformanceToday: []domain.SalesPerformanceToday{{
						SalesName:       "Rina",
						ProductCategory: "Kemeja",
						TotalQty:        12,
					}},
				},
			},
		}

		uc := usecase.NewReportUsecase(repo, 2*time.Second)

		result, err := uc.GetDailyReport(context.Background(), "2026-07-08")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, float64(150000), result.DailySnapshot.TotalRevenueToday)
		assert.Equal(t, int64(12), result.DailySnapshot.TotalQtyToday)
	})

	t.Run("uses today when no date is provided", func(t *testing.T) {
		repo := stubReportRepository{report: &domain.ReportResponse{}}
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
