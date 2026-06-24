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

	// Karena format tanggal dan kalkulasi murni sudah ditangani di repo,
	// usecase cukup memanggilnya secara langsung.
	return u.dashboardRepo.GetSummary(ctx, filter)
}
