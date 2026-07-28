package dto

import (
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
)

// ==========================================
// REQUEST DTO: KATEGORI PENGELUARAN
// ==========================================

type ExpenseCategoryCreateRequest struct {
	Name        string `json:"name" validate:"required" example:"Bahan Baku"`
	Type        string `json:"type" validate:"required,oneof=HPP OPEX" example:"HPP"`
	Description string `json:"description" validate:"omitempty" example:"Pembelian kain, resleting, dll"`
}

// ==========================================
// REQUEST DTO: TRANSAKSI PENGELUARAN
// ==========================================

type ExpenseCreateRequest struct {
	ExpenseCategoryID string    `json:"expense_category_id" validate:"required,uuid4" example:"123e4567-e89b-12d3-a456-426614174000"`
	BatchPoID         *string   `json:"batch_po_id" validate:"omitempty,uuid4" example:"123e4567-e89b-12d3-a456-426614174001"`
	CreatedByID       string    `json:"created_by_id" validate:"omitempty,uuid4" example:"123e4567-e89b-12d3-a456-426614174003"`
	Title             string    `json:"title" validate:"required" example:"Beli Kain Ripstop 1 Roll"`
	Amount            float64   `json:"amount" validate:"required,gt=0" example:"1500000"`
	ExpenseDate       time.Time `json:"expense_date" validate:"required" example:"2026-07-28T00:00:00Z"`
	Notes             string    `json:"notes" validate:"omitempty" example:"Beli di Toko Maju Jaya"`
}

// ==========================================
// RESPONSE DTO: KATEGORI PENGELUARAN
// ==========================================

type ExpenseCategoryResponse struct {
	ID          string `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Name        string `json:"name" example:"Bahan Baku"`
	Type        string `json:"type" example:"HPP"`
	Description string `json:"description" example:"Pembelian kain, resleting, dll"`
}

func ToExpenseCategoryResponse(cat *domain.ExpenseCategory) ExpenseCategoryResponse {
	return ExpenseCategoryResponse{
		ID:          cat.ID,
		Name:        cat.Name,
		Type:        string(cat.Type),
		Description: cat.Description,
	}
}

func ToExpenseCategoryResponseList(list []domain.ExpenseCategory) []ExpenseCategoryResponse {
	if len(list) == 0 {
		return []ExpenseCategoryResponse{}
	}

	responses := make([]ExpenseCategoryResponse, len(list))
	for i, c := range list {
		responses[i] = ToExpenseCategoryResponse(&c)
	}

	return responses
}

// ==========================================
// RESPONSE DTO: TRANSAKSI PENGELUARAN
// ==========================================

type ExpenseResponse struct {
	ID                string    `json:"id" example:"123e4567-e89b-12d3-a456-426614174002"`
	ExpenseCategoryID string    `json:"expense_category_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	CategoryName      string    `json:"category_name" example:"Bahan Baku"`
	CategoryType      string    `json:"category_type" example:"HPP"`
	BatchPoID         *string   `json:"batch_po_id" example:"123e4567-e89b-12d3-a456-426614174001"`
	BatchPoName       *string   `json:"batch_po_name" example:"PO Edisi Juli 2026"`
	Title             string    `json:"title" example:"Beli Kain Ripstop 1 Roll"`
	Amount            float64   `json:"amount" example:"1500000"`
	ExpenseDate       time.Time `json:"expense_date" example:"2026-07-28T00:00:00Z"`
	Notes             string    `json:"notes" example:"Beli di Toko Maju Jaya"`
	CreatedByID       string    `json:"created_by_id" example:"123e4567-e89b-12d3-a456-426614174003"`
	CreatorName       string    `json:"creator_name" example:"Admin Keuangan"`
	CreatedAt         time.Time `json:"created_at" example:"2026-07-28T10:16:10Z"`
}

func ToExpenseResponse(exp *domain.Expense) ExpenseResponse {
	return ExpenseResponse{
		ID:                exp.ID,
		ExpenseCategoryID: exp.ExpenseCategoryID,
		CategoryName:      exp.CategoryName,
		CategoryType:      exp.CategoryType,
		BatchPoID:         exp.BatchPoID,
		BatchPoName:       exp.BatchPoName,
		Title:             exp.Title,
		Amount:            exp.Amount,
		ExpenseDate:       exp.ExpenseDate,
		Notes:             exp.Notes,
		CreatedByID:       exp.CreatedByID,
		CreatorName:       exp.CreatorName,
		CreatedAt:         exp.CreatedAt,
	}
}

func ToExpenseResponseList(list []domain.Expense) []ExpenseResponse {
	if len(list) == 0 {
		return []ExpenseResponse{}
	}

	responses := make([]ExpenseResponse, len(list))
	for i, e := range list {
		responses[i] = ToExpenseResponse(&e)
	}

	return responses
}
