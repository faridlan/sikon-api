package postgres

import (
	"context"

	"github.com/faridlan/sikon-api/internal/domain"
	"gorm.io/gorm"
)

type workLogRepository struct {
	db *gorm.DB
}

func NewWorkLogRepository(db *gorm.DB) domain.WorkLogRepository {
	return &workLogRepository{db: db}
}

func (r *workLogRepository) Create(ctx context.Context, log *domain.WorkLog) error {
	model := FromWorkLogDomain(log)

	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return TranslateError(err)
	}

	log.ID = model.ID
	log.CreatedAt = model.CreatedAt
	log.UpdatedAt = model.UpdatedAt
	return nil
}

func (r *workLogRepository) GetByID(ctx context.Context, id string) (*domain.WorkLog, error) {
	var model WorkLogModel
	err := r.db.WithContext(ctx).
		Preload("Worker").
		Preload("BatchPO").
		Preload("Creator").
		Where("id = ?", id).
		First(&model).Error

	if err != nil {
		return nil, TranslateError(err)
	}

	return model.ToDomain(), nil
}

func (r *workLogRepository) Fetch(ctx context.Context, filter domain.WorkLogFilter, limit, offset int) ([]domain.WorkLog, int64, error) {
	var models []WorkLogModel
	var total int64

	query := r.db.WithContext(ctx).Model(&WorkLogModel{})

	if filter.WorkerID != nil && *filter.WorkerID != "" {
		query = query.Where("worker_id = ?", *filter.WorkerID)
	}

	if filter.BatchPoID != nil && *filter.BatchPoID != "" {
		query = query.Where("batch_po_id = ?", *filter.BatchPoID)
	}

	if filter.PayrollID != nil && *filter.PayrollID != "" {
		query = query.Where("payroll_id = ?", *filter.PayrollID)
	}

	if filter.IsUnpaid {
		query = query.Where("payroll_id IS NULL")
	}

	if filter.JobType != "" {
		query = query.Where("job_type = ?", filter.JobType)
	}

	if filter.StartDate != nil && filter.EndDate != nil {
		query = query.Where("work_date >= ? AND work_date <= ?", filter.StartDate, filter.EndDate)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, TranslateError(err)
	}

	err := query.
		Preload("Worker").
		Preload("BatchPO").
		Preload("Creator").
		Order("work_date DESC, created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&models).Error

	if err != nil {
		return nil, 0, TranslateError(err)
	}

	var results []domain.WorkLog
	for _, m := range models {
		results = append(results, *m.ToDomain())
	}

	return results, total, nil
}

func (r *workLogRepository) Update(ctx context.Context, log *domain.WorkLog) error {
	model := FromWorkLogDomain(log)

	result := r.db.WithContext(ctx).
		Model(&WorkLogModel{}).
		Where("id = ?", log.ID).
		Updates(map[string]interface{}{
			"worker_id":    model.WorkerID,
			"batch_po_id":  model.BatchPoID,
			"job_type":     model.JobType,
			"qty":          model.Qty,
			"rate_per_qty": model.RatePerQty,
			"total_amount": model.TotalAmount,
			"work_date":    model.WorkDate,
			"notes":        model.Notes,
			"updated_at":   model.UpdatedAt,
		})

	if result.Error != nil {
		return TranslateError(result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *workLogRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&WorkLogModel{})
	if result.Error != nil {
		return TranslateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
