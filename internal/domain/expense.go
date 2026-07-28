package domain

import (
	"context"
	"time"
)

type ExpenseCategoryType string

const (
	ExpenseTypeHPP  ExpenseCategoryType = "HPP"
	ExpenseTypeOPEX ExpenseCategoryType = "OPEX"
)

// ==========================================
// ENTITAS & INPUT: EXPENSE CATEGORY
// ==========================================
type ExpenseCategory struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Type        ExpenseCategoryType `json:"type"`
	Description string              `json:"description"`
}

type ExpenseCategoryCreateInput struct {
	Name        string
	Type        string
	Description string
}

// ==========================================
// ENTITAS & INPUT: EXPENSE TRANSACTION
// ==========================================
type Expense struct {
	ID                string    `json:"id"`
	ExpenseCategoryID string    `json:"expense_category_id"`
	CategoryName      string    `json:"category_name"`
	CategoryType      string    `json:"category_type"`
	BatchPoID         *string   `json:"batch_po_id"`
	BatchPoName       *string   `json:"batch_po_name"`
	Title             string    `json:"title"`
	Amount            float64   `json:"amount"`
	ExpenseDate       time.Time `json:"expense_date"`
	Notes             string    `json:"notes"`
	CreatedByID       string    `json:"created_by_id"`
	CreatorName       string    `json:"creator_name"`
	CreatedAt         time.Time `json:"created_at"`
}

type ExpenseCreateInput struct {
	ExpenseCategoryID string
	BatchPoID         *string
	Title             string
	Amount            float64
	ExpenseDate       time.Time
	Notes             string
	CreatedByID       string
}

// ==========================================
// KONTRAK REPOSITORY & USECASE
// ==========================================
type ExpenseRepository interface {
	// Kategori
	CreateCategory(ctx context.Context, category *ExpenseCategory) error
	FetchCategories(ctx context.Context) ([]ExpenseCategory, error)

	// Transaksi
	Create(ctx context.Context, expense *Expense) error
	GetByID(ctx context.Context, id string) (*Expense, error)
	Fetch(ctx context.Context, limit, offset int, poID *string, startDate, endDate *time.Time) ([]Expense, int64, error)
	Delete(ctx context.Context, id string) error
}

type ExpenseUsecase interface {
	// Kategori
	CreateCategory(ctx context.Context, input ExpenseCategoryCreateInput) (*ExpenseCategory, error)
	ListCategories(ctx context.Context) ([]ExpenseCategory, error)

	// Transaksi
	CreateExpense(ctx context.Context, input ExpenseCreateInput) (*Expense, error)
	GetExpense(ctx context.Context, id string) (*Expense, error)
	ListExpenses(ctx context.Context, query PaginationQuery, poID *string, startDate, endDate *string) ([]Expense, PaginationMeta, error)
	DeleteExpense(ctx context.Context, id string) error
}
