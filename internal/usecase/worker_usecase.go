package usecase

import (
	"context"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type workerUsecase struct {
	workerRepo     domain.WorkerRepository
	contextTimeout time.Duration
}

func NewWorkerUsecase(repo domain.WorkerRepository, timeout time.Duration) domain.WorkerUsecase {
	return &workerUsecase{
		workerRepo:     repo,
		contextTimeout: timeout,
	}
}

func (u *workerUsecase) CreateWorker(c context.Context, input domain.WorkerCreateInput) (*domain.Worker, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if input.Name == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Nama pekerja tidak boleh kosong")
	}

	if input.Role != domain.WorkerRoleTailor &&
		input.Role != domain.WorkerRoleCutter &&
		input.Role != domain.WorkerRoleFinishing &&
		input.Role != domain.WorkerRoleSales &&
		input.Role != domain.WorkerRoleStaff &&
		input.Role != domain.WorkerRoleHelper {
		return nil, domain.NewError(domain.ErrBadParamInput, "Peran pekerja (role) tidak valid")
	}

	if input.SalaryType != domain.WorkerSalaryTypePieceRate &&
		input.SalaryType != domain.WorkerSalaryTypeDaily &&
		input.SalaryType != domain.WorkerSalaryTypeMonthly {
		return nil, domain.NewError(domain.ErrBadParamInput, "Tipe gaji pekerja (salary_type) tidak valid")
	}

	if input.SalaryType == domain.WorkerSalaryTypeDaily && input.DailyRate <= 0 {
		return nil, domain.NewError(domain.ErrBadParamInput, "Tarif gaji harian (daily_rate) harus lebih besar dari 0 untuk pekerja harian")
	}

	worker := &domain.Worker{
		UserID:     input.UserID,
		Name:       input.Name,
		Phone:      input.Phone,
		Role:       input.Role,
		SalaryType: input.SalaryType,
		DailyRate:  input.DailyRate,
		Status:     domain.WorkerStatusActive,
	}

	if err := u.workerRepo.Create(ctx, worker); err != nil {
		return nil, err
	}

	return worker, nil
}

func (u *workerUsecase) GetWorker(c context.Context, id string) (*domain.Worker, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "ID Pekerja tidak valid")
	}

	return u.workerRepo.GetByID(ctx, id)
}

func (u *workerUsecase) ListWorkers(c context.Context, query domain.PaginationQuery, filter domain.WorkerFilter) ([]domain.Worker, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 || query.Limit > 100 {
		query.Limit = 10
	}
	offset := (query.Page - 1) * query.Limit

	workers, total, err := u.workerRepo.Fetch(ctx, filter, query.Limit, offset)
	if err != nil {
		return nil, domain.PaginationMeta{}, err
	}

	meta := domain.PaginationMeta{
		CurrentPage: query.Page,
		Limit:       query.Limit,
		TotalItems:  total,
		TotalPages:  int(math.Ceil(float64(total) / float64(query.Limit))),
	}

	return workers, meta, nil
}

func (u *workerUsecase) UpdateWorker(c context.Context, id string, input domain.WorkerUpdateInput) (*domain.Worker, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "ID Pekerja tidak valid")
	}

	existing, err := u.workerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.UserID != nil {
		existing.UserID = input.UserID
	}
	if input.Name != "" {
		existing.Name = input.Name
	}
	if input.Phone != "" {
		existing.Phone = input.Phone
	}
	if input.Role != "" {
		existing.Role = input.Role
	}
	if input.SalaryType != "" {
		existing.SalaryType = input.SalaryType
	}
	if input.DailyRate >= 0 {
		existing.DailyRate = input.DailyRate
	}
	if input.Status != "" {
		existing.Status = input.Status
	}

	existing.UpdatedAt = time.Now()

	if err := u.workerRepo.Update(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (u *workerUsecase) DeleteWorker(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return domain.NewError(domain.ErrBadParamInput, "ID Pekerja tidak valid")
	}

	return u.workerRepo.Delete(ctx, id)
}
