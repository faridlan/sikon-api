package postgres

import (
	"context"
	"time"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type payrollRepository struct {
	db *gorm.DB
}

func NewPayrollRepository(db *gorm.DB) domain.PayrollRepository {
	return &payrollRepository{db: db}
}

func (r *payrollRepository) Create(ctx context.Context, payroll *domain.Payroll, workLogIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model := FromPayrollDomain(payroll)

		// 1. Create Payroll Record
		if err := tx.Create(model).Error; err != nil {
			return TranslateError(err)
		}

		// 2. Link work_logs ke payroll ini dan hitung total amount
		var totalAmount float64
		err := tx.Model(&WorkLogModel{}).
			Where("id IN ? AND payroll_id IS NULL", workLogIDs).
			Updates(map[string]any{"payroll_id": model.ID}).Error
		if err != nil {
			return TranslateError(err)
		}

		tx.Model(&WorkLogModel{}).
			Where("payroll_id = ?", model.ID).
			Select("COALESCE(SUM(total_amount), 0)").
			Scan(&totalAmount)

		// Update Total Amount Payroll
		if err := tx.Model(model).Update("total_amount", totalAmount).Error; err != nil {
			return TranslateError(err)
		}

		payroll.ID = model.ID
		payroll.TotalAmount = totalAmount
		payroll.CreatedAt = model.CreatedAt
		payroll.UpdatedAt = model.UpdatedAt
		return nil
	})
}

func (r *payrollRepository) GetByID(ctx context.Context, id string) (*domain.Payroll, error) {
	var model PayrollModel
	err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("Expense").
		Preload("WorkLogs.Worker").
		Preload("WorkLogs.BatchPO").
		Where("id = ?", id).
		First(&model).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	return model.ToDomain(), nil
}

func (r *payrollRepository) Fetch(ctx context.Context, filter domain.PayrollFilter, limit, offset int) ([]domain.Payroll, int64, error) {
	var models []PayrollModel
	var total int64

	query := r.db.WithContext(ctx).Model(&PayrollModel{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if filter.StartDate != nil && filter.EndDate != nil {
		query = query.Where("start_date >= ? AND end_date <= ?", filter.StartDate, filter.EndDate)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	err := query.
		Preload("Creator").
		Preload("Expense").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&models).Error

	if err != nil {
		return nil, 0, TranslateError(err)
	}

	var results []domain.Payroll
	for _, m := range models {
		results = append(results, *m.ToDomain())
	}

	return results, total, nil
}

func (r *payrollRepository) UpdateStatus(ctx context.Context, id string, status domain.PayrollStatus, expenseID *string, paidAt *time.Time) error {
	updates := map[string]interface{}{
		"status":     string(status),
		"updated_at": time.Now(),
	}

	if expenseID != nil {
		updates["expense_id"] = *expenseID
	}
	if paidAt != nil {
		updates["paid_at"] = *paidAt
	}

	result := r.db.WithContext(ctx).
		Model(&PayrollModel{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return TranslateError(result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *payrollRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Reset payroll_id pada work_logs yang terikat
		if err := tx.Model(&WorkLogModel{}).Where("payroll_id = ?", id).Update("payroll_id", nil).Error; err != nil {
			return TranslateError(err)
		}

		result := tx.Where("id = ?", id).Delete(&PayrollModel{})
		if result.Error != nil {
			return TranslateError(result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
