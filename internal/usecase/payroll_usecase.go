package usecase

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type payrollUsecase struct {
	payrollRepo    domain.PayrollRepository
	expenseRepo    domain.ExpenseRepository
	contextTimeout time.Duration
}

func NewPayrollUsecase(repo domain.PayrollRepository, expRepo domain.ExpenseRepository, timeout time.Duration) domain.PayrollUsecase {
	return &payrollUsecase{
		payrollRepo:    repo,
		expenseRepo:    expRepo,
		contextTimeout: timeout,
	}
}

func (u *payrollUsecase) CreatePayroll(c context.Context, input domain.PayrollCreateInput) (*domain.Payroll, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if input.StartDate.IsZero() || input.EndDate.IsZero() {
		return nil, domain.NewError(domain.ErrBadParamInput, "Rentang tanggal penggajian tidak valid")
	}

	if len(input.WorkLogIDs) == 0 {
		return nil, domain.NewError(domain.ErrBadParamInput, "Pilih minimal 1 catatan borongan (work log) untuk digaji")
	}

	now := time.Now()
	payrollNumber := fmt.Sprintf("PAY-%s-%03d", now.Format("200601"), now.Unix()%1000)

	payroll := &domain.Payroll{
		PayrollNumber: payrollNumber,
		StartDate:     input.StartDate,
		EndDate:       input.EndDate,
		Status:        domain.PayrollStatusDraft,
		CreatedByID:   input.CreatedByID,
	}

	if err := u.payrollRepo.Create(ctx, payroll, input.WorkLogIDs); err != nil {
		return nil, err
	}

	return u.payrollRepo.GetByID(ctx, payroll.ID)
}

func (u *payrollUsecase) GetPayroll(c context.Context, id string) (*domain.Payroll, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "ID Payroll tidak valid")
	}

	return u.payrollRepo.GetByID(ctx, id)
}

func (u *payrollUsecase) ListPayrolls(c context.Context, query domain.PaginationQuery, filter domain.PayrollFilter) ([]domain.Payroll, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 || query.Limit > 100 {
		query.Limit = 10
	}
	offset := (query.Page - 1) * query.Limit

	payrolls, total, err := u.payrollRepo.Fetch(ctx, filter, query.Limit, offset)
	if err != nil {
		return nil, domain.PaginationMeta{}, err
	}

	meta := domain.PaginationMeta{
		CurrentPage: query.Page,
		Limit:       query.Limit,
		TotalItems:  total,
		TotalPages:  int(math.Ceil(float64(total) / float64(query.Limit))),
	}

	return payrolls, meta, nil
}

func (u *payrollUsecase) ProcessPayrollPayment(c context.Context, id string, operatorID string) (*domain.Payroll, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "ID Payroll tidak valid")
	}

	payroll, err := u.payrollRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if payroll.Status == domain.PayrollStatusPaid {
		return nil, domain.NewError(domain.ErrBadParamInput, "Payroll ini sudah dibayar sebelumnya")
	}

	// 1. Ambil/Cari Kategori Expense "Gaji & Borongan"
	categories, err := u.expenseRepo.FetchCategories(ctx)
	if err != nil {
		return nil, err
	}

	var targetCategoryID string
	for _, cat := range categories {
		if cat.Type == domain.ExpenseTypeHPP {
			targetCategoryID = cat.ID
			break
		}
	}

	// Fallback jika belum ada kategori HPP
	if targetCategoryID == "" && len(categories) > 0 {
		targetCategoryID = categories[0].ID
	}

	// Tentukan Batch PO utama jika ada work logs yang diikat
	var primaryBatchPOID *string
	for _, wl := range payroll.WorkLogs {
		if wl.BatchPoID != nil && *wl.BatchPoID != "" {
			primaryBatchPOID = wl.BatchPoID
			break
		}
	}

	now := time.Now()
	// 2. Buat Record Expense Otomatis di Backend
	expense := &domain.Expense{
		ExpenseCategoryID: targetCategoryID,
		BatchPoID:         primaryBatchPOID,
		Title:             fmt.Sprintf("Pembayaran Rekap Gaji %s (%s - %s)", payroll.PayrollNumber, payroll.StartDate.Format("02/01"), payroll.EndDate.Format("02/01/2006")),
		Amount:            payroll.TotalAmount,
		ExpenseDate:       now,
		Notes:             fmt.Sprintf("Otomatis digenerate oleh sistem dari modul Payroll #%s", payroll.PayrollNumber),
		CreatedByID:       operatorID,
	}

	if err := u.expenseRepo.Create(ctx, expense); err != nil {
		return nil, err
	}

	// 3. Update Status Payroll menjadi PAID
	if err := u.payrollRepo.UpdateStatus(ctx, id, domain.PayrollStatusPaid, &expense.ID, &now); err != nil {
		return nil, err
	}

	return u.payrollRepo.GetByID(ctx, id)
}

func (u *payrollUsecase) DeletePayroll(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return domain.NewError(domain.ErrBadParamInput, "ID Payroll tidak valid")
	}

	payroll, err := u.payrollRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if payroll.Status == domain.PayrollStatusPaid {
		return domain.NewError(domain.ErrBadParamInput, "Payroll yang sudah dibayar tidak dapat dihapus")
	}

	return u.payrollRepo.Delete(ctx, id)
}
