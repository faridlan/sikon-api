package postgres

import (
	"context"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type expenseRepository struct {
	db *gorm.DB
}

func NewExpenseRepository(db *gorm.DB) domain.ExpenseRepository {
	return &expenseRepository{db: db}
}

// --- KATEGORI PENGELUARAN ---

func (r *expenseRepository) CreateCategory(ctx context.Context, cat *domain.ExpenseCategory) error {
	model := FromExpenseCategoryDomain(cat)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return TranslateError(err)
	}

	cat.ID = model.ID
	return nil
}

func (r *expenseRepository) FetchCategories(ctx context.Context) ([]domain.ExpenseCategory, error) {
	var models []ExpenseCategoryModel
	if err := r.db.WithContext(ctx).Order("name ASC").Find(&models).Error; err != nil {
		return nil, TranslateError(err)
	}

	var results []domain.ExpenseCategory
	for _, m := range models {
		results = append(results, *m.ToDomain())
	}
	return results, nil
}

// --- TRANSAKSI PENGELUARAN ---

func (r *expenseRepository) Create(ctx context.Context, exp *domain.Expense) error {
	model := FromExpenseDomain(exp)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return TranslateError(err)
	}

	exp.ID = model.ID
	exp.CreatedAt = model.CreatedAt
	return nil
}

func (r *expenseRepository) GetByID(ctx context.Context, id string) (*domain.Expense, error) {
	var model ExpenseModel
	err := r.db.WithContext(ctx).
		Preload("Category").
		Preload("BatchPO").
		Preload("Creator").
		Where("id = ?", id).
		First(&model).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	return model.ToDomain(), nil
}

func (r *expenseRepository) Fetch(ctx context.Context, limit, offset int, poID *string, startDate, endDate *time.Time) ([]domain.Expense, int64, error) {
	var models []ExpenseModel
	var total int64

	query := r.db.WithContext(ctx).Model(&ExpenseModel{})

	if poID != nil && *poID != "" {
		query = query.Where("batch_po_id = ?", *poID)
	}

	if startDate != nil && endDate != nil {
		query = query.Where("expense_date >= ? AND expense_date <= ?", startDate, endDate)
	}

	query.Count(&total)

	err := query.
		Preload("Category").
		Preload("BatchPO").
		Preload("Creator").
		Order("expense_date DESC, created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&models).Error

	if err != nil {
		return nil, 0, TranslateError(err)
	}

	var results []domain.Expense
	for _, m := range models {
		results = append(results, *m.ToDomain())
	}

	return results, total, nil
}

func (r *expenseRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&ExpenseModel{})
	if result.Error != nil {
		return TranslateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
