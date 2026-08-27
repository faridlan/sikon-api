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

// 🚨 BATCH INSERT: Untuk Auto Distribute Load
func (r *workLogRepository) CreateBatch(ctx context.Context, logs []domain.WorkLog) error {
	var models []WorkLogModel
	for _, l := range logs {
		models = append(models, *FromWorkLogDomain(&l))
	}

	if err := r.db.WithContext(ctx).Create(&models).Error; err != nil {
		return TranslateError(err)
	}

	return nil
}

func (r *workLogRepository) GetByID(ctx context.Context, id string) (*domain.WorkLog, error) {
	var model WorkLogModel
	err := r.db.WithContext(ctx).
		Preload("Worker").
		Preload("BatchPO").
		Preload("Order"). // 👈 Preload Order
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

	if filter.OrderID != nil && *filter.OrderID != "" {
		query = query.Where("order_id = ?", *filter.OrderID)
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
		Preload("Order").          // 👈 Preload Order Utama
		Preload("Order.Customer"). // 👈 NESTED PRELOAD: Ambil Customer di dalam Order
		Preload("Order.Sales").    // 👈 NESTED PRELOAD: Ambil Sales di dalam Order
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
		Updates(map[string]any{
			"worker_id":    model.WorkerID,
			"batch_po_id":  model.BatchPoID,
			"order_id":     model.OrderID, // 👈 Update OrderID
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

// 🚨 HELPER 1: Guard Akumulasi Qty per Order Konsumen & JobType
func (r *workLogRepository) GetTotalQtyByOrderAndJobType(ctx context.Context, orderID string, jobType domain.JobType, excludeLogID string) (int, error) {
	var totalQty int
	query := r.db.WithContext(ctx).Model(&WorkLogModel{}).
		Where("order_id = ? AND job_type = ? AND deleted_at IS NULL", orderID, jobType)

	if excludeLogID != "" {
		query = query.Where("id != ?", excludeLogID)
	}

	err := query.Select("COALESCE(SUM(qty), 0)").Scan(&totalQty).Error
	if err != nil {
		return 0, TranslateError(err)
	}

	return totalQty, nil
}

// 🚨 HELPER 2: Total Biaya Borongan per Order Spesifik
func (r *workLogRepository) GetTotalCostByOrder(ctx context.Context, orderID string) (float64, error) {
	var totalCost float64
	err := r.db.WithContext(ctx).Model(&WorkLogModel{}).
		Where("order_id = ? AND deleted_at IS NULL", orderID).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&totalCost).Error

	if err != nil {
		return 0, TranslateError(err)
	}

	return totalCost, nil
}

// 🚨 HELPER 3: Total Biaya Borongan Seluruh PO
func (r *workLogRepository) GetTotalCostByBatchPO(ctx context.Context, batchPoID string) (float64, error) {
	var totalCost float64
	err := r.db.WithContext(ctx).Model(&WorkLogModel{}).
		Where("batch_po_id = ? AND deleted_at IS NULL", batchPoID).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&totalCost).Error

	if err != nil {
		return 0, TranslateError(err)
	}

	return totalCost, nil
}
