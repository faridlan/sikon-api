package usecase

import (
	"context"
	"math"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

type expenseUsecase struct {
	expenseRepo    domain.ExpenseRepository
	contextTimeout time.Duration
}

func NewExpenseUsecase(repo domain.ExpenseRepository, timeout time.Duration) domain.ExpenseUsecase {
	return &expenseUsecase{
		expenseRepo:    repo,
		contextTimeout: timeout,
	}
}

// ==========================================
// KATEGORI PENGELUARAN
// ==========================================

func (u *expenseUsecase) CreateCategory(c context.Context, input domain.ExpenseCategoryCreateInput) (*domain.ExpenseCategory, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if input.Name == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Nama kategori tidak boleh kosong")
	}

	inputType := domain.ExpenseCategoryType(input.Type)
	if inputType != domain.ExpenseTypeHPP && inputType != domain.ExpenseTypeOPEX {
		return nil, domain.NewError(domain.ErrBadParamInput, "Tipe kategori harus HPP atau OPEX")
	}

	cat := &domain.ExpenseCategory{
		Name:        input.Name,
		Type:        inputType,
		Description: input.Description,
	}

	if err := u.expenseRepo.CreateCategory(ctx, cat); err != nil {
		return nil, err
	}

	return cat, nil
}

func (u *expenseUsecase) ListCategories(c context.Context) ([]domain.ExpenseCategory, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	return u.expenseRepo.FetchCategories(ctx)
}

// ==========================================
// TRANSAKSI PENGELUARAN
// ==========================================

func (u *expenseUsecase) CreateExpense(c context.Context, input domain.ExpenseCreateInput) (*domain.Expense, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	// Validasi Bisnis
	if input.Title == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Judul pengeluaran tidak boleh kosong")
	}
	if input.Amount <= 0 {
		return nil, domain.NewError(domain.ErrBadParamInput, "Jumlah pengeluaran harus lebih besar dari 0")
	}
	if input.ExpenseCategoryID == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "Kategori pengeluaran harus diisi")
	}
	if input.ExpenseDate.IsZero() {
		return nil, domain.NewError(domain.ErrBadParamInput, "Tanggal pengeluaran tidak valid")
	}

	exp := &domain.Expense{
		ExpenseCategoryID: input.ExpenseCategoryID,
		BatchPoID:         input.BatchPoID,
		Title:             input.Title,
		Amount:            input.Amount,
		ExpenseDate:       input.ExpenseDate,
		Notes:             input.Notes,
		CreatedByID:       input.CreatedByID,
	}

	if err := u.expenseRepo.Create(ctx, exp); err != nil {
		return nil, err
	}

	return exp, nil
}

func (u *expenseUsecase) GetExpense(c context.Context, id string) (*domain.Expense, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return nil, domain.NewError(domain.ErrBadParamInput, "ID Pengeluaran tidak valid")
	}

	return u.expenseRepo.GetByID(ctx, id)
}

func (u *expenseUsecase) ListExpenses(c context.Context, query domain.PaginationQuery, poID *string, startDate, endDate *string) ([]domain.Expense, domain.PaginationMeta, error) {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 || query.Limit > 100 {
		query.Limit = 10
	}
	offset := (query.Page - 1) * query.Limit

	// Parsing Tanggal (Jika ada)
	var parsedStart, parsedEnd *time.Time
	if startDate != nil && *startDate != "" {
		t, err := time.Parse("2006-01-02", *startDate)
		if err == nil {
			parsedStart = &t
		}
	}
	if endDate != nil && *endDate != "" {
		t, err := time.Parse("2006-01-02", *endDate)
		if err == nil {
			// Set ke akhir hari (23:59:59)
			endOfDay := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
			parsedEnd = &endOfDay
		}
	}

	data, total, err := u.expenseRepo.Fetch(ctx, query.Limit, offset, poID, parsedStart, parsedEnd)
	if err != nil {
		return nil, domain.PaginationMeta{}, err
	}

	meta := domain.PaginationMeta{
		CurrentPage: query.Page,
		Limit:       query.Limit,
		TotalItems:  total,
		TotalPages:  int(math.Ceil(float64(total) / float64(query.Limit))),
	}

	return data, meta, nil
}

func (u *expenseUsecase) DeleteExpense(c context.Context, id string) error {
	ctx, cancel := context.WithTimeout(c, u.contextTimeout)
	defer cancel()

	if id == "" {
		return domain.NewError(domain.ErrBadParamInput, "ID tidak valid")
	}

	return u.expenseRepo.Delete(ctx, id)
}
