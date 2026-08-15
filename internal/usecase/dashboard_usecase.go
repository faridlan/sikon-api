package usecase

import (
	"context"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type dashboardUsecase struct {
	dashboardRepo  domain.DashboardRepository
	contextTimeout time.Duration
}

func NewDashboardUsecase(dr domain.DashboardRepository, timeout time.Duration) domain.DashboardUsecase {
	return &dashboardUsecase{
		dashboardRepo:  dr,
		contextTimeout: timeout,
	}
}

func (u *dashboardUsecase) GetSummary(c context.Context, filter domain.DashboardFilter) (*domain.DashboardSummary, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.dashboardRepo.GetSummary(ctx, filter)
}

// Tambahkan fungsi ini di bagian bawah file
func (u *dashboardUsecase) GetSalesReport(c context.Context, filter domain.DashboardFilter) ([]domain.SalesReportItem, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.dashboardRepo.GetSalesReport(ctx, filter)
}

func (u *dashboardUsecase) GetReceivablesReport(c context.Context, salesID string) ([]domain.ReceivableReportItem, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.dashboardRepo.GetReceivablesReport(ctx, salesID)
}
