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

func (u *reportUsecase) GetDailyReport(c context.Context, date string) (*domain.ReportResponse, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// 1. Resolve date: default to today if not provided.
	selectedDate, err := time.Parse("2006-01-02", date)
	if err != nil || date == "" {
		selectedDate = time.Now()
	}
	startOfDay := time.Date(selectedDate.Year(), selectedDate.Month(), selectedDate.Day(), 0, 0, 0, 0, time.Local)
	endOfDay := startOfDay.Add(24*time.Hour - time.Second)

	// 2. Fetch revenue & qty for the specific date.
	revenue, qty, err := u.reportRepo.GetDailyRevenueAndQty(ctx, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}

	// 3. Fetch sales performance grouped by active PO (ignores date filter).
	salesPerformance, err := u.reportRepo.GetSalesPerformanceByActivePO(ctx)
	if err != nil {
		return nil, err
	}

	// 4. Fetch active PO stats (accumulation of all orders in active POs, no date filter).
	activePOs, err := u.reportRepo.GetActivePOStats(ctx)
	if err != nil {
		return nil, err
	}

	// 5. Fetch receivables: total (all-time) and split by active vs. past-due PO.
	totalOutstanding, activePOReceivables, err := u.reportRepo.GetReceivablesStats(ctx)
	if err != nil {
		return nil, err
	}

	// 6. Fetch detailed past-due receivables (POs that are no longer active).
	pastDue, err := u.reportRepo.GetPastDueReceivables(ctx)
	if err != nil {
		return nil, err
	}

	// 7. Assemble and return the final response.
	return &domain.ReportResponse{
		DailySnapshot: domain.DailySnapshot{
			TotalRevenueToday:     revenue,
			TotalQtyToday:         qty,
			SalesPerformanceToday: salesPerformance,
		},
		ActivePOs:                   activePOs,
		TotalOutstandingReceivables: totalOutstanding,
		ActivePOReceivables:         activePOReceivables,
		PastDueReceivables:          pastDue,
	}, nil
}

func (u *reportUsecase) GenerateDailyReport(c context.Context, dateStr string) (*domain.DailyReport, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	targetDate := time.Now()
	var err error
	if dateStr != "" {
		targetDate, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, errors.New("format tanggal tidak valid, gunakan YYYY-MM-DD")
		}
	}

	report, err := u.reportRepo.GetDailyReportData(ctx, targetDate)
	if err != nil {
		return nil, err
	}

	return report, nil
}

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
