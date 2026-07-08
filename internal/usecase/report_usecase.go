package usecase

import (
	"context"
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

func (u *reportUsecase) GetDailyReport(c context.Context, date string) (*domain.ReportResponse, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	return u.reportRepo.GetDailyReport(ctx, date)
}
