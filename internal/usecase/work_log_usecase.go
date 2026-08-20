package usecase

import (
	"context"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type workLogUsecase struct {
	workLogRepo    domain.WorkLogRepository
	workerRepo     domain.WorkerRepository
	contextTimeout time.Duration
}

func NewWorkLogUsecase(repo domain.WorkLogRepository, workerRepo domain.WorkerRepository, timeout time.Duration) domain.WorkLogUsecase {
	return &workLogUsecase{
		workLogRepo:    repo,
		workerRepo:     workerRepo,
		contextTimeout: timeout,
	}
}

func (u *workLogUsecase) CreateWorkLog(c context.Context, input domain.WorkLogCreateInput) (*domain.WorkLog, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if input.WorkerID == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Worker ID harus diisi")
	}
	if input.Qty <= 0 {
		return nil, domain.NewError(domain.ErrBadParamInput, "Qty harus lebih besar dari 0")
	}
	if input.RatePerQty <= 0 {
		return nil, domain.NewError(domain.ErrBadParamInput, "Tarif per pcs harus lebih besar dari 0")
	}
	if input.WorkDate.IsZero() {
		return nil, domain.NewError(domain.ErrBadParamInput, "Tanggal kerja tidak valid")
	}

	// Pastikan worker ada
	_, err := u.workerRepo.GetByID(ctx, input.WorkerID)
	if err != nil {
		return nil, err
	}

	totalAmount := float64(input.Qty) * input.RatePerQty

	log := &domain.WorkLog{
		WorkerID:    input.WorkerID,
		BatchPoID:   input.BatchPoID,
		JobType:     input.JobType,
		Qty:         input.Qty,
		RatePerQty:  input.RatePerQty,
		TotalAmount: totalAmount,
		WorkDate:    input.WorkDate,
		Notes:       input.Notes,
		CreatedByID: input.CreatedByID,
	}

	if err := u.workLogRepo.Create(ctx, log); err != nil {
		return nil, err
	}

	return log, nil
}

func (u *workLogUsecase) GetWorkLog(c context.Context, id string) (*domain.WorkLog, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "ID WorkLog tidak valid")
	}

	return u.workLogRepo.GetByID(ctx, id)
}

func (u *workLogUsecase) ListWorkLogs(c context.Context, query domain.PaginationQuery, filter domain.WorkLogFilter) ([]domain.WorkLog, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 || query.Limit > 100 {
		query.Limit = 10
	}
	offset := (query.Page - 1) * query.Limit

	logs, total, err := u.workLogRepo.Fetch(ctx, filter, query.Limit, offset)
	if err != nil {
		return nil, domain.PaginationMeta{}, err
	}

	meta := domain.PaginationMeta{
		CurrentPage: query.Page,
		Limit:       query.Limit,
		TotalItems:  total,
		TotalPages:  int(math.Ceil(float64(total) / float64(query.Limit))),
	}

	return logs, meta, nil
}

func (u *workLogUsecase) UpdateWorkLog(c context.Context, id string, input domain.WorkLogUpdateInput) (*domain.WorkLog, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "ID WorkLog tidak valid")
	}

	existing, err := u.workLogRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// WorkLog yang sudah masuk ke Payroll yang LUNAS tidak boleh diubah
	if existing.PayrollID != nil && *existing.PayrollID != "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Catatan borongan ini sudah masuk ke rekap penggajian dan tidak dapat diubah")
	}

	if input.WorkerID != "" {
		existing.WorkerID = input.WorkerID
	}
	if input.BatchPoID != nil {
		existing.BatchPoID = input.BatchPoID
	}
	if input.JobType != "" {
		existing.JobType = input.JobType
	}
	if input.Qty > 0 {
		existing.Qty = input.Qty
	}
	if input.RatePerQty > 0 {
		existing.RatePerQty = input.RatePerQty
	}
	if !input.WorkDate.IsZero() {
		existing.WorkDate = input.WorkDate
	}
	existing.Notes = input.Notes

	existing.TotalAmount = float64(existing.Qty) * existing.RatePerQty
	existing.UpdatedAt = time.Now()

	if err := u.workLogRepo.Update(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (u *workLogUsecase) DeleteWorkLog(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return domain.NewError(domain.ErrBadParamInput, "ID WorkLog tidak valid")
	}

	existing, err := u.workLogRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.PayrollID != nil && *existing.PayrollID != "" {
		return domain.NewError(domain.ErrBadParamInput, "Catatan borongan ini sudah masuk ke rekap penggajian dan tidak dapat dihapus")
	}

	return u.workLogRepo.Delete(ctx, id)
}
